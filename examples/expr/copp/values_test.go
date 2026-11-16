package main

import (
	"context"
	"os"
	"testing"

	"github.com/LorenzoTavani/gopapageno"
)

// TestSumsAreCorrect checks the value computed for a few inputs whose result is
// known by hand. The benchmark entries have a Result field that nobody compares,
// and the golden file only records what the parser did, so without this test a
// wrong semantic action (see E6 in KNOWN_ISSUES.md) goes unnoticed.
func TestSumsAreCorrect(t *testing.T) {
	cases := []struct {
		file string
		want int64
	}{
		{"small.txt", 2 * 45},                // two sums of 1 + 2 + ... + 9
		{"copp_example.txt", 18},             // 1 + 2 + 3/4/5/6 + 7 + 8, and 3/4/5/6 is 0
		{"copp_example_1MB.txt", 18 * 40000}, // the same line 40000 times
		{"LeftAssoc.txt", 2250000},           // 450000 terms, 1 + ... + 9 on each of the 50000 lines
	}

	strategies := map[string]gopapageno.ReductionStrategy{
		"sweep":    gopapageno.ReductionSweep,
		"parallel": gopapageno.ReductionParallel,
		"mixed":    gopapageno.ReductionMixed,
	}

	for _, c := range cases {
		source, err := os.ReadFile("../data/" + c.file)
		if err != nil {
			t.Fatalf("%s: %v", c.file, err)
		}

		for name, strategy := range strategies {
			for _, threads := range []int{1, 2, 3, 4, 8} {
				r := gopapageno.NewRunner(
					NewLexer(),
					NewGrammar(),
					gopapageno.WithConcurrency(threads),
					gopapageno.WithReductionStrategy(strategy),
					gopapageno.WithAverageTokenLength(2),
					gopapageno.WithParallelFactor(1),
				)

				root, err := r.Run(context.Background(), source)
				if err != nil {
					t.Fatalf("%s, %s, %d threads: %v", c.file, name, threads, err)
				}

				v, ok := root.Value.(*int64)
				if !ok {
					t.Errorf("%s, %s reduction, %d threads: the root holds %T, want *int64", c.file, name, threads, root.Value)
				} else if *v != c.want {
					t.Errorf("%s, %s reduction, %d threads: got %d, want %d", c.file, name, threads, *v, c.want)
				}
			}
		}
	}
}
