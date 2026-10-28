package generator

import (
	"testing"
)

func TestGetIdentifier(t *testing.T) {
	for _, c := range []struct{ input, want string }{
		{"Expr rest", "Expr"},
		{"E1 rest", "E1"},
		{"term_2x+", "term_2x"},
		{"_a9 b", "_a9"},
		{"9abc", ""}, // an identifier cannot start with a digit
	} {
		pos := 0
		if got := getIdentifier(c.input, &pos); got != c.want {
			t.Errorf("getIdentifier(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}
