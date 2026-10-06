package generator

import (
	"bufio"
	"fmt"
	"github.com/giornetta/gopapageno"
	"io"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
)

var (
	axiomRegexp = regexp.MustCompile("^%axiom\\s*([a-zA-Z][a-zA-Z0-9]*)\\s*$")
)

const (
	emptyToken = "__EMPTY__"
	termToken  = "__TERM__"
)

type grammarDescription struct {
	axiom        string
	preambleFunc string

	rules []ruleDescription

	code string

	// nonterminals is nil until inferTokens() is executed successfully.
	nonterminals *set[string]

	// terminals is nil until inferTokens() is executed successfully.
	terminals *set[string]

	precMatrix precedenceMatrix
}

type ruleDescription struct {
	LHS      string
	RHS      []string
	Action   string
	Flags    gopapageno.RuleFlags
	Prefixes [][]string
}

func (r ruleDescription) String() string {
	return fmt.Sprintf("%s -> %s", r.LHS, strings.Join(r.RHS, " "))
}

func parseGrammarDescription(r io.Reader, opts *Options) (*grammarDescription, error) {
	opts.Logger.Printf("Parsing parser description file...\n")

	scanner := bufio.NewScanner(r)

	// Parse axiom definition and other options
	var axiom string
	moreThanOneAxiomWarning := false

	var preambleFunc string

	for scanner.Scan() {
		l := scanner.Text()
		if separatorRegexp.MatchString(l) {
			break
		}

		if match := axiomRegexp.FindStringSubmatch(l); match != nil {
			if axiom != "" && !moreThanOneAxiomWarning {
				log.Println("Warning: axiom is defined more than once.")
				moreThanOneAxiomWarning = true
			}
			axiom = match[1]
		} else if match := preambleRegex.FindStringSubmatch(l); match != nil {
			preambleFunc = match[1]
		} else if l != "" {
			return nil, fmt.Errorf("unrecognized parser option: %s", l)
		}
	}

	if axiom == "" {
		return nil, fmt.Errorf("no axiom is defined")
	}

	opts.Logger.Printf("Axiom: %s\n", axiom)

	// Parse rules
	var sb strings.Builder
	for scanner.Scan() {
		l := scanner.Bytes()
		if separatorRegexp.Match(l) {
			break
		} else if commentRegexp.Match(l) {
			continue
		}

		sb.Write(l)
		sb.WriteString("\n")
	}

	rules, err := parseRules(sb.String(), opts.Strategy)
	if err != nil {
		return nil, fmt.Errorf("could not parse rules: %w", err)
	}

	opts.Logger.Printf("\n--- Grammar Rules:\n")
	for _, rule := range rules {
		opts.Logger.Printf("%s\n", rule)
	}
	opts.Logger.Printf("\n")

	// Parse code
	var preambleBuilder strings.Builder
	for scanner.Scan() {
		l := scanner.Bytes()

		preambleBuilder.Write(l)
		preambleBuilder.WriteString("\n")
	}

	return &grammarDescription{
		axiom:        axiom,
		preambleFunc: preambleFunc,
		code:         preambleBuilder.String(),
		rules:        rules,
	}, nil
}

