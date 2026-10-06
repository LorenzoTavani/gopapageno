package main

import (
	"github.com/giornetta/gopapageno"
	"github.com/giornetta/gopapageno/benchmark"
	"runtime"
	"testing"
)

const baseFolder = "data/"

const fileExample = "example.txt"

var entries = []*benchmark.Entry[any]{
	{
		Filename:       baseFolder + fileExample,
		ParallelFactor: gopapageno.DefaultParallelFactor,
		AvgTokenLength: gopapageno.DefaultAverageTokenLength,
		Result:         nil,
	},
}

func BenchmarkParse(b *testing.B) {
	benchmark.Runner[any](b, gopapageno.COPP, NewLexer, NewGrammar, entries)
}

func TestProfile(t *testing.T) {
	opts := &gopapageno.RunOptions{
		Concurrency:       runtime.NumCPU(),
		AvgTokenLength:    gopapageno.DefaultAverageTokenLength,
		ReductionStrategy: gopapageno.ReductionParallel,
		ParallelFactor:    gopapageno.DefaultParallelFactor,
	}

	benchmark.Profile(t, NewLexer, NewGrammar, opts, baseFolder+fileExample)
}
