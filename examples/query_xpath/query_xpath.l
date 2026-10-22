%cut /
%preamble LexerPreallocMem

%%

IDENT ([a-zA-Z0-9_\-]+|\*)(:[a-zA-Z0-9_\-]+)*

CHILD /
PARENT \\\\
ANCESTOR \\\\\\\\
DESCENDANT //

QUOTE ["'`]
CHAR [^"\n]

LBR \[
RBR \]
LPAR \(
RPAR \)

AT @
EQ =

TEXT text\(\)

OR or
AND and
NOT not

SPACE [ \t]
NEWLINE [\r\n]

%%

{ANCESTOR}
{
	token.Value = "\\\\"
	token.Type = ANCESTOR
}
{DESCENDANT}
{
	token.Value = "//"
	token.Type = DESCENDANT
}
{CHILD}
{
	token.Value = "/"
	token.Type = CHILD
}
{PARENT}
{
	token.Value = "\\"
	token.Type = PARENT
}
{LBR}
{
	token.Value = "["
	token.Type = LBR
}
{RBR}
{
	token.Value = "]"
	token.Type = RBR
}
{AT}
{
	token.Value = "@"
	token.Type = AT
}
{EQ}
{
	token.Value = "="
	token.Type = EQ
}
{TEXT}
{
	token.Value = "TEXT()"
	token.Type = TEXT
}
{OR}
{
	token.Value = "OR"
	token.Type = OR
}
{AND}
{
	token.Value = "AND"
	token.Type = AND
}
{NOT}
{
	token.Value = "NOT"
	token.Type = NOT
}
{IDENT}
{
	token.Value = text
	token.Type = IDENT
}
{LPAR}
{
	token.Value = "("
	token.Type = LPAR
}
{RPAR}
{
	token.Value = ")"
	token.Type = RPAR
}
{QUOTE}{CHAR}*{QUOTE}
{
	token.Value = text[1:len(text)-1]
	token.Type = STRING
}
({SPACE}|{NEWLINE})+
{
	return gopapageno.LexSkip
}

%%

// var lexerPool []*gopapageno.Pool[int]

func LexerPreallocMem(inputSize int, numThreads int) {
	// lexerOpenTagSemanticValuePools = make([]*gopapageno.Pool[int], numThreads)

	// avgCharsPerNumber := float64(2)
	// poolSizePerThread := int(math.Ceil((float64(inputSize) / avgCharsPerNumber) / float64(numThreads)))

	// for i := 0; i < numThreads; i++ {
	// 	lexerOpenTagSemanticValuePools[i] = gopapageno.NewPool[int](poolSizePerThread)
	// }
}
