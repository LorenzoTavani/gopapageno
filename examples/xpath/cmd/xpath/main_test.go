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

// cpu: AMD Ryzen 7 3750H with Radeon Vega Mobile Gfx

// Initial Results
// BenchmarkRun-8   	      18	  72582430 ns/op	123337611 B/op	  360566 allocs/op
// BenchmarkRun-8   	      27	  82007256 ns/op	123316399 B/op	  360566 allocs/op

// Remove interace for executionThreadList and use a slice instead of a list
// BenchmarkRun-8   	      28	  78592613 ns/op	122558957 B/op	  322765 allocs/op

// Use a swapback array instead of a list for speculations
// BenchmarkRun-8   	      22	  76039987 ns/op	122569769 B/op	  322766 allocs/op

// Remove interfaces from execution tables, use a swapbackArray data structure
// BenchmarkRun-8   	      30	  75965866 ns/op	120287893 B/op	  262283 allocs/op

// Remove interfaces of NonTerminal, ContextSolutions, and NUDPETable
// BenchmarkRun-8   	      19	  75595343 ns/op	122624059 B/op	  303864 allocs/op

// Fix: Do not take pointers of contextSolutionsMap
// BenchmarkRun-8   	      21	  74695023 ns/op	122467561 B/op	  284964 allocs/op

// Use parser pools for NonTerminals in parser
// BenchmarkRun-8   	      26	  67675224 ns/op	84245962 B/op	  274962 allocs/op

// Setting a preallocation size of the pool relative to input size
// BenchmarkRun-8   	      24	  74176000 ns/op	123574721 B/op	  230700 allocs/op

// cpu: Intel(R) Core(TM) i7-7820HQ CPU @ 2.90GHz
// BenchmarkRun-8   	      36	  32285544 ns/op	121926649 B/op	  230699 allocs/op

// Remove the executionTable abstraction
// BenchmarkRun-8   	      30	  35824162 ns/op	122477624 B/op	  253381 allocs/op

// Working query execution after refactoring
// BenchmarkRun-8   	       8	 127442388 ns/op	124818912 B/op	  310091 allocs/op

// Remove execution thread update logging
// BenchmarkRun-8   	      10	 118023990 ns/op	122643165 B/op	  219362 allocs/op
// Benchmark with reduction handling and resutls collection skipped completely, for comparison.
// BenchmarkRun-8   	      10	 107254672 ns/op	117773687 B/op	   90842 allocs/op

// Working main tests, predicates and a query parser generated with gopapageno
// BenchmarkRun-8   	      28	  39542104 ns/op	133530844 B/op	  227109 allocs/op

// TODO(vvihorev): benchmark against the original implementation
// TODO(vvihorev): use an arena allocator to reduce malloc calls and GC at runtime

func BenchmarkRun(b *testing.B) {
	bytes, err := os.ReadFile("../../data/1MB.xml")
	if err != nil {
		return
	}
	r := gopapageno.NewRunner(
		xpath.NewLexer(),
		xpath.NewGrammar(),
		gopapageno.WithConcurrency(1),
	)

	for i := 0; i < b.N; i++ {
		cmd := x.Execute("//PS_PARTKEY/PS_SUPPKEY").Against(bytes).WithNumberOfThreads(1)
		_, err := cmd.Run(r)
		if err != nil {
			log.Fatal(fmt.Sprintf("%e", err))
		}
	}
}
