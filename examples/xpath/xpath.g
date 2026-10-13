%axiom DOCUMENT

%preamble ParserPreallocMem

%%

DOCUMENT : ELEM {
	$$.Value = $1.Value
};

ELEM : ELEM OPENTAG ELEM CLOSETAG
{
	openTag := $2.Value.(*xpath.OpenTagSemanticValue)
	closeTag := $4.Value.(*xpath.CloseTagSemanticValue)

	element := parserElementsPools[thread].Get()
	element.SetFromExtremeTags(*openTag, *closeTag)

	generativeNonTerminal := $1.Value.(*xpath.NonTerminal)
	wrappedNonTerminal := $3.Value.(*xpath.NonTerminal)

	nt := parserNonTerminalPools[thread].Get()
	reducedNonTerminal := nt.SetNode(element).SetDirectChildAndInheritItsChildren(generativeNonTerminal)

	xpath.Reduce(reducedNonTerminal, generativeNonTerminal, wrappedNonTerminal)

	$$.Value = reducedNonTerminal
} | OPENTAG ELEM CLOSETAG
{
	openTag := $1.Value.(*xpath.OpenTagSemanticValue)
	closeTag := $3.Value.(*xpath.CloseTagSemanticValue)

	element := parserElementsPools[thread].Get()
	element.SetFromExtremeTags(*openTag, *closeTag)

	wrappedNonTerminal := $2.Value.(*xpath.NonTerminal)
	nt := parserNonTerminalPools[thread].Get()
	reducedNonTerminal := nt.SetNode(element)

	xpath.Reduce(reducedNonTerminal, nil, wrappedNonTerminal)

	$$.Value = reducedNonTerminal
} | ELEM OPENTAG CLOSETAG
{
	openTag := $2.Value.(*xpath.OpenTagSemanticValue)
	closeTag := $3.Value.(*xpath.CloseTagSemanticValue)

	element := parserElementsPools[thread].Get()
	element.SetFromExtremeTags(*openTag, *closeTag)

	generativeNonTerminal := $1.Value.(*xpath.NonTerminal)

	nt := parserNonTerminalPools[thread].Get()
	reducedNonTerminal := nt.SetNode(element).SetDirectChildAndInheritItsChildren(generativeNonTerminal)

	xpath.Reduce(reducedNonTerminal, generativeNonTerminal, nil)

	$$.Value = reducedNonTerminal
} | OPENTAG CLOSETAG
{
	openTag := $1.Value.(*xpath.OpenTagSemanticValue)
	closeTag := $2.Value.(*xpath.CloseTagSemanticValue)

	element := parserElementsPools[thread].Get()
	element.SetFromExtremeTags(*openTag, *closeTag)

	nt := parserNonTerminalPools[thread].Get()
	reducedNonTerminal := nt.SetNode(element)

	xpath.Reduce(reducedNonTerminal, nil, nil)

	$$.Value = reducedNonTerminal
} | ELEM OPENCLOSETAG
{
	openCloseTag := $2.Value.(*xpath.OpenTagSemanticValue)

	element := parserElementsPools[thread].Get()
	element.SetFromSingleTag(*openCloseTag)

	generativeNonTerminal := $1.Value.(*xpath.NonTerminal)
	nt := parserNonTerminalPools[thread].Get()
	reducedNonTerminal := nt.SetNode(element).SetDirectChildAndInheritItsChildren(generativeNonTerminal)

	xpath.Reduce(reducedNonTerminal, generativeNonTerminal, nil)

	$$.Value = reducedNonTerminal
} | OPENCLOSETAG
{
	openCloseTag := $1.Value.(*xpath.OpenTagSemanticValue)

	element := parserElementsPools[thread].Get()
	element.SetFromSingleTag(*openCloseTag)

	nt := parserNonTerminalPools[thread].Get()
	reducedNonTerminal := nt.SetNode(element)

	xpath.Reduce(reducedNonTerminal, nil, nil)

	$$.Value = reducedNonTerminal
} | ELEM TEXT
{
	tsv := $2.Value.(*xpath.TextSemanticValue)

	text := new(xpath.Text)
	text.SetFromText(*tsv)

	generativeNonTerminal := $1.Value.(*xpath.NonTerminal)

	nt := parserNonTerminalPools[thread].Get()
	reducedNonTerminal := nt.SetNode(text).SetDirectChildAndInheritItsChildren(generativeNonTerminal)

	xpath.Reduce(reducedNonTerminal, generativeNonTerminal, nil)

	$$.Value = reducedNonTerminal
} | TEXT
{
	tsv := $1.Value.(*xpath.TextSemanticValue)

	text := new(xpath.Text)
	text.SetFromText(*tsv)

	nt := parserNonTerminalPools[thread].Get()
	reducedNonTerminal := nt.SetNode(text)

	xpath.Reduce(reducedNonTerminal, nil, nil)


	$$.Value = reducedNonTerminal
};

%%

import (
	"github.com/LorenzoTavani/gopapageno/ext/xpath"
	"math"
)

var parserElementsPools []*gopapageno.Pool[xpath.Element]
var parserNonTerminalPools []*gopapageno.Pool[xpath.NonTerminal]

// ParserPreallocMem initializes all the memory pools required by the semantic function of the parser.
func ParserPreallocMem(inputSize int, numThreads int) {
	tagTypes := float64(2)
	poolSizePerThread := int(math.Ceil(float64(inputSize) / tagTypes))

	parserElementsPools = make([]*gopapageno.Pool[xpath.Element], numThreads)
	parserNonTerminalPools = make([]*gopapageno.Pool[xpath.NonTerminal], numThreads)
	for i := 0; i < numThreads; i++ {
		parserElementsPools[i] = gopapageno.NewPool[xpath.Element](poolSizePerThread)
		parserNonTerminalPools[i] = gopapageno.NewPool[xpath.NonTerminal](poolSizePerThread)
	}
}
