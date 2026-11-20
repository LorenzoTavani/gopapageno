package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/LorenzoTavani/gopapageno"
	"github.com/LorenzoTavani/gopapageno/internal/treecheck"
)

// cut says where sum puts a new line before every seventh term. The lexer cuts
// the input at new lines (%cut \n), so the text can be split between the workers
// there.
type cut int

const (
	cutAfterOperator cut = iota // ... 6 *+=+= \n7 ...
	cutBeforeTimes              // ... 6\n *+=+= 7 ...
)

// sum builds the source 1 *+=+= 2 *+=+= ... *+=+= n and the value of the sum.
func sum(n int, at cut) (string, int64) {
	var sb strings.Builder
	var want int64

	for i := 1; i <= n; i++ {
		if i > 1 {
			if at == cutBeforeTimes && i%7 == 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(" *+=+= ")
			if at == cutAfterOperator && i%7 == 0 {
				sb.WriteString("\n")
			}
		}

		fmt.Fprint(&sb, i)
		want += int64(i)
	}

	return sb.String(), want
}

func newRunner(threads int, strategy gopapageno.ReductionStrategy) *gopapageno.Runner {
	return gopapageno.NewRunner(
		NewLexer(),
		NewGrammar(),
		gopapageno.WithConcurrency(threads),
		gopapageno.WithReductionStrategy(strategy),
		gopapageno.WithAverageTokenLength(2),
		gopapageno.WithParallelFactor(0.5),
	)
}

var strategies = []struct {
	name     string
	strategy gopapageno.ReductionStrategy
}{
	{"sweep", gopapageno.ReductionSweep},
	{"parallel", gopapageno.ReductionParallel},
	{"mixed", gopapageno.ReductionMixed},
}

// TestOneThread checks the value and the shape of the tree for the inputs of this
// example when a single worker parses the whole text. The digests are stored in
// testdata/trees.golden: write the file with -update on the code to preserve, then
// run the test without it after changing the parser.
//
// Only one thread is used because the example does not work with more (C13 in
// KNOWN_ISSUES.md, see TestKnownIssueC13).
func TestOneThread(t *testing.T) {
	example, err := os.ReadFile("../data/example.txt")
	if err != nil {
		t.Fatal(err)
	}

	long, longValue := sum(3000, cutAfterOperator)

	inputs := []struct {
		name   string
		source string
		want   int64
	}{
		{"example.txt", string(example), 1 + 2 + 10},
		{"sum of 3000 terms", long, longValue},
	}

	var lines []string

	for _, in := range inputs {
		for _, s := range strategies {
			root, err := newRunner(1, s.strategy).Run(context.Background(), []byte(in.source))
			if err != nil {
				t.Fatalf("%s, %s reduction: %v", in.name, s.name, err)
			}

			v, ok := root.Value.(int64)
			if !ok {
				t.Fatalf("%s, %s reduction: the root holds %T, want int64", in.name, s.name, root.Value)
			}

			if v != in.want {
				t.Errorf("%s, %s reduction: got %d, want %d", in.name, s.name, v, in.want)
			}

			digest, nodes := treecheck.Digest(root)
			lines = append(lines, fmt.Sprintf("%s | %s | 1 thread | %s %d %d", in.name, s.name, digest, nodes, v))
		}
	}

	treecheck.Check(t, "testdata/trees.golden", lines)
}

// TestKnownIssueC13 documents that the example cannot be parsed by more than one
// worker: with two or more threads the parser returns an error as soon as the
// text is split. A worker that starts inside the cycle puts aside its first
// terminal, *, and the line that turns ≐ into ⋖ after a terminal put aside
// (copp.go) makes the next terminal, the operator, the start of a new right-hand
// side, which is not a prefix of any rule. The failure is the same on commit
// ce75289, the last one before any change to the cyclic parser (its parser is
// the one of commit 46dea04), so it is not caused by those changes.
//
// The test passes while the problem exists. When it starts failing, the problem
// is fixed: move these cases into TestOneThread and delete this test.
//
// Eight terms are the smallest input that fails: the second worker gets only
// "7 *+=+= 8". Before the fix of the binary searches in grammar.go the cut
// before the * made a worker panic with an index out of range, which a test
// cannot catch because the panic is raised in another goroutine; now it returns
// an error like the other cut.
func TestKnownIssueC13(t *testing.T) {
	inputs := []struct {
		name string
		n    int
		at   cut
	}{
		{"8 terms, cut after the operator", 8, cutAfterOperator},
		{"3000 terms, cut after the operator", 3000, cutAfterOperator},
		{"3000 terms, cut before the *", 3000, cutBeforeTimes},
	}

	for _, in := range inputs {
		source, want := sum(in.n, in.at)

		for _, s := range strategies {
			for _, threads := range []int{2, 3, 4, 8} {
				root, err := newRunner(threads, s.strategy).Run(context.Background(), []byte(source))
				if err != nil {
					continue // still broken, as documented
				}

				if v, ok := root.Value.(int64); ok && v == want {
					t.Errorf("%s, %s reduction, %d threads: parsed correctly, so C13 seems fixed: "+
						"move this case into TestOneThread and delete TestKnownIssueC13", in.name, s.name, threads)
				} else {
					t.Errorf("%s, %s reduction, %d threads: no error but the value is %v, want %d: the failure has changed, update C13",
						in.name, s.name, threads, root.Value, want)
				}
			}
		}
	}
}
