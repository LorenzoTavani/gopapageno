package generator

import (
	"fmt"
	"strings"
	"github.com/LorenzoTavani/gopapageno"
)

func LogPrecedenceConflict(p gopapageno.Precedence, c conflict) {
	sb := strings.Builder{}
	for _ = range len(c.rule.LHS) + 4 {
		sb.WriteString(" ")
	}

	for i, symbol := range c.rule.RHS {
		pad := len(symbol) + 1

		if i == c.i || i == c.j {
			sb.WriteString("^")
			pad--;
		}

		for _ = range pad {
			sb.WriteString(" ")
		}
	}
	sb.WriteString(fmt.Sprintf("%v precedence conflict between terminals", p))

	fmt.Printf("%v : %v\n", c.rule.LHS, c.rule.RHS)
	fmt.Printf("%v\n", sb.String())
}
