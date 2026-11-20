package gopapageno

import "testing"

// prefixesOfNested is the table of prefixes generated for examples/nested/copp.
// Its only cyclic rule is E : (V TIMES OPERATOR)+ V, with E = 1, V = 3,
// OPERATOR = 32770 and TIMES = 32771, so the trie holds one path: the doubled
// prefix V TIMES OPERATOR V TIMES OPERATOR, which leads to E and rule 1.
var prefixesOfNested = []uint16{0, 0, 1, 3, 5, 0, 0, 1, 32771, 10, 0, 0, 1, 32770, 15, 0, 0, 1, 3, 20, 0, 0, 1, 32771, 25, 0, 0, 1, 32770, 30, 1, 1, 0}

// rulesOfNested is the table of rules generated for examples/nested/copp:
// S : E (rule 0), E : V TIMES OPERATOR V (rule 1, the cyclic rule with the cycle
// taken once) and V : NUMBER (rule 2), with S = 2 and NUMBER = 32769.
var rulesOfNested = []uint16{0, 0, 3, 1, 9, 3, 12, 32769, 30, 2, 0, 0, 0, 0, 1, 32771, 17, 0, 0, 1, 32770, 22, 0, 0, 1, 3, 27, 1, 1, 0, 3, 2, 0}

func TestFindRuleMatch(t *testing.T) {
	g := &Grammar{CompressedRules: rulesOfNested}

	const (
		E      = TokenType(1)
		S      = TokenType(2)
		V      = TokenType(3)
		NUMBER = TokenType(32769)
		OP     = TokenType(32770)
		TIMES  = TokenType(32771)
	)

	cases := []struct {
		name string
		rhs  []TokenType
		lhs  TokenType
		rule uint16
	}{
		{"S : E", []TokenType{E}, S, 0},
		{"E : V TIMES OPERATOR V", []TokenType{V, TIMES, OP, V}, E, 1},
		{"V : NUMBER", []TokenType{NUMBER}, V, 2},
		{"a prefix of a rule", []TokenType{V, TIMES}, TokenEmpty, 0},
		// E is smaller than TIMES, the only key of the node after V: the same
		// unsigned binary search as in findPrefixMatch.
		{"a symbol smaller than every key of its node", []TokenType{V, E}, TokenEmpty, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			lhs, rule := g.findRuleMatch(c.rhs)
			if lhs != c.lhs || rule != c.rule {
				t.Errorf("findRuleMatch(%v) = (%d, %d), want (%d, %d)", c.rhs, lhs, rule, c.lhs, c.rule)
			}
		})
	}
}

func TestFindPrefixMatch(t *testing.T) {
	g := &Grammar{CompressedPrefixes: prefixesOfNested}

	const (
		E     = TokenType(1)
		V     = TokenType(3)
		OP    = TokenType(32770)
		TIMES = TokenType(32771)
	)

	cases := []struct {
		name string
		rhs  []TokenType
		lhs  TokenType
		rule uint16
	}{
		{"the doubled prefix", []TokenType{V, TIMES, OP, V, TIMES, OP}, E, 1},
		{"a prefix that is not doubled", []TokenType{V, TIMES, OP}, TokenEmpty, 0},
		{"a symbol greater than every key of its node", []TokenType{V, TIMES, TIMES}, TokenEmpty, 0},
		// OP is smaller than TIMES, the only key of its node. The binary search
		// works on unsigned integers, so high = indexPos - 1 went from 0 to 65535
		// and the search read past the end of the table.
		{"a symbol smaller than every key of its node", []TokenType{V, TIMES, OP, V, OP}, TokenEmpty, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			lhs, rule := g.findPrefixMatch(c.rhs)
			if lhs != c.lhs || rule != c.rule {
				t.Errorf("findPrefixMatch(%v) = (%d, %d), want (%d, %d)", c.rhs, lhs, rule, c.lhs, c.rule)
			}
		})
	}
}
