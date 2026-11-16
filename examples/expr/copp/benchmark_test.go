package main

import (
	"testing"

	"github.com/LorenzoTavani/gopapageno"
	"github.com/LorenzoTavani/gopapageno/benchmark"
)

const baseFolder = "../data/"

// The files are written for this grammar, which has + and / but not *: the
// lexer of the cyclic grammar does not know the character, so 1MB.txt and
// 10MB.txt (made for the grammar with *) cannot be used here.
var entries = []*benchmark.Entry[int64]{
	{
		// Two sums of 1 + 2 + ... + 9.
		Filename:       baseFolder + "small.txt",
		ParallelFactor: 1,
		AvgTokenLength: 2,
		Result:         2 * 45,
	},
	{
		// 40000 lines of 1 + 2 + 3/4/5/6 + 7 + 8, where 3/4/5/6 is 0 with integer division.
		Filename:       baseFolder + "copp_example_1MB.txt",
		ParallelFactor: 1,
		AvgTokenLength: 2,
		Result:         18 * 40000,
	},
	{
		// 50000 lines of 1 + 2 + ... + 9, with the last one cut short.
		Filename:       baseFolder + "LeftAssoc.txt",
		ParallelFactor: 1,
		AvgTokenLength: 2,
		Result:         2250000,
	},
}

func BenchmarkParse(b *testing.B) {
	benchmark.Runner(b, gopapageno.COPP, gopapageno.ReductionParallel, NewLexer, NewGrammar, entries)
}

func BenchmarkParseOnly(b *testing.B) {
	benchmark.ParserRunner(b, gopapageno.COPP, gopapageno.ReductionParallel, NewLexer, NewGrammar, entries)
}
