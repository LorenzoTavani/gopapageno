package main

import (
	"github.com/giornetta/gopapageno"
	"github.com/giornetta/gopapageno/benchmark"
	"runtime"
	"testing"
)

const baseFolder = "../data/"

const (
	fileSmall = "small.txt"
	fileMB    = "1MB.txt"
	file10MB  = "10MB.txt"
)

const (
	resultSmall = 1 + 2*3*(4+5)
	resultMB    = (1*2*3 + 11*222*3333*(1+2)) * 25966
	result10MB  = (1*2*3 + 11*222*3333*(1+2)) * 257473
)

var entries = []*benchmark.Entry[int64]{
	{
		Filename:       baseFolder + fileSmall,
		ParallelFactor: gopapageno.DefaultParallelFactor,
		AvgTokenLength: gopapageno.DefaultAverageTokenLength,
		Result:         resultSmall,
	},
	{
		Filename:       baseFolder + fileMB,
		ParallelFactor: gopapageno.DefaultParallelFactor,
		AvgTokenLength: gopapageno.DefaultAverageTokenLength,
		Result:         resultMB,
	},
	{
		Filename:       baseFolder + file10MB,
		ParallelFactor: gopapageno.DefaultParallelFactor,
		AvgTokenLength: gopapageno.DefaultAverageTokenLength,
		Result:         result10MB,
	},
}

func BenchmarkParse(b *testing.B) {
	benchmark.Runner[int64](b, gopapageno.AOPP, NewLexer, NewGrammar, entries)
}

func TestProfile(t *testing.T) {
	opts := &gopapageno.RunOptions{
		Concurrency:       runtime.NumCPU(),
		AvgTokenLength:    gopapageno.DefaultAverageTokenLength,
		ReductionStrategy: gopapageno.ReductionParallel,
		ParallelFactor:    gopapageno.DefaultParallelFactor,
	}

	benchmark.Profile(t, NewLexer, NewGrammar, opts, baseFolder+fileMB)
}
