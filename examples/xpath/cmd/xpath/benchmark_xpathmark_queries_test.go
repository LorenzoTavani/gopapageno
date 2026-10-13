package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/LorenzoTavani/gopapageno"
	"github.com/LorenzoTavani/gopapageno/examples/xpath"
	x "github.com/LorenzoTavani/gopapageno/ext/xpath"
)

func BenchmarkAll(b *testing.B) {
	bytes, err := os.ReadFile("../../data/bench_small.xml")
	if err != nil {
		return
	}

	for numThreads := 1; numThreads < 3; numThreads++ {
	// for numThreads := 1; numThreads < 8; numThreads++ {
		b.Run(fmt.Sprintf("threads=%d", numThreads), func(b *testing.B) {
			for _, queryCode := range []string{
				"A1",
				"A2",
				"A3",
				"A4",
				"A5",
				"A6",
				"A7",
				"A8",
				"B1",
				"B2",
			} {
				b.Run(fmt.Sprintf("query=%s", queryCode), func(b *testing.B) {
					r := gopapageno.NewRunner(
						xpath.NewLexer(),
						xpath.NewGrammar(),
						gopapageno.WithConcurrency(numThreads),
					)
					for i := 0; i < b.N; i++ {
						b.StopTimer()
						exec := x.PrepareBenchmark(r, bytes, 1)
						exec.LoadQuery(queryCode)
						b.StartTimer()
						res := exec.ExecuteBenchmark()
						if len(res) == 0 {
							//			b.Logf("empty results")
						}
					}
				})
			}
		})
	}
}