// parseRules parses rules specified in the input string and returns a slice of rules.
func parseRules(input string, strategy gopapageno.ParsingStrategy) ([]ruleDescription, error) {
	rules := make([]ruleDescription, 0)

	var pos int
	skipSpaces(input, &pos)

	var lhs string
	for pos < len(input) {
		var r ruleDescription
		r.Flags = gopapageno.RuleSimple

		// If we're reading an "alternate rule"
		if lhs == "" {
			// Read LHS
			lhs = getIdentifier(input, &pos)
			if lhs == "" {
				return nil, fmt.Errorf("missing or invalid identifier for lhs")
			}
			skipSpaces(input, &pos)

			// Read production delimiter
			if input[pos] == ':' {
				pos++
			} else {
				return nil, fmt.Errorf("rule %s is missing a colon between lhs and rhs", lhs)
			}
			skipSpaces(input, &pos)
		}
		r.LHS = lhs

		// Read Rhs
		r.RHS = make([]string, 0)
		r.Prefixes = make([][]string, 0)

		tokensAfterPrefix := 0

		for input[pos] != '{' {
			var rhsToken string
			if strategy != gopapageno.COPP {
				rhsToken = getIdentifier(input, &pos)
				if rhsToken == "" {
					return nil, fmt.Errorf("rule %s is missing an identifier for rhs", lhs)
				}

				r.RHS = append(r.RHS, rhsToken)
			} else {
				if input[pos] == '(' {
					// If the next section is a ()+ part, get the list of all produced alternatives (even nested).
					flattened, alternatives, err := getAlternatives(input, &pos)
					if err != nil {
						return nil, fmt.Errorf("rule %s is missing an alternative body for lhs", lhs)
					}

					r.RHS = append(r.RHS, flattened...)

					tokensAfterPrefix = 0

					if len(r.Prefixes) == 0 {
						r.Prefixes = make([][]string, 1)
					}
					// Add each produced alternative to every rhs found so far.
					newPrefixes := make([][]string, len(r.Prefixes)*len(alternatives))
					for i := 0; i < len(r.Prefixes); i++ {
						for j := 0; j < len(alternatives); j++ {
							newPrefixes[i*len(alternatives)+j] = append(r.Prefixes[i], alternatives[j]...)
						}
					}
					r.Prefixes = newPrefixes
					r.Flags = gopapageno.RuleCyclic
				} else {
					// Get a simple identifier
					rhsToken = getIdentifier(input, &pos)
					if rhsToken == "" {
						return nil, fmt.Errorf("r %s is missing an identifier for rhs", lhs)
					}
					tokensAfterPrefix++

					r.RHS = append(r.RHS, rhsToken)

					for i := 0; i < len(r.Prefixes); i++ {
						r.Prefixes[i] = append(r.Prefixes[i], rhsToken)
					}
				}
			}

			skipSpaces(input, &pos)
		}

		semFun := getSemanticFunction(input, &pos)
		r.Action = semFun

		if strategy != gopapageno.COPP {
			r.Flags = gopapageno.RuleSimple
			rules = append(rules, r)
		} else {
			if len(r.Prefixes) >= 1 {
				fullPrefixes := r.Prefixes

				newPrefixes := make([][]string, len(r.Prefixes))
				for i, prefix := range r.Prefixes {
					newPrefixes[i] = prefix[:len(prefix)-tokensAfterPrefix]
				}
				r.Prefixes = newPrefixes

				rules = append(rules, r)

				r.Flags = gopapageno.RulePrefix
				r.Prefixes = fullPrefixes

				for _, p := range r.Prefixes {
					r.RHS = p
					rules = append(rules, r)
				}
			} else {
				r.Prefixes = r.Prefixes[:0]
				rules = append(rules, r)
			}
		}

		skipSpaces(input, &pos)

		if input[pos] == ';' {
			// We're done with rules with this lhs
			// Reset current lhs
			lhs = ""
		} else if input[pos] == '|' {
			// We have another r with the same lhs
		} else {
			return nil, fmt.Errorf("invalid character at the end of r %s", lhs)
		}

		pos++
		skipSpaces(input, &pos)
	}

	return rules, nil
}

// compile completes the parser description by doing all necessary checks and
// transformations in order to produce a correct OPG.
func (p *grammarDescription) compile(opts *Options) error {
	p.inferTokens()

	if !p.isAxiomUsed() {
		return fmt.Errorf("axiom isn't used in any ruleDescription")
	}

	p.deleteRepeatedRHS()

	var precMatrix precedenceMatrix
	var err error

	precMatrix, err = p.newPrecedenceMatrix(opts)
	if err != nil {
		return fmt.Errorf("could not create precedence matrix: %w", err)
	}
	p.precMatrix = precMatrix

	p.removePrefixRules()

	p.sortRulesByRHS()

	opts.Logger.Printf("\n--- Updated Grammar Rules:\n")
	for _, rule := range p.rules {
		opts.Logger.Printf("%s\n", rule)
	}
	opts.Logger.Printf("\n")

	return nil
}

