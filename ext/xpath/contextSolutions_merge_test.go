package xpath

import "testing"

func TestMergeContextSolutions(t *testing.T) {
	a, b, c := new(NonTerminal), new(NonTerminal), new(NonTerminal)

	small := make(contextSolutionsMap)
	small.addContextSolution(1, 2, a)
	large := make(contextSolutionsMap)
	large.addContextSolution(1, 2, b)
	large.addContextSolution(3, 4, c)
	large.addContextSolution(5, 6, c)

	got := mergeContextSolutions(small, large)
	if len(got[1][2]) != 2 || len(got[3][4]) != 1 || len(got[5][6]) != 1 {
		t.Errorf("small into large: got %v", got)
	}

	// An empty receiver adopts the incoming map, and an empty incoming map leaves the receiver as it is.
	m := make(contextSolutionsMap)
	m.addContextSolution(7, 8, a)
	if got := mergeContextSolutions(make(contextSolutionsMap), m); len(got[7][8]) != 1 {
		t.Errorf("empty receiver: got %v", got)
	}
	if got := mergeContextSolutions(m, make(contextSolutionsMap)); len(got[7][8]) != 1 {
		t.Errorf("empty incoming: got %v", got)
	}
	if got := mergeContextSolutions(m, nil); len(got[7][8]) != 1 {
		t.Errorf("nil incoming: got %v", got)
	}
}
