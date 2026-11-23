package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/LorenzoTavani/gopapageno"
	"github.com/LorenzoTavani/gopapageno/internal/treecheck"
)

func randomSpace(r *rand.Rand) string {
	if r.Intn(5) == 0 {
		return "\n"
	}

	return " "
}

func randomWord(r *rand.Rand) string {
	letters := "abcdefghij"
	word := make([]byte, 1+r.Intn(6))
	for i := range word {
		word[i] = letters[r.Intn(len(letters))]
	}

	return string(word)
}

func genValue(r *rand.Rand, depth int, sb *strings.Builder) {
	switch k := r.Intn(8); {
	case depth > 0 && k < 2:
		genObject(r, depth-1, r.Intn(20), sb)
	case depth > 0 && k < 4:
		genArray(r, depth-1, sb)
	case k == 4:
		sb.WriteString(`"` + randomWord(r) + `"`)
	case k == 5:
		fmt.Fprintf(sb, "%d", r.Intn(1000))
	case k == 6:
		if r.Intn(2) == 0 {
			sb.WriteString("true")
		} else {
			sb.WriteString("false")
		}
	default:
		sb.WriteString("null")
	}
}

func genArray(r *rand.Rand, depth int, sb *strings.Builder) {
	sb.WriteString("[")
	for i, n := 0, r.Intn(30); i < n; i++ {
		if i > 0 {
			sb.WriteString("," + randomSpace(r))
		}
		genValue(r, depth, sb)
	}
	sb.WriteString("]")
}

func genObject(r *rand.Rand, depth, members int, sb *strings.Builder) {
	sb.WriteString("{")
	for i := 0; i < members; i++ {
		if i > 0 {
			sb.WriteString("," + randomSpace(r))
		}
		sb.WriteString(`"` + randomWord(r) + `": `)
		genValue(r, depth, sb)
	}
	sb.WriteString("}")
}

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
			gopapageno.WithAverageTokenLength(4),
			gopapageno.WithParallelFactor(1),
		)

		return r.Run(context.Background(), source)
	}

	var lines []string
	add := func(label string, source []byte) {
		ref, err := parse(source, 1, gopapageno.ReductionParallel)
		if err != nil {
			t.Fatalf("%s, reference with one thread: %v", label, err)
		}
		refDigest, refNodes := treecheck.Digest(ref)

		for _, s := range strategies {
			for _, threads := range []int{1, 2, 3, 4, 8} {
				root, err := parse(source, threads, s.strategy)
				if err != nil {
					t.Fatalf("%s, %s reduction, %d threads: %v", label, s.name, threads, err)
				}

				digest, nodes := treecheck.Digest(root)
				if digest != refDigest {
					t.Errorf("%s, %s reduction, %d threads: the tree depends on the split: %s %d against %s %d with one thread",
						label, s.name, threads, digest, nodes, refDigest, refNodes)
				}

				lines = append(lines, fmt.Sprintf("%s | %s | %d threads | %s %d", label, s.name, threads, digest, nodes))
			}
		}
	}

	for seed := int64(1); seed <= 5; seed++ {
		rng := rand.New(rand.NewSource(seed))

		var sb strings.Builder
		genObject(rng, 3, 200+rng.Intn(300), &sb)

		add(fmt.Sprintf("random object %d", seed), []byte(sb.String()))
	}

	// The smallest input that lost nodes before the fix of CombineSweepLOS: with
	// the sweep reduction and three threads the pairs of the middle chunk were
	// lost, 83 nodes instead of 115.
	pairs := make([]string, 7)
	for i := range pairs {
		pairs[i] = "\"k\": [1,\n2,\n3]"
	}
	add("seven pairs", []byte("{"+strings.Join(pairs, ", ")+"}"))

	for _, name := range []string{"example.json", "emojis.json"} {
		source, err := os.ReadFile("../data/" + name)
		if err != nil {
			t.Logf("skipping %s: %v", name, err)
			continue
		}

		add("file "+name, source)
	}

	treecheck.Check(t, "testdata/trees.golden", lines)
}
