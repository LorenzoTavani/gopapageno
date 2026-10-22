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

// The wildcard matches any element.
func TestWildcard(t *testing.T) {
	source := `<body><div></div><p></p></body>`
	got := runQuery(t, source, `/body/*`)
	want := []string{`<div></div>`, `<p></p>`}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// An axis written with '::' is not supported: it must be an error, not a
// silent miss.
func TestAxisSyntaxIsRejected(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("descendant:: should be rejected, but the query was accepted")
		}
	}()
	runQuery(t, `<body><div><b></b></div></body>`, `//div[descendant::b]`)
}

// A tag with two attributes can be matched on the second one.
func TestMultipleAttributes(t *testing.T) {
	source := `<body><div a="1" b="2"></div><div a="1"></div></body>`
	got := runQuery(t, source, `//div[@b="2"]`)
	want := []string{`<div a="1" b="2"></div>`}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// Measurement: a relative path in a predicate is written with a leading axis.
// A chain of or: the three matching divs, in document order.
func TestOrChain(t *testing.T) {
	source := `<body><div class="a"></div><div class="b"></div><div class="d"></div><div class="c"></div></body>`
	got := runQuery(t, source, `//div[@class="a" or @class="b" or @class="c"]`)
	want := []string{`<div class="a"></div>`, `<div class="b"></div>`, `<div class="c"></div>`}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// A chain of and with two negations: only the first div qualifies.
func TestAndChainWithNot(t *testing.T) {
	source := `<body><div class="x"></div><div class="y"></div></body>`
	got := runQuery(t, source, `//div[@class="x" and not @id and not @lang]`)
	want := []string{`<div class="x"></div>`}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// Measurement, not an assertion: path predicates with and without '//'
// before the predicate. Logs what the executor returns.
func TestPathPredicateMatrix(t *testing.T) {

	t.Skip("known limitation: path predicates return no results, see KNOWN_ISSUES.md")

	source := `<body><div><p><b></b></p></div><div><p></p></div></body>`
	for _, q := range []string{
		`/body/div[/p/b]`,
		`//div[/p/b]`,
		`/body/div[/p]`,
		`//div[/p]`,
	} {
		t.Logf("%-18s -> %v", q, runQuery(t, source, q))
	}
}
