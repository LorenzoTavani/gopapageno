package xpath

import (
	"errors"
)

// TODO(vvihorev): make the map use the start and end positions of a non terminal in a more efficient way?
type contextSolutionsMap map[int]map[int][]*NonTerminal

func (ctxSolMap contextSolutionsMap) addContextSolution(start, end int, sols ...*NonTerminal) {
	_, ok := ctxSolMap[start]
	if !ok {
		ctxSolMap[start] = make(map[int][]*NonTerminal)
	}
	_, ok = ctxSolMap[start][end]
	if !ok {
		ctxSolMap[start][end] = make([]*NonTerminal, 0)
	}
	ctxSolMap[start][end] = append(ctxSolMap[start][end], sols...)
}

func (ctxSolMap contextSolutionsMap) hasSolutionsFor(ctx *NonTerminal) bool {
	return len(ctxSolMap.solutionsFor(ctx.Position().Start(), ctx.Position().End())) > 0
}

func (ctxSolMap contextSolutionsMap) solutionsFor(start, end int, maps ...contextSolutionsMap) (solutions []*NonTerminal) {
	solutions = ctxSolMap[start][end]

	for i := 0; i < len(maps); i++ {
		tmpNodesToVisit := []*NonTerminal{}

		for len(solutions) > 0 {
			currentNode := solutions[0]
			solutions = solutions[1:]
			tmpNodesToVisit = append(tmpNodesToVisit, maps[i].solutionsFor(currentNode.Position().Start(), currentNode.Position().End())...)
		}
		solutions = tmpNodesToVisit
	}
	return
}

func (ctxSolMap contextSolutionsMap) transitiveClosure(maps ...contextSolutionsMap) (result contextSolutionsMap) {
	dest := make(contextSolutionsMap)

	for ctxStart := range ctxSolMap {
		for ctxEnd := range ctxSolMap[ctxStart] {
			solutionsReachableFromContext := ctxSolMap.solutionsFor(ctxStart, ctxEnd, maps...)
			dest.addContextSolution(ctxStart, ctxEnd, solutionsReachableFromContext...)
		}
	}
	result = dest
	return
}

func (ctxSolMap contextSolutionsMap) convertToGroupOfSolutionsPositions() (positions []Position) {
	for start := range ctxSolMap {
		for end := range ctxSolMap[start] {
			for _, solution := range ctxSolMap[start][end] {
				positions = append(positions, solution.Position())
			}
		}
	}
	return
}

func transitiveClosure(maps []contextSolutionsMap) contextSolutionsMap {
	start := maps[0]
	return start.transitiveClosure(maps[1:]...)
}

// mergeContextSolutions joins two context-solution maps and returns the result.
//
// The smaller map is merged into the larger one, and when one of the two is
// empty the other one is returned as it is, so the cost is proportional to the
// size of the smaller map and not to the size of the larger one. The input maps
// are consumed: only the returned map must be used afterwards.
func mergeContextSolutions(receiving, incoming contextSolutionsMap) contextSolutionsMap {
	if len(incoming) == 0 {
		return receiving
	}
	if len(receiving) == 0 {
		return incoming
	}
	if len(receiving) < len(incoming) {
		receiving, incoming = incoming, receiving
	}
	for start, ends := range incoming {
		for end, solutions := range ends {
			receiving.addContextSolution(start, end, solutions...)
		}
	}
	return receiving
}

// solutionsFor returns all the solutions that are reachable from the specified context
// by traversing all the contextSolutionsMaps which are passed as parameters
func solutionsFor(context *NonTerminal, maps ...contextSolutionsMap) (solutions []*NonTerminal, err error) {
	if context == nil {
		err = errors.New("context can NOT be nil")
		return
	}
	solutions = maps[0].solutionsFor(context.Position().Start(), context.Position().End())

	for i := 1; i < len(maps); i++ {
		tmpNodesToVisit := []*NonTerminal{}

		for len(solutions) > 0 {
			currentNode := solutions[0]
			solutions = solutions[1:]
			tmpNodesToVisit = append(tmpNodesToVisit, maps[i].solutionsFor(currentNode.Position().Start(), currentNode.Position().End())...)
		}
		solutions = tmpNodesToVisit
	}
	return
}
