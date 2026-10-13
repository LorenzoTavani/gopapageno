package main

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/LorenzoTavani/gopapageno"
	"github.com/LorenzoTavani/gopapageno/examples/xpath"
	x "github.com/LorenzoTavani/gopapageno/ext/xpath"
)

func BenchmarkXPathMark(b *testing.B) {
	bytes, err := os.ReadFile("../../data/bench_small.xml")
	if err != nil {
		return
	}
	r := gopapageno.NewRunner(
		xpath.NewLexer(),
		xpath.NewGrammar(),
		gopapageno.WithConcurrency(1),
	)

	// NOTE(vvihorev): query parser is broken
	queries := []string{
		`/site/closed_auctions/closed_auction/annotation/description/text/keyword`,
		`//closed_auction//keyword`,
		`/site/closed_auctions/closed_auction//keyword`,
		`/site/closed_auctions/closed_auction[annotation/description/text/keyword]/date`,
		`/site/closed_auctions/closed_auction[descendant::keyword]/date`,
		`/site/people/person[profile/gender and profile/age]/name`,
		`/site/people/person[phone or homepage]/name`,
		`/site/people/person[address and (phone or homepage) and (creditcard or profile)]/name`,
		`/site/regions/*/item[parent::namerica or parent::samerica]/name`,
	}

	for _, query := range queries {
		b.Run(fmt.Sprintf("query=%s", query), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				cmd := x.Execute(query).Against(bytes).WithNumberOfThreads(1)
				res, err := cmd.Run(r)
				if err != nil {
					log.Fatal(fmt.Sprintf("%e", err))
				}
				if len(res) == 0 {
					b.Fatalf("no matches found")
				}
			}
		})
	}
}

func TestQuery(t *testing.T) {
	bytes, err := os.ReadFile("../../data/bench_small.xml")
	if err != nil {
		return
	}
	r := gopapageno.NewRunner(
		xpath.NewLexer(),
		xpath.NewGrammar(),
		gopapageno.WithConcurrency(1),
	)

	query := (
	// `/site/closed_auctions/closed_auction[/annotation/description/text/keyword]/date`)
		`/site/closed_auctions[closed_auction/annotation/description/text/keyword]`)
	cmd := x.Execute(query).Against(bytes).WithNumberOfThreads(1)
	res, err := cmd.Run(r)
	if err != nil {
		log.Fatal(fmt.Sprintf("%e", err))
	}
	t.Fatalf("Found %d matches", len(res))
	if len(res) == 0 {
		t.Fatalf("no matches found")
	}
}