func (p *grammarDescription) removePrefixRules() {
	rules := make([]ruleDescription, 0)

	for _, r := range p.rules {
		if r.Flags == gopapageno.RulePrefix {
			continue
		}

		if r.LHS == r.RHS[0] && r.LHS == p.axiom {
			continue
		}

		// E -> T + T
		rules = append(rules, r)
	}

	p.rules = rules
}

// inferTokens populates the two sets nonterminals and terminals
// with tokens found in the grammar rules.
func (p *grammarDescription) inferTokens() {
	p.nonterminals = newSet[string]()
	tokens := newSet[string]()

	for _, rule := range p.rules {
		if !p.nonterminals.Contains(rule.LHS) {
			p.nonterminals.Add(rule.LHS)
		}
		for _, token := range rule.RHS {
			if !tokens.Contains(token) {
				tokens.Add(token)
			}
		}
	}

	p.terminals = tokens.Difference(p.nonterminals)

	p.nonterminals.Add(emptyToken)
	p.terminals.Add(termToken)
}

// isAxiomUsed checks if the axiom is present in any rules' LHS.
func (p *grammarDescription) isAxiomUsed() bool {
	for _, rule := range p.rules {
		if rule.LHS == p.axiom {
			return true
		}
	}

	return false
}

func (p *grammarDescription) emit(opts *Options, packageName string) error {
	pPath := path.Join(opts.OutputDirectory, GeneratedParserFilename)
	opts.Logger.Printf("Creating parser file %s...\n", pPath)

	f, err := os.Create(pPath)
	if err != nil {
		return fmt.Errorf("could not create parser file %s: %w", pPath, err)
	}

	/************
	 * Preamble *
	 ************/
	fmt.Fprintf(f, "// Code generated by Gopapageno; DO NOT EDIT.\n")
	fmt.Fprintf(f, "package %s\n\n", packageName)
	fmt.Fprintf(f, `import (
	"github.com/giornetta/gopapageno"
	"strings"
	"fmt"
	"os"
)
`)

	/********
	 * Code *
	 ********/
	fmt.Fprint(f, p.code)
	fmt.Fprintf(f, "\n\n")

	/**********
	 * Tokens *
	 **********/
	p.emitTokens(f)

	// NewParser func starts here.
	fmt.Fprintf(f, "\nfunc NewGrammar() *gopapageno.Grammar {\n")

	/*****************
	 * Token Numbers *
	 *****************/
	fmt.Fprintf(f, "\tnumTerminals := uint16(%d)\n", p.terminals.Len())
	fmt.Fprintf(f, "\tnumNonTerminals := uint16(%d)\n\n", p.nonterminals.Len())

	/*********
	 * Rules *
	 *********/
	maxRHSLen := 0
	for _, rule := range p.rules {
		if len(rule.RHS) > maxRHSLen {
			maxRHSLen = len(rule.RHS)
		}
	}

	fmt.Fprintf(f, "\tmaxRHSLen := %d\n", maxRHSLen)
	fmt.Fprint(f, "\trules := []gopapageno.Rule{\n")
	for _, rule := range p.rules {
		fmt.Fprintf(f, "\t\t{%s, []gopapageno.TokenType{%s}, gopapageno.%s},\n", rule.LHS, strings.Join(rule.RHS, ", "), rule.Flags)
	}
	fmt.Fprintf(f, "\t}\n")

	trie, err := newRulesTrie(p.rules, p.nonterminals, p.terminals)
	if err != nil {
		// TODO: Change this handling.
		panic(err)
	}

	compressedTrie := trie.Compress(p.nonterminals, p.terminals)

	fmt.Fprintf(f, "\tcompressedRules := []uint16{")
	if len(compressedTrie) > 0 {
		fmt.Fprintf(f, "%d", compressedTrie[0])
		for i := 1; i < len(compressedTrie); i++ {
			fmt.Fprintf(f, ", %d", compressedTrie[i])
		}
	}
	fmt.Fprintf(f, "\t}\n\n")

	///*****************
	// * COPP Prefixes *
	// *****************/
	if opts.Strategy == gopapageno.COPP {
		maxPrefixLen := 0
		for _, rule := range p.rules {
			for _, prefix := range rule.Prefixes {
				if len(prefix) > maxPrefixLen {
					maxPrefixLen = len(prefix)
				}
			}
		}

		fmt.Fprintf(f, "\tmaxPrefixLength := %d\n", maxPrefixLen)
		fmt.Fprint(f, "\tprefixes := [][]gopapageno.TokenType{\n")
		for _, rule := range p.rules {
			for _, prefix := range rule.Prefixes {
				fmt.Fprintf(f, "\t\t{%s},\n", strings.Join(prefix, ", "))
			}
		}
		fmt.Fprintf(f, "\t}\n")

		prefixTrie, err := newPrefixesTrie(p.rules, p.nonterminals, p.terminals)
		if err != nil {
			// TODO: Change this handling.
			panic(err)
		}

		compressedPrefixTrie := prefixTrie.Compress(p.nonterminals, p.terminals)

		fmt.Fprintf(f, "\tcompressedPrefixes := []uint16{")
		if len(compressedPrefixTrie) > 0 {
			fmt.Fprintf(f, "%d", compressedPrefixTrie[0])
			for i := 1; i < len(compressedPrefixTrie); i++ {
				fmt.Fprintf(f, ", %d", compressedPrefixTrie[i])
			}
		}
		fmt.Fprintf(f, "\t}\n\n")
	}

	/*********************
	 * Precedence Matrix *
	 *********************/
	fmt.Fprintf(f, "\tprecMatrix := [][]gopapageno.Precedence{\n")
	for i, _ := range p.precMatrix {
		fmt.Fprintf(f, "\t\t{")

		for j, _ := range p.precMatrix {
			if j < len(p.precMatrix)-1 {
				fmt.Fprintf(f, "gopapageno.Prec%s, ", p.precMatrix[i][j].String())
			} else {
				fmt.Fprintf(f, "gopapageno.Prec%s", p.precMatrix[i][j].String())
			}
		}
		fmt.Fprintf(f, "},\n")
	}
	fmt.Fprintf(f, "\t}\n")

	bitPackedMatrix := bitPack(p.precMatrix)
	fmt.Fprintf(f, "\tbitPackedMatrix := []uint64{\n\t\t")
	for _, v := range bitPackedMatrix {
		fmt.Fprintf(f, "%d, ", v)
	}
	fmt.Fprintf(f, "\n\t}\n\n")

	/*******************
	 * Grammar Function *
	 *******************/
	p.emitParserFunctions(f)

	/********************
	 * Construct Grammar *
	 ********************/
	fmt.Fprintf(f, "\treturn &gopapageno.Grammar{\n")
	fmt.Fprintf(f, "\t\tNumTerminals:  numTerminals,\n")
	fmt.Fprintf(f, "\t\tNumNonterminals: numNonTerminals,\n")
	fmt.Fprintf(f, "\t\tMaxRHSLength: maxRHSLen,\n")
	fmt.Fprintf(f, "\t\tRules: rules,\n")
	fmt.Fprintf(f, "\t\tCompressedRules: compressedRules,\n")
	fmt.Fprintf(f, "\t\tPrecedenceMatrix: precMatrix,\n")
	fmt.Fprintf(f, "\t\tBitPackedPrecedenceMatrix: bitPackedMatrix,\n")
	if opts.Strategy == gopapageno.COPP {
		fmt.Fprintf(f, "\t\tMaxPrefixLength: maxPrefixLength,\n")
		fmt.Fprintf(f, "\t\tPrefixes: prefixes,\n")
		fmt.Fprintf(f, "\t\tCompressedPrefixes: compressedPrefixes,\n")
	}
	fmt.Fprintf(f, "\t\tFunc: fn,\n")
	fmt.Fprintf(f, "\t\tParsingStrategy: gopapageno.%s,\n", opts.Strategy)

	if p.preambleFunc != "" {
		fmt.Fprintf(f, "\t\tPreambleFunc: %s,\n", p.preambleFunc)
	}

	fmt.Fprintf(f, "\t}\n}\n\n")

	return nil
}

