package main

import (
	"fmt"
	"log"
	"testing"

	"github.com/LorenzoTavani/gopapageno"
	"github.com/LorenzoTavani/gopapageno/examples/xpath"
	x "github.com/LorenzoTavani/gopapageno/ext/xpath"
)

func ExpectResults(t *testing.T, source, query string, results []string) {
	r := gopapageno.NewRunner(
		xpath.NewLexer(),
		xpath.NewGrammar(),
		gopapageno.WithConcurrency(1),
	)

	// NOTE(vvihorev): enable verbose mode for debugging if a test fails
	cmd := x.Execute(query).Against([]byte(source)).WithNumberOfThreads(1) //.InVerboseMode()
	res, err := cmd.Run(r)

	if err != nil {
		log.Fatal(fmt.Sprintf("%e", err))
	}
	if len(res) != len(results) {
		for i := range res {
			t.Logf("Result %d: %s", i+1, string(source[res[i].Start():res[i].End()+1]))
		}
		t.Fatalf("result count does not match: %v", res)
	}
	for i := range results {
		if string(source[res[i].Start():res[i].End()+1]) != results[i] {
			t.Fatalf("result[%d] did not match, got: %v", i, string(source[res[0].Start():res[0].End()]))
		}
	}
}

func TestSingleFPEQueryExecution(t *testing.T) {
	ExpectResults(
		t,
		"<html><body></body></html>",
		"/body",
		[]string{"<body></body>"},
	)
}

func TestMultipleStepFPEQueryExecution(t *testing.T) {
	ExpectResults(
		t,
		"<html><body></body></html>",
		"/html/body",
		[]string{"<body></body>"},
	)
}

func TestSingleRPEQueryExecution(t *testing.T) {
	ExpectResults(
		t,
		"<html><body><div></div><div></div></body></html>",
		"\\\\body",
		[]string{"<body><div></div><div></div></body>"},
	)
}

func TestMultipleStepRPEQueryExecution(t *testing.T) {
	ExpectResults(
		t,
		"<html><body><div></div><div><p></p></div></body></html>",
		"\\\\\\\\div\\\\body",
		[]string{"<body><div></div><div><p></p></div></body>"},
	)
}

func TestNUDPEExpression(t *testing.T) {
	ExpectResults(
		t,
		`<html><body><div><p></p></div><div><p></p></div></body></html>`,
		"/html//p\\\\div",
		[]string{"<div><p></p></div>", "<div><p></p></div>"},
	)
}

func TestShouldMatchTwoParagraphs(t *testing.T) {
	ExpectResults(
		t,
		`<body><div><p></p></div><div><p></p></div></body>`,
		"/body//p",
		[]string{"<p></p>", "<p></p>"},
	)
}

func TestMatchAttribute(t *testing.T) {
	ExpectResults(
		t,
		`<body><div class="row"></div><div class="col"></div></body>`,
		`//div[@class="row"]`,
		[]string{`<div class="row"></div>`},
	)
}

func TestMatchPredicateAndClause(t *testing.T) {
	ExpectResults(
		t,
		`<body><div class="row"></div><div class="col"><p></p></div></body>`,
		`//div[@class and @class="row"]`,
		[]string{`<div class="row"></div>`},
	)
}

// NOTE(vvihorev): this test is flaky because of results order
// func TestMatchPredicateOrClause(t *testing.T) {
// 	ExpectResults(
// 		t,
// 		`<body><div class="row"></div><div class="col"><p></p></div></body>`,
// 		`//div[@class="col" or @class="row"]`,
// 		[]string{`<div class="col"><p></p></div>`, `<div class="row"></div>`},
// 	)
// }

func TestMatchPredicateNotClause(t *testing.T) {
	ExpectResults(
		t,
		`<body><div class="row"></div><div class="col"><p></p></div></body>`,
		`//div[@class="col" or not @class="row"]`,
		[]string{`<div class="col"><p></p></div>`},
	)
}

// NOTE(vvihorev): multiple NUDPEs are not supported by the query parser
// func TestNUDPEInPredicate(t *testing.T) {
// 	ExpectResults(
// 		t,
// 		`<body><div class="row"></div><div class="col"><p></p></div></body>`,
// 		`/body[//p\\div]`,
// 		[]string{`<body><div class="row"></div><div class="col"><p></p></div></body>`},
// 	)
// }
