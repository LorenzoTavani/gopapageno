# Known issues

Open problems of the code, found while reviewing it and merging the two lines. 
The defects that were already fixed are not listed here: they are in the commit history (`git log`). Last updated: 2026-10-27.

Status legend:
- **confirmed**: reproduced with a test, a build or a measurement;
- **read**: confirmed by reading the code, not executed;
- **author-reported**: stated in a comment by the original author;
- **unverified**: suspected, needs a test.

## XPath extension (`ext/xpath`)

| # | Issue | Status |
|---|---|---|
| X1 | **Time and memory grow with the square of the number of results** collected in one sequence of siblings. `contextSolutionsMap.merge` copies all accumulated solutions at every reduction, and every intermediate node of the parse tree keeps its own copy. Measured with `//div`, one thread: 1500 results take 1.2 s; 2000 take 4.9 s and 800 MB; 3000 take 93 s and 1.8 GB (the machine, with 2.8 GiB of RAM, starts swapping). Queries that select few nodes, as the XPathMark ones, are not affected. The copy was already present in giornetta's version. A possible fix is to reuse the table of the left node, which the reduction consumes, instead of copying it. | confirmed |
| X2 | The query parser covers a **fragment of XPath**: downward and upward paths with boolean predicates over paths and attribute tests (`=` only). Not supported: positions (`[2]`), functions (`count`, `contains`), sibling axes, more than one predicate per step. | read |
| X3 | Relative multi-step paths in a predicate need a leading axis: write `[/a/b]`, not `[a/b]`. A single element name (`[a]`) works. | confirmed |
| X4 | Axes written with `::` (`child::`, `descendant::`, `parent::`, `ancestor::`) are not supported; the query is rejected with a lexing error. | confirmed |
| X5 | A path that mixes forward and reverse axes inside a predicate (`[//p\\div]`) is accepted by the parser, but its evaluation is not verified. The original author notes that multiple nested path expressions (NUDPEs) are not supported. | unverified |
| X6 | In `reduction.go` (around the creation of a speculative thread) a speculation is added to a **copy** of the thread returned by value, not to the stored one. Not reproduced with the queries tried. | read, unverified |
| X7 | Query state is global (`udpeGlobalTable`, `nudpeGlobalTable`): only one query can be prepared at a time per process. | read |
| X8 | A `panic` raised inside a semantic action runs in a parser goroutine and terminates the process; it cannot be recovered by the caller. The cases found were fixed, others may exist (for example a predicate on `text()` asserts the wrong type). | confirmed, read |
| X9 | **The XPathMark benchmark cannot be reproduced from this repository.** `examples/xpath/data` has no `bench_small.xml` or `standard.xml` (the files `TestQuery`, `BenchmarkXPathMark` and the `Makefile` refer to). The bundled XML files are TPC-H-like tables, while the hand-built queries use XMark names. The test and the benchmark are skipped, with a message, when the file is missing. | confirmed |
| X10 | The performance figures in the comments of `examples/xpath/cmd/xpath/main_test.go` were taken on two different CPUs (about 72-82 ms per operation on a Ryzen 7, between 32 and 127 ms on a Core i7) and are not comparable. The timings of A6 and A7 were taken before the predicate trees of those two queries were fixed, so they measured a wrong predicate. | read |
| X11 | The hand-built queries A1, A4 and A5 look for the element `Text` (capital T), while the XMark schema and the author's own example `examples/xpath/data/testbug.xml` use `text`, so A1 finds nothing on that file. The name comes from the original XPath code (April 2024), so it is inherited. Whether the data that code was written for used `Text` is unknown. | confirmed (mismatch), cause unknown |
| X12 | Five white-box test files of `ext/xpath` (`contextSolutions`, `executionThread`, `fpe`, `rpe`, `speculation`; about 2000 lines) were written for the previous internal design and no longer compile. They are excluded with the build tag `wip` (`go test -tags wip ...` lists the errors). Internals are covered only through black-box tests. | confirmed |
| X15 | The same program disables the garbage collector unconditionally (`debug.SetGCPercent(-1)` and an unlimited memory limit, with a note that it is faster on a 30 MB file). Together with X1 this makes queries with many results exhaust the memory sooner. | read |

## Parsing core and generator

| # | Issue | Status |
|---|---|---|
| C1 | There are no tests for `opp.go`, `aopp.go`, `copp.go`, `copp_stack.go`, `runner.go` and `lexer.go`. | read |
| C2 | `opp.go` stops with `syntax error: no Yield precedence found on the stack` when the stack has no yield position; a `TODO` of the original author says he does not know in which case this happens in a correct run. A commented-out block at the end of the same function carries the note "catches an infinite loop on xpath, A1 query, testbug.xml": with the current code the text version of A1 terminates on that file with 1, 2, 4 and 8 threads (`TestTestbugDoesNotLoop`), so the loop was not reproduced. `copp.go` may have an analogous loop without the check. | author-reported, not reproduced |
| C3 | About twenty `panic()` calls in non-generated code (`stack.go`, `grammar.go`, `generator/grammar.go` and, above all, `ext/xpath`: `fpe.go`, `rpe.go`, `executor.go`, `reduction.go`) where an error could be returned. | read |
| C6 | `precedence_matrix.go`: the associativity heuristic for AOPP is declared incomplete by its author. | author-reported |
| C7 | Dead code in `parser_transforms.go` (`extractTerminalRules`, `addNewRules`, an empty `else`). | read |
| C8 | Semantic actions on renaming rules (`A -> B`) are removed by the generator, by construction (Fischer normal form). The generator prints `WARNING: semantic action is deleted`, also when the action only passed a value on, so the loss is harmless. | confirmed |
| C9 | `generator/regex` keeps its own copies of the stack and pool code (`stack.go`, `stackpool.go`, `listofstacks.go`, ...), and `generator/lexer.go` has a TODO that asks whether a hand-written regular expression parser should be built. | author-reported |
| C10 | `benchmark.Entry.Result` and `benchmark.RunExpect` are never used: the benchmarks do not check that the parsing result is the expected one. | confirmed |
| C11 | The `main.pg.go` programs produced by the generator take `-pf` with default `4`, while the help text and the README give the range `(0, 1]`, and they always run with the garbage collector disabled (`WithGarbageCollection(false)`). | read |

## Examples and tooling

| # | Issue | Status |
|---|---|---|
| E1 | `examples/polish/aopp/polish.g` creates an error with `fmt.Errorf` and discards it (reported by `go vet`). | confirmed |
| E2 | `scripts/filter_benchmarks.py` is not called by `scripts/evaluate.sh`; the Docker image name refers to giornetta's repository. | read |
| E3 | `go vet ./...` prints many `struct literal uses unkeyed fields` warnings for generated files; use `go vet -composites=false ./...`. | confirmed |
| E4 | The Go module path is provisional (`github.com/LorenzoTavani/gopapageno`) until the final location is decided. | |
| E5 | The reproducibility of the generator (same grammar, same output) was checked on a few examples only. All committed generated files are identical to the output of the current generator. | confirmed |
