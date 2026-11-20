package gopapageno

import "testing"

// dfaState returns a state of a lexer automaton with the given transitions;
// every other character has no transition.
func dfaState(final bool, transitions map[byte]int) LexerDFAState {
	var s LexerDFAState
	for i := range s.Transitions {
		s.Transitions[i] = -1
	}
	for c, next := range transitions {
		s.Transitions[c] = next
	}
	s.IsFinal = final
	return s
}

// TestCutPointsFollowTheCutAutomaton checks that a cut point is placed only where
// the whole %cut pattern matches. The pattern here is "ab", two characters. The
// automaton of the tokens has a final state at the same index that the cut
// automaton reaches after "a": a search that moves to the automaton of the tokens
// after the first character stops at the lone "a".
func TestCutPointsFollowTheCutAutomaton(t *testing.T) {
	lexer := &Lexer{
		Automaton: LexerDFA{
			dfaState(false, nil),
			dfaState(true, nil),
			dfaState(false, nil),
		},
		CutPointsAutomaton: LexerDFA{
			dfaState(false, map[byte]int{'a': 1}),
			dfaState(false, map[byte]int{'b': 2}),
			dfaState(true, nil),
		},
	}

	// With two threads the search starts at the middle, index 10, where there is
	// a lone "a"; the pattern "ab" starts at index 12.
	source := "xxxxxxxxxx" + "ax" + "ab" + "xxxxxx"
	s := &Scanner{Lexer: lexer, source: []byte(source)}

	cutPoints, concurrency := s.findCutPoints(2)

	want := []int{0, 12, len(source)}
	if concurrency != 2 || len(cutPoints) != len(want) || cutPoints[1] != want[1] || cutPoints[2] != want[2] {
		t.Errorf("findCutPoints(2) = %v, %d; want %v, 2", cutPoints, concurrency, want)
	}
}
