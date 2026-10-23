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

// runHardcoded runs one of the XPathMark queries that are built by hand in the
// executor, instead of parsing a query string.
func runHardcoded(t *testing.T, source, code string) []string {
	t.Helper()
	r := gopapageno.NewRunner(
		xpath.NewLexer(),
		xpath.NewGrammar(),
		gopapageno.WithConcurrency(1),
	)
	exec := x.PrepareBenchmark(r, []byte(source), 1)
	if err := exec.LoadQuery(code); err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, p := range exec.ExecuteBenchmark() {
		out = append(out, source[p.Start():p.End()+1])
	}
	return out
}

// The XPathMark query A4, built by hand in the executor, evaluates a path
// predicate correctly: the engine is fine.
func TestHandBuiltPathPredicate(t *testing.T) {
	source := `<site><closed_auctions>` +
		`<closed_auction><annotation><description><Text><keyword></keyword></Text></description></annotation><date>one</date></closed_auction>` +
		`<closed_auction><annotation><description><Text></Text></description></annotation><date>two</date></closed_auction>` +
		`</closed_auctions></site>`
	got := runHardcoded(t, source, "A4")
	want := []string{`<date>one</date>`}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// Paths in predicates start at the element that carries the predicate.
func TestPathPredicates(t *testing.T) {
	source := `<body><div><p><b></b></p></div><div><p></p></div></body>`
	div1 := `<div><p><b></b></p></div>`
	div2 := `<div><p></p></div>`
	for _, c := range []struct {
		query string
		want  []string
	}{
		{`/body/div[/p/b]`, []string{div1}},
		{`//div[/p/b]`, []string{div1}},
		{`//div[/p]`, []string{div1, div2}},
		{`//div[p]`, []string{div1, div2}},
		{`//div[b]`, nil},
		{`//div[/p/b or /p/c]`, []string{div1}},
		{`//div[not /p/b]`, []string{div2}},
	} {
		got := runQuery(t, source, c.query)
		if fmt.Sprint(got) != fmt.Sprint(c.want) {
			t.Errorf("%s: got %v, want %v", c.query, got, c.want)
		}
	}
}

// The text versions of the XPathMark queries A4 and A6 give the same result as
// the queries built by hand in the executor.
func TestTextQueriesEqualHandBuilt(t *testing.T) {
	a4 := `<site><closed_auctions>` +
		`<closed_auction><annotation><description><Text><keyword></keyword></Text></description></annotation><date>one</date></closed_auction>` +
		`<closed_auction><annotation><description><Text></Text></description></annotation><date>two</date></closed_auction>` +
		`</closed_auctions></site>`
	a6 := `<site><people>` +
		`<person><profile><gender>m</gender><age>3</age></profile><name>ann</name></person>` +
		`<person><profile><gender>f</gender></profile><name>bob</name></person>` +
		`</people></site>`
	a7 := `<site><people>` +
		`<person><phone>1</phone><name>ann</name></person>` +
		`<person><homepage>h</homepage><name>bob</name></person>` +
		`<person><name>cid</name></person>` +
		`</people></site>`
	for _, c := range []struct {
		name, source, code, query string
		want                      []string
	}{
		{"A4", a4, "A4", `/site/closed_auctions/closed_auction[/annotation/description/Text/keyword]/date`, []string{`<date>one</date>`}},
		{"A6", a6, "A6", `/site/people/person[/profile/gender and /profile/age]/name`, []string{`<name>ann</name>`}},
		{"A7", a7, "A7", `/site/people/person[phone or homepage]/name`, []string{`<name>ann</name>`, `<name>bob</name>`}},
	} {
		hand := runHardcoded(t, c.source, c.code)
		text := runQuery(t, c.source, c.query)
		if fmt.Sprint(hand) != fmt.Sprint(c.want) || fmt.Sprint(text) != fmt.Sprint(c.want) {
			t.Errorf("%s: hand-built %v, text %v, want %v", c.name, hand, text, c.want)
		}
	}
}
