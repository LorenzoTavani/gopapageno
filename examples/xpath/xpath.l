%cut <
%preamble LexerPreallocMem

%%

IDENTIFIER [a-zA-Z0-9_\-:]+
INFO [^<]+
SPACE [ \t]
NEWLINE [\r\n]
VALUE ([^"\\]|\\.)*

%%

<{IDENTIFIER}({SPACE}+{IDENTIFIER}="{VALUE}")*>
{
	id, attributesList := getIdAndAttributesListFrom(text)
	semanticValue := lexerOpenTagSemanticValuePools[thread].Get()
	semanticValue.SemanticValue.Id = id
	semanticValue.SemanticValue.StartPos = start
	semanticValue.SemanticValue.EndPos = end
	semanticValue.Attribute = attributesList

	token.Type = OPENTAG
	token.Value = semanticValue
}
<{IDENTIFIER}({SPACE}+{IDENTIFIER}="{VALUE}")*/>
{
	id, attributesList := getIdAndAttributesListFrom(text)
	semanticValue := lexerOpenTagSemanticValuePools[thread].Get()
	semanticValue.SemanticValue.Id = id
	semanticValue.SemanticValue.StartPos = start
	semanticValue.SemanticValue.EndPos = end
	semanticValue.Attribute = attributesList

	token.Type = OPENCLOSETAG
	token.Value = semanticValue
}
</{IDENTIFIER}>
{
	id, _ := getIdAndAttributesListFrom(text)
	semanticValue := lexerCloseTagSemanticValuePools[thread].Get()
	semanticValue.SemanticValue.Id = id
	semanticValue.SemanticValue.StartPos = start
	semanticValue.SemanticValue.EndPos = end

	token.Type = CLOSETAG
	token.Value = semanticValue
}
({SPACE}|{NEWLINE})+
{
	return gopapageno.LexSkip
}
<?[^?]+?>
{
	return gopapageno.LexSkip
}
<![^>]+>
{
	return gopapageno.LexSkip
}
{INFO}
{
	semanticValue := xpath.NewTextSemanticValue(text, start, end)

	token.Type = TEXT
	token.Value = semanticValue
}

%%

import (
	"math"
	xpath "github.com/LorenzoTavani/gopapageno/ext/xpath"
)

func getIdAndAttributesListFrom(text string) (id string, attribute *xpath.Attribute) {
	var l, r int
	attribute = nil

	l = 1
	if text[1] == byte('/') {
		l = 2
	}

	for ; r < (len(text) - 1) && text[r] != byte(' '); r++ {}

	if text[r] == byte('>') {
		if text[r-1] == byte('/') {
			r--
			id = string(text[l:r])
		} else {
			id = string(text[l:r])
		}
		return
	} else {
		id = string(text[l:r])
	}

	for r < len(text) {
		for ; text[r] == byte(' ') && r < len(text); r++ {}
		l = r

		for ; text[r] != byte('=') && r < len(text); r++ {}
		key := string(text[l:r])
		r = r + 2 // skip ="
		l = r

		for ; text[r] != byte('"') && r < len(text); r++ {}
		value := string(text[l:r])
		r = r + 1 // skip "

		attribute = &xpath.Attribute{Key: key, Value: value, Next: attribute}

		if text[r] == byte('/') || text[r] == byte('>') {
			break
		}
	}
	return
}


var lexerOpenTagSemanticValuePools []*gopapageno.Pool[xpath.OpenTagSemanticValue]
var lexerCloseTagSemanticValuePools []*gopapageno.Pool[xpath.CloseTagSemanticValue]

func LexerPreallocMem(inputSize int, numThreads int) {
	lexerOpenTagSemanticValuePools = make([]*gopapageno.Pool[xpath.OpenTagSemanticValue], numThreads)
	lexerCloseTagSemanticValuePools = make([]*gopapageno.Pool[xpath.CloseTagSemanticValue], numThreads)

	tagTypes := float64(2)
	poolSizePerThread := int(math.Ceil((float64(inputSize) / tagTypes) / float64(numThreads)))

	for i := 0; i < numThreads; i++ {
		lexerOpenTagSemanticValuePools[i] = gopapageno.NewPool[xpath.OpenTagSemanticValue](poolSizePerThread)
		lexerCloseTagSemanticValuePools[i] = gopapageno.NewPool[xpath.CloseTagSemanticValue](poolSizePerThread)
	}
}
