package main

// Copiare questo file, senza modifiche, in examples/adder/opp, examples/adder/aopp
// e examples/adder/copp: ogni cartella e' un pacchetto main con le sue NewLexer,
// NewGrammar ed entries (quest'ultima in benchmark_test.go).

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/LorenzoTavani/gopapageno"
)

var goldenStrategies = map[string]gopapageno.ReductionStrategy{
	"sweep":    gopapageno.ReductionSweep,
	"parallel": gopapageno.ReductionParallel,
	"mixed":    gopapageno.ReductionMixed,
}

func goldenParse(source []byte, threads int, strategy gopapageno.ReductionStrategy, avg int, pf float64) (int64, error) {
	r := gopapageno.NewRunner(
		NewLexer(),
		NewGrammar(),
		gopapageno.WithConcurrency(threads),
		gopapageno.WithReductionStrategy(strategy),
		gopapageno.WithAverageTokenLength(avg),
		gopapageno.WithParallelFactor(pf),
	)
	root, err := r.Run(context.Background(), source)
	if err != nil {
		return 0, err
	}
	v, ok := root.Value.(*int64)
	if !ok {
		return 0, fmt.Errorf("the value of the root is %T, not *int64", root.Value)
	}
	return *v, nil
}

// Small inputs: if the big one below fails, these tell why.
func TestSmallSums(t *testing.T) {
	for _, c := range []struct {
		source string
		want   int64
	}{
		{"1 + 2", 3},
		{"1 + 2 + 3 + 4", 10},
		{"(1 + 2) + (3 + 4)", 10},
		{"1 + (2 + (3 + 4))", 10},
		{"1 +\n2 +\n3 +\n4", 10},
	} {
		for name, strategy := range goldenStrategies {
			for _, threads := range []int{1, 2, 4, 8} {
				got, err := goldenParse([]byte(c.source), threads, strategy, 2, 1)
				if err != nil || got != c.want {
					t.Errorf("%q, %s reduction, %d threads: got %d (%v), want %d", c.source, name, threads, got, err, c.want)
				}
			}
		}
	}
}

// The benchmark entries carry the expected result of every input file, but the
// benchmarks never check it. Inputs over 2 MB are left out to keep the test
// fast and light on memory.
func TestBenchmarkInputsGiveTheExpectedResult(t *testing.T) {
	for _, entry := range entries {
		info, err := os.Stat(entry.Filename)
		if err != nil {
			t.Logf("skipping %s: %v", entry.Filename, err)
			continue
		}
		if info.Size() > 2<<20 {
			continue
		}
		source, err := os.ReadFile(entry.Filename)
		if err != nil {
			t.Fatal(err)
		}
		for name, strategy := range goldenStrategies {
			for _, threads := range []int{1, 2, 4, 8} {
				got, err := goldenParse(source, threads, strategy, entry.AvgTokenLength, entry.ParallelFactor)
				if err != nil || got != entry.Result {
					t.Errorf("%s, %s reduction, %d threads: got %d (%v), want %d", entry.Filename, name, threads, got, err, entry.Result)
				}
			}
		}
	}
}
