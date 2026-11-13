package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/LorenzoTavani/gopapageno"
	"github.com/LorenzoTavani/gopapageno/internal/treecheck"
)

// TestTreesDoNotChange checks that the trees built for a fixed set of inputs are
// the ones stored in testdata/trees.golden, whatever the number of threads and the
// reduction strategy. Write the file with -update on the code to preserve, then
// run the test without it after changing the parser.
func TestTreesDoNotChange(t *testing.T) {
	strategies := []struct {
		name     string
		strategy gopapageno.ReductionStrategy
	}{
		{"sweep", gopapageno.ReductionSweep},
		{"parallel", gopapageno.ReductionParallel},
		{"mixed", gopapageno.ReductionMixed},
	}

	parse := func(source []byte, threads int, strategy gopapageno.ReductionStrategy) (*gopapageno.Token, error) {
		r := gopapageno.NewRunner(
			NewLexer(),
			NewGrammar(),
			gopapageno.WithConcurrency(threads),
			gopapageno.WithReductionStrategy(strategy),
			gopapageno.WithAverageTokenLength(2),
			gopapageno.WithParallelFactor(1),
		)

		return r.Run(context.Background(), source)
	}

	var lines []string

	// 1MB.txt and 10MB.txt are written for the grammar with the * operator, which
	// the cyclic grammar does not have, so they are not used here.
	for _, name := range []string{"small.txt", "copp_example.txt", "copp_example_1MB.txt", "LeftAssoc.txt"} {
		source, err := os.ReadFile("../data/" + name)
		if err != nil {
			t.Logf("skipping %s: %v", name, err)
			continue
		}

		ref, err := parse(source, 1, gopapageno.ReductionParallel)
		if err != nil {
			t.Fatalf("%s, reference with one thread: %v", name, err)
		}
		refDigest, refNodes := treecheck.Digest(ref)

		for _, s := range strategies {
			for _, threads := range []int{1, 2, 3, 4, 8} {
				root, err := parse(source, threads, s.strategy)
				if err != nil {
					t.Fatalf("%s, %s reduction, %d threads: %v", name, s.name, threads, err)
				}

				digest, nodes := treecheck.Digest(root)
				if digest != refDigest {
					t.Errorf("%s, %s reduction, %d threads: the tree depends on the split: %s %d against %s %d with one thread",
						name, s.name, threads, digest, nodes, refDigest, refNodes)
				}

				value := "-"
				if v, ok := root.Value.(*int64); ok {
					value = fmt.Sprint(*v)
				}

				lines = append(lines, fmt.Sprintf("file %s | %s | %d threads | %s %d %s", name, s.name, threads, digest, nodes, value))
			}
		}
	}

	treecheck.Check(t, "testdata/trees.golden", lines)
}
