package main

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/LorenzoTavani/gopapageno"
)

const maxNodes = 5_000_000

// genE writes an E with the given number of terms and returns the sum of its numbers.
func genE(r *rand.Rand, depth, terms int, sb *strings.Builder) int64 {
	var sum int64
	for i := 0; i < terms; i++ {
		if i > 0 {
			sb.WriteString(randomSep(r) + "+" + randomSep(r))
		}
		sum += genT(r, depth, sb)
	}
	return sum
}

func genT(r *rand.Rand, depth int, sb *strings.Builder) int64 {
	if depth <= 0 || r.Intn(3) > 0 {
		n := int64(1 + r.Intn(99))
		fmt.Fprintf(sb, "%d", n)
		return n
	}
	sb.WriteString("(")
	var sum int64
	if r.Intn(2) == 0 {
		sum = genT(r, depth-1, sb)
	} else {
		sum = genE(r, depth-1, 2+r.Intn(3), sb)
	}
	sb.WriteString(")")
	return sum
}

func randomSep(r *rand.Rand) string {
	if r.Intn(4) == 0 {
		return "\n"
	}
	return " "
}

// shape describes an unranked tree: the type of each node and, in parentheses, its children.
func shape(t *gopapageno.Token, budget *int) string {
	*budget--
	if *budget < 0 {
		return "LOOP"
	}
	s := fmt.Sprint(t.Type)
	if t.Child != nil {
		var parts []string
		for c := t.Child; c != nil; c = c.Next {
			parts = append(parts, shape(c, budget))
			if *budget < 0 {
				return "LOOP"
			}
		}
		s += "(" + strings.Join(parts, " ") + ")"
	}
	return s
}

// lastChildOK checks that LastChild is really the last child of every node.
func lastChildOK(t *gopapageno.Token, budget *int) error {
	*budget--
	if *budget < 0 {
		return fmt.Errorf("too many nodes: probably a cycle")
	}
	if t.Child == nil {
		return nil
	}
	last := t.Child
	for c := t.Child; c != nil; c = c.Next {
		if err := lastChildOK(c, budget); err != nil {
			return err
		}
		last = c
	}
	if t.LastChild != last {
		return fmt.Errorf("node of type %v: LastChild is not the last child", t.Type)
	}
	return nil
}

func parseRoot(source []byte, threads int, strategy gopapageno.ReductionStrategy) (*gopapageno.Token, error) {
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

// The tree must not depend on how the input is split among the workers.
func TestTreeDoesNotDependOnTheSplit(t *testing.T) {
	strategies := map[string]gopapageno.ReductionStrategy{
		"sweep":    gopapageno.ReductionSweep,
		"parallel": gopapageno.ReductionParallel,
		"mixed":    gopapageno.ReductionMixed,
	}
	for seed := int64(1); seed <= 5; seed++ {
		rng := rand.New(rand.NewSource(seed))
		var sb strings.Builder
		want := genE(rng, 3, 100+rng.Intn(300), &sb)
		source := []byte(sb.String())

		ref, err := parseRoot(source, 1, gopapageno.ReductionSweep)
		if err != nil {
			t.Fatalf("seed %d, 1 thread: %v", seed, err)
		}
		budget := maxNodes
		refShape := shape(ref, &budget)
		if v, ok := ref.Value.(*int64); !ok || *v != want {
			t.Fatalf("seed %d, 1 thread: wrong sum, want %d", seed, want)
		}

		for name, strategy := range strategies {
			for _, threads := range []int{2, 3, 4, 8} {
				label := fmt.Sprintf("seed %d, %s, %d threads", seed, name, threads)
				root, err := parseRoot(source, threads, strategy)
				if err != nil {
					t.Errorf("%s: %v", label, err)
					continue
				}
				budget := maxNodes
				if got := shape(root, &budget); got != refShape {
					t.Errorf("%s: the tree differs from the sequential one (%d against %d characters)", label, len(got), len(refShape))
				}
				budget = maxNodes
				if err := lastChildOK(root, &budget); err != nil {
					t.Errorf("%s: %v", label, err)
				}
				if v, ok := root.Value.(*int64); !ok || *v != want {
					t.Errorf("%s: wrong sum, want %d", label, want)
				}
			}
		}
	}
}