func (p *grammarDescription) emitParserFunctions(f io.Writer) {
	fmt.Fprintf(f, "\tfn := func(ruleDescription uint16, ruleFlags gopapageno.RuleFlags, lhs *gopapageno.Token, rhs []*gopapageno.Token, thread int){\n")
	fmt.Fprintf(f, "\t\tswitch ruleDescription {\n")
	for i, rule := range p.rules {
		if len(rule.RHS) == 0 || rule.Flags.Has(gopapageno.RulePrefix) {
			continue
		}

		fmt.Fprintf(f, "\t\tcase %d:\n", i)
		fmt.Fprintf(f, "\t\t\t%s0 := lhs\n", rule.LHS)
		for j, _ := range rule.RHS {
			fmt.Fprintf(f, "\t\t\t%s%d := rhs[%d]\n", rule.RHS[j], j+1, j)
		}
		fmt.Fprintf(f, "\n")

		switch rule.Flags {
		case gopapageno.RuleSimple:
			fmt.Fprintf(f, "\t\t\t%s0.Child = %s1\n", rule.LHS, rule.RHS[0])
			for j := 0; j < len(rule.RHS)-1; j++ {
				fmt.Fprintf(f, "\t\t\t%s%d.Next = %s%d\n", rule.RHS[j], j+1, rule.RHS[j+1], j+2)
			}
			fmt.Fprintf(f, "\t\t\t%s0.LastChild = %s%d\n", rule.LHS, rule.RHS[len(rule.RHS)-1], len(rule.RHS))
		case gopapageno.RuleCyclic:
			fmt.Fprintf(f, "\t\t\tif ruleFlags.Has(gopapageno.RuleAppend) {\n")
			fmt.Fprintf(f, "\t\t\t\t%s0.LastChild.Next = %s2\n", rule.LHS, rule.RHS[1])
			fmt.Fprintf(f, "\t\t\t} else {\n")
			fmt.Fprintf(f, "\t\t\t\t%s0.Child = %s1\n", rule.LHS, rule.RHS[0])
			fmt.Fprintf(f, "\t\t\t\t%s1.Next = %s%d\n", rule.RHS[0], rule.RHS[1], 2)
			fmt.Fprintf(f, "\t\t\t}\n\n")

			for j := 1; j < len(rule.RHS)-1; j++ {
				fmt.Fprintf(f, "\t\t\t%s%d.Next = %s%d\n", rule.RHS[j], j+1, rule.RHS[j+1], j+2)
			}

			fmt.Fprintf(f, "\n\t\t\tif ruleFlags.Has(gopapageno.RuleCombine) {\n")
			fmt.Fprintf(f, "\t\t\t\t%s%d.Next = %s%d.Child\n",
				rule.RHS[len(rule.RHS)-2], len(rule.RHS)-2+1,
				rule.RHS[len(rule.RHS)-1], len(rule.RHS)-1+1)
			fmt.Fprintf(f, "\t\t\t\t%s0.LastChild = %s%d.LastChild\n", rule.LHS, rule.RHS[len(rule.RHS)-1], len(rule.RHS))
			fmt.Fprintf(f, "\t\t\t} else {\n")
			fmt.Fprintf(f, "\t\t\t\t%s%d.Next = %s%d\n",
				rule.RHS[len(rule.RHS)-2], len(rule.RHS)-2+1,
				rule.RHS[len(rule.RHS)-1], len(rule.RHS)-1+1)
			fmt.Fprintf(f, "\t\t\t\t%s0.LastChild = %s%d\n", rule.LHS, rule.RHS[len(rule.RHS)-1], len(rule.RHS))
			fmt.Fprintf(f, "\t\t\t}\n")
		default:
			continue
		}
		fmt.Fprintf(f, "\n")

		action := rule.Action
		action = strings.Replace(action, "$$", rule.LHS+"0", -1)
		for j, _ := range rule.RHS {
			action = strings.Replace(action, fmt.Sprintf("$%d", j+1), fmt.Sprintf("%s%d", rule.RHS[j], j+1), -1)
		}
		lines := strings.Split(action, "\n")
		for _, line := range lines {
			fmt.Fprintf(f, "\t\t\t")
			fmt.Fprint(f, line)
			fmt.Fprintf(f, "\n")
		}

		for j, _ := range rule.RHS {
			fmt.Fprintf(f, "\t\t\t_ = %s%d\n", rule.RHS[j], j+1)
		}
	}
	fmt.Fprintf(f, "\t\t}\n")
	fmt.Fprintf(f, "\t\t_ = ruleFlags\n")
	fmt.Fprintf(f, "\t}\n\n")
}

