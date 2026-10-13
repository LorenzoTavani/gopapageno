package generator

import (
	"fmt"
	"strings"
	"github.com/LorenzoTavani/gopapageno"
	"slices"
)

// rulesDictionary is a data structure used to store unique RHS -> LHS mappings.
// It is useful to remove repeated RHS productions.
type rulesDictionary struct {
	KeysRHS [][]string

	ValuesLHS  []*set[string]
	SemActions []*string
	Flags      []gopapageno.RuleFlags
	Prefixes   [][][]string
}

func newRulesDictionary(capacity int) *rulesDictionary {
	return &rulesDictionary{
		KeysRHS:    make([][]string, 0, capacity),
		ValuesLHS:  make([]*set[string], 0, capacity),
		SemActions: make([]*string, 0, capacity),
		Flags:      make([]gopapageno.RuleFlags, 0, capacity),
		Prefixes:   make([][][]string, 0, capacity),
	}
}

// Used for debugging purposes only
func (d *rulesDictionary) String() string {
	sb := strings.Builder{}
	for i, v := range d.ValuesLHS {
		vx := make([]string, 0)
		for _, v := range v.Iter {
			vx = append(vx, v)
		}
		sb.WriteString(fmt.Sprintf("rule: (%v) %v <- %v\n", (*d.SemActions[i]), d.KeysRHS[i], vx))
	}
	return sb.String()
}

func (d *rulesDictionary) Add(r *ruleDescription) {
	for i, keyRhs := range d.KeysRHS {
		if slices.Equal(keyRhs, r.RHS) {
			d.ValuesLHS[i].Add(r.LHS)

			return
		}
	}

	d.KeysRHS = append(d.KeysRHS, r.RHS)

	d.ValuesLHS = append(d.ValuesLHS, newSet[string]())
	d.ValuesLHS[len(d.ValuesLHS)-1].Add(r.LHS)

	d.SemActions = append(d.SemActions, &r.Action)
	d.Flags = append(d.Flags, r.Flags)

	d.Prefixes = append(d.Prefixes, r.Prefixes)
}

func (d *rulesDictionary) Remove(rhs []string) {
	for i, curKeyRHS := range d.KeysRHS {
		if slices.Equal(curKeyRHS, rhs) {
			d.KeysRHS = append(d.KeysRHS[:i], d.KeysRHS[i+1:]...)

			d.ValuesLHS = append(d.ValuesLHS[:i], d.ValuesLHS[i+1:]...)
			d.SemActions = append(d.SemActions[:i], d.SemActions[i+1:]...)
			d.Flags = append(d.Flags[:i], d.Flags[i+1:]...)
			d.Prefixes = append(d.Prefixes[:i], d.Prefixes[i+1:]...)
		}
	}
}

func (d *rulesDictionary) Len() int {
	return len(d.KeysRHS)
}

func (d *rulesDictionary) Copy() *rulesDictionary {
	newDict := newRulesDictionary(d.Len())

	for i, _ := range d.KeysRHS {
		newDict.KeysRHS = append(newDict.KeysRHS, make([]string, len(d.KeysRHS[i])))
		copy(newDict.KeysRHS[i], d.KeysRHS[i])

		newDict.ValuesLHS = append(newDict.ValuesLHS, d.ValuesLHS[i].Copy())
		newDict.SemActions = append(newDict.SemActions, d.SemActions[i])
		newDict.Flags = append(newDict.Flags, d.Flags[i])
		newDict.Prefixes = append(newDict.Prefixes, d.Prefixes[i])
	}

	return newDict
}

func (d *rulesDictionary) LHSSets() []*set[string] {
	valueLHSSets := make([]*set[string], 0)

	for _, curLHSSet := range d.ValuesLHS {
		alreadyContained := false
		for _, curSet := range valueLHSSets {
			if curSet.Equals(curLHSSet) {
				alreadyContained = true
				break
			}
		}
		if !alreadyContained {
			valueLHSSets = append(valueLHSSets, curLHSSet)
		}
	}

	return valueLHSSets
}
