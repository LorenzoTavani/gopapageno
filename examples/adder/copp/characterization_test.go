package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
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

	var lines []string
	add := func(label string, source []byte) {
		ref, err := parseRoot(source, 1, gopapageno.ReductionParallel)
		if err != nil {
			t.Fatalf("%s, reference with one thread: %v", label, err)
		}
		refDigest, refNodes := treecheck.Digest(ref)

		for _, s := range strategies {
			for _, threads := range []int{1, 2, 3, 4, 8} {
				root, err := parseRoot(source, threads, s.strategy)
				if err != nil {
					t.Fatalf("%s, %s reduction, %d threads: %v", label, s.name, threads, err)
				}

				digest, nodes := treecheck.Digest(root)
				if digest != refDigest {
					t.Errorf("%s, %s reduction, %d threads: the tree depends on the split: %s %d against %s %d with one thread",
						label, s.name, threads, digest, nodes, refDigest, refNodes)
				}

				value := "-"
				if v, ok := root.Value.(*int64); ok {
					value = fmt.Sprint(*v)
				}

				lines = append(lines, fmt.Sprintf("%s | %s | %d threads | %s %d %s", label, s.name, threads, digest, nodes, value))
			}
		}
	}

	for seed := int64(1); seed <= 5; seed++ {
		rng := rand.New(rand.NewSource(seed))

		var sb strings.Builder
		genE(rng, 3, 100+rng.Intn(300), &sb)

		add(fmt.Sprintf("random sum %d", seed), []byte(sb.String()))
	}

	for _, name := range []string{"small.txt", "1MB.txt"} {
		source, err := os.ReadFile("../data/" + name)
		if err != nil {
			t.Logf("skipping %s: %v", name, err)
			continue
		}

		add("file "+name, source)
	}

	treecheck.Check(t, "testdata/trees.golden", lines)
}