func (p *grammarDescription) emitTokens(f io.Writer) {
	fmt.Fprintf(f, "// Non-terminals\n")
	fmt.Fprintf(f, "const (\n")
	for i, token := range p.nonterminals.Slice() {
		if token == emptyToken {
			continue
		}

		if i == 0 {
			fmt.Fprintf(f, "\t%s = gopapageno.TokenEmpty + 1 + iota\n", token)
		} else {
			fmt.Fprintf(f, "\t%s\n", token)
		}
	}
	fmt.Fprintf(f, ")\n\n")

	fmt.Fprintf(f, "// Terminals\n")
	fmt.Fprintf(f, "const (\n")
	for i, token := range p.terminals.Slice() {
		if token == termToken {
			continue
		}

		if i == 0 {
			fmt.Fprintf(f, "\t%s = gopapageno.TokenTerm + 1 + iota\n", token)
		} else {
			fmt.Fprintf(f, "\t%s\n", token)
		}
	}
	fmt.Fprintf(f, ")\n\n")


	fmt.Fprintf(f, `func DumpGraph[ValueType any](root *gopapageno.Token, f *os.File) {
	sb := strings.Builder{}
	sb.WriteString("digraph parse_tree {\n")
	sb.WriteString("ratio = fill;\n")
	sb.WriteString("node [style=filled];\n")

	var graphPrintRec func(t *gopapageno.Token, p *gopapageno.Token, sb *strings.Builder, i int)
	graphPrintRec = func(t *gopapageno.Token, p *gopapageno.Token, sb *strings.Builder, i int) {
		if t == nil {
			return
		}

		if p == nil {
			graphPrintRec(t.Child, t, sb, i+1)
			return
		}

		var t_name, t_color, p_name, p_color string
`)

	termColor := "0.641 0.212 1.000"
	nonTermColor := "0.408 0.498 1.000"

	fmt.Fprintf(f, "\n\t\tswitch p.Type {\n")
	for _, token := range p.nonterminals.Slice() {
		if token == emptyToken {
			fmt.Fprintf(f, "\t\tcase gopapageno.TokenEmpty:\n\t\t\tp_name, p_color = \"%s\", \"%s\"\n", token, nonTermColor)
		} else {
			fmt.Fprintf(f, "\t\tcase %s:\n\t\t\tp_name, p_color = \"%s\", \"%s\"\n", token, token, nonTermColor)
		}
	}
	for _, token := range p.terminals.Slice() {
		if token == termToken {
			fmt.Fprintf(f, "\t\tcase gopapageno.TokenTerm:\n\t\t\tp_name, p_color = \"%s\", \"%s\"\n", token, termColor)
		} else {
			fmt.Fprintf(f, "\t\tcase %s:\n\t\t\tp_name, p_color = \"%s\", \"%s\"\n", token, token, termColor)
		}
	}
	fmt.Fprintf(f, "\t\t}\n")

	fmt.Fprintf(f, "\n\t\tswitch t.Type {\n")
	for _, token := range p.nonterminals.Slice() {
		if token == emptyToken {
			fmt.Fprintf(f, "\t\tcase gopapageno.TokenEmpty:\n\t\t\tt_name, t_color = \"%s\", \"%s\"\n", token, nonTermColor)
		} else {
			fmt.Fprintf(f, "\t\tcase %s:\n\t\t\tt_name, t_color = \"%s\", \"%s\"\n", token, token, nonTermColor)
		}
	}
	for _, token := range p.terminals.Slice() {
		if token == termToken {
			fmt.Fprintf(f, "\t\tcase gopapageno.TokenTerm:\n\t\t\tt_name, t_color = \"%s\", \"%s\"\n", token, termColor)
		} else {
			fmt.Fprintf(f, "\t\tcase %s:\n\t\t\tt_name, t_color = \"%s\", \"%s\"\n", token, token, termColor)
		}
	}
	fmt.Fprintf(f, "\t\t}\n")

	fmt.Fprintf(f, `
		sb.WriteString(fmt.Sprintf("\"%%p\" -> \"%%p\";\n", p, t))
		sb.WriteString(fmt.Sprintf("\"%%p\" [label=\"%%s\" color=\"%%s\"];\n", p, p_name, p_color))
		sb.WriteString(fmt.Sprintf("\"%%p\" [label=\"%%s\" color=\"%%s\"];\n", t, t_name, t_color))

		graphPrintRec(t.Child, t, sb, i+1)
		graphPrintRec(t.Next, p, sb, i)
	}
	graphPrintRec(root, nil, &sb, 0)
	sb.WriteString("}\n")

	fmt.Fprint(f, sb.String())
}
`)
	fmt.Fprintf(f, "\n\n")

	fmt.Fprintf(f, "func SprintToken[ValueType any](root *gopapageno.Token) string {\n")
	fmt.Fprintf(f, "\tvar sprintRec func(t *gopapageno.Token, sb *strings.Builder, indent string)\n\n")
	fmt.Fprintf(f, "\tsprintRec = func(t *gopapageno.Token, sb *strings.Builder, indent string) {\n\t\t")
	fmt.Fprintf(f, `if t == nil {
			return
		}

		sb.WriteString(indent)
		if t.Next == nil {
			sb.WriteString("└── ")
			indent += "    "
		} else {
			sb.WriteString("├── ")
			indent += "|   "
		}
`)

	fmt.Fprintf(f, "\n\t\tswitch t.Type {\n")

	for _, token := range p.nonterminals.Slice() {
		if token == emptyToken {
			fmt.Fprintf(f, "\t\tcase gopapageno.TokenEmpty:\n\t\t\tsb.WriteString(\"Empty\")\n")
		} else {
			fmt.Fprintf(f, "\t\tcase %s:\n\t\t\tsb.WriteString(\"%s\")\n", token, token)
		}
	}

	for _, token := range p.terminals.Slice() {
		if token == termToken {
			fmt.Fprintf(f, "\t\tcase gopapageno.TokenTerm:\n\t\t\tsb.WriteString(\"Term\")\n")
		} else {
			fmt.Fprintf(f, "\t\tcase %s:\n\t\t\tsb.WriteString(\"%s\")\n", token, token)
		}
	}

	fmt.Fprintf(f, "\t\tdefault:\n\t\t\tsb.WriteString(\"Unknown\")\n\t\t}\n")
	fmt.Fprintf(f, `
		if t.Value != nil {
			if v, ok := any(t.Value).(*ValueType); ok {
				sb.WriteString(fmt.Sprintf(": %%v", *v))
			} else {
				sb.WriteString(fmt.Sprintf("%%v", v))
			}
		}
		
		sb.WriteString("\n")
		
		sprintRec(t.Child, sb, indent)
		sprintRec(t.Next, sb, indent[:len(indent)-4])
	}

	var sb strings.Builder
	
	sprintRec(root, &sb, "")
	
	return sb.String()
}
`)
}
