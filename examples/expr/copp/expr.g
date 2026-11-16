%axiom S

%preamble ParserPreallocMem

%%

S : P
{
	$$.Value = $1.Value
} | D
{
    $$.Value = $1.Value
} | E
{
  $$.Value = $1.Value
} | T
{
    $$.Value = $1.Value
};

P : (T PLUS)+ T
{
	// The rule is cyclic, so it is applied once per term. The first time the
	// node does not exist and takes the sum of the first two terms; when the
	// sequence is extended (RuleAppend) the node already holds the partial sum.
	var firstValue int64

	if !ruleFlags.Has(gopapageno.RuleAppend) {
		$$.Value = parserInt64Pools[thread].Get()

		firstValue = *$1.Value.(*int64)
	} else {
		firstValue = *$$.Value.(*int64)
	}

	*$$.Value.(*int64) = firstValue + *$3.Value.(*int64)
} | LPAR S RPAR
{
	$$.Value = $2.Value
} | NUMBER
{
    $$.Value = $1.Value
};

T : D DIVIDE E
{
    newValue := parserInt64Pools[thread].Get()
    *newValue = *$1.Value.(*int64) / *$3.Value.(*int64)
    $$.Value = newValue
} | NUMBER
{
    $$.Value = $1.Value
} | LPAR S RPAR
{
  $$.Value = $2.Value
};

D : D DIVIDE E
{
    newValue := parserInt64Pools[thread].Get()
    *newValue = *$1.Value.(*int64) / *$3.Value.(*int64)
    $$.Value = newValue
} | NUMBER
{
    $$.Value = $1.Value
} | LPAR S RPAR
{
    $$.Value = $2.Value
};

E : NUMBER
{
    $$.Value = $1.Value
} | LPAR S RPAR
{
    $$.Value = $2.Value
};

%%

import (
	"math"
)

var parserInt64Pools []*gopapageno.Pool[int64]

// ParserPreallocMem initializes all the memory pools required by the semantic function of the parser.
func ParserPreallocMem(inputSize int, numThreads int) {
	parserInt64Pools = make([]*gopapageno.Pool[int64], numThreads)

	avgCharsPerNumber := float64(4)
	poolSizePerThread := int(math.Ceil((float64(inputSize) / avgCharsPerNumber) / float64(numThreads)))

	for i := 0; i < numThreads; i++ {
		parserInt64Pools[i] = gopapageno.NewPool[int64](poolSizePerThread)
	}
}
