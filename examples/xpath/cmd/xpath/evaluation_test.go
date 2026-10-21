package main

import (
	"fmt"
	"sort"
	"testing"

	"github.com/LorenzoTavani/gopapageno"
	"github.com/LorenzoTavani/gopapageno/examples/xpath"
	x "github.com/LorenzoTavani/gopapageno/ext/xpath"
)

// runQuery runs a query and returns the matched fragments in the order
// the executor returned them.
func runQuery(t *testing.T, source, query string) []string {
	t.Helper()
	r := gopapageno.NewRunner(
		xpath.NewLexer(),
		xpath.NewGrammar(),
		gopapageno.WithConcurrency(1),
	)
	res, err := x.Execute(query).Against([]byte(source)).WithNumberOfThreads(1).Run(r)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, p := range res {
		out = append(out, source[p.Start():p.End()+1])
	}
	return out
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sa := append([]string(nil), a...)
	sb := append([]string(nil), b...)
	sort.Strings(sa)
	sort.Strings(sb)
	for i := range sa {
		if sa[i] != sb[i] {
			return false
		}
	}
	return true
}

// The results must always come back in document order.
func TestResultsAreInDocumentOrder(t *testing.T) {
	source := `<body><div class="row"></div><div class="col"><p></p></div></body>`
	want := []string{`<div class="row"></div>`, `<div class="col"><p></p></div>`}
	for i := 0; i < 200; i++ {
		got := runQuery(t, source, `//div[@class]`)
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("run %d: got %v, want %v", i, got, want)
		}
	}
}

// Hypothesis 2: a predicate attached after '//' is lost when the path
// pattern branches (nested divs).
func TestPredicateOnNestedDivs(t *testing.T) {
	source := `<body><div class="row"><div class="col"></div></div></body>`
	got := runQuery(t, source, `//div[@class="col"]`)
	want := []string{`<div class="col"></div>`}
	if !sameSet(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
