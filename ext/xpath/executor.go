package xpath

import (
	"context"
	"fmt"
	"log"
	"os"

	"runtime/debug"
	"math"

	"github.com/LorenzoTavani/gopapageno"
)

const defaultExecutorNumberOfThreads = 1

// Singletons
var nudpeGlobalTable *globalNudpeTable
var udpeGlobalTable *globalUdpeTable
var logger Logger
var DEBUG = false

func init() {
	logger = newNopLogger()
}

type mainQueryType int

const (
	UDPE mainQueryType = iota
	NUDPE
)

type executor struct {
	numberOfThreads         int
	mainQueryType           mainQueryType
	resultingExecutionTable executionTable

	source []byte
	runner *gopapageno.Runner
}

// ExecutorCommand represents a command that can be made by a
// client to execute a XPath query
type ExecutorCommand struct {
	xpathQuery      string
	source          []byte
	numberOfThreads int
}

// Execute specify the XPath query to be executed
func Execute(xpathQuery string) *ExecutorCommand {
	DEBUG = false
	logger = newNopLogger()
	return &ExecutorCommand{
		xpathQuery: xpathQuery,
	}
}

func (executorCommand *ExecutorCommand) WithNumberOfThreads(numberOfThreads int) *ExecutorCommand {
	executorCommand.numberOfThreads = numberOfThreads
	return executorCommand
}

func (executorCommand *ExecutorCommand) InVerboseMode() *ExecutorCommand {
	DEBUG = true
	logger = log.New(os.Stderr, "", log.LstdFlags)
	return executorCommand
}

func (executorCommand *ExecutorCommand) Against(source []byte) *ExecutorCommand {
	executorCommand.source = source
	return executorCommand
}

// Run takes care of executing the command and to return
func (executorCommand *ExecutorCommand) Run(runner *gopapageno.Runner) (results []Position, err error) {
	executor := &executor{
		numberOfThreads: defaultExecutorNumberOfThreads,
		runner:          runner,
	}

	executor.numberOfThreads = executorCommand.numberOfThreads
	executor.source = executorCommand.source

	executor.parseQuery(executorCommand.xpathQuery)
	if DEBUG {
		logger.Printf("EXECUTING QUERY: %v", executorCommand.xpathQuery)
	}

	err = executor.executeUDPEsWhileParsing()
	if err != nil {
		return
	}

	err = executor.completeExecutionOfUDPEsAndNUDPEs()
	if err != nil {
		return
	}

	results = executor.retrieveResults()

	return
}

func (executor *executor) parseQuery(xpathQuery string) {
	executor.mainQueryType = parse(xpathQuery)
}

func (executor *executor) executeUDPEsWhileParsing() error {
	axiomToken, err := executor.runner.Run(context.Background(), executor.source)
	if err != nil {
		return fmt.Errorf("could not parse: %v", err)
	}
	executor.resultingExecutionTable = axiomToken.Value.(*NonTerminal).executionTable

	return err
}

func (executor *executor) completeExecutionOfUDPEsAndNUDPEs() (err error) {
	if numberOfNUDPEs := nudpeGlobalTable.size(); numberOfNUDPEs == 0 {
		return
	}

	var currentNudpeRecord *globalNudpeRecord
	var currentNudpeContextSolutionsMaps []contextSolutionsMap

	for _, er := range executor.resultingExecutionTable.records {
		if er.gNudpeRecord != nil {
			if er.gNudpeRecord != currentNudpeRecord {
				if currentNudpeRecord != nil {
					currentNudpeRecord.setContextSolutions(transitiveClosure(currentNudpeContextSolutionsMaps))
				}
				currentNudpeRecord = er.gNudpeRecord
				currentNudpeContextSolutionsMaps = []contextSolutionsMap{}
			}
		} else {
			if currentNudpeRecord != nil {
				currentNudpeRecord.setContextSolutions(transitiveClosure(currentNudpeContextSolutionsMaps))
				currentNudpeRecord = nil
				currentNudpeContextSolutionsMaps = nil
			}
		}

		// stop unfounded speculative execution threads
		for i := 0; i < er.threads.size; i++ {
			if areSpeculationsFounded := er.threads.array[i].checkAndUpdateSpeculations(executor.nudpeBooleanValueEvaluator); !areSpeculationsFounded {
				er.removeExecutionThreadAt(i)
				i--
			}
		}

		// produce context solutions out of completed non speculative execution threads
		for i := 0; i < er.threads.size; i++ {
			if er.threads.array[i].pp.isEmpty() && !er.threads.array[i].isSpeculative() {
				er.ctxSols.addContextSolution(er.threads.array[i].ctx.Position().Start(), er.threads.array[i].ctx.Position().End(), er.threads.array[i].sol)
				er.removeExecutionThreadAt(i)
				i--
			}
		}

		if er.gNudpeRecord != nil {
			currentNudpeContextSolutionsMaps = append(currentNudpeContextSolutionsMaps, er.ctxSols)
		}
	}

	if currentNudpeRecord != nil {
		currentNudpeRecord.setContextSolutions(transitiveClosure(currentNudpeContextSolutionsMaps))
	}

	return
}

func (executor *executor) nudpeBooleanValueEvaluator(udpeID int, context *NonTerminal, evaluationsCount int) customBool {
	record, err := executor.resultingExecutionTable.recordByID(udpeID)

	if err != nil {
		panic(fmt.Sprintf(`cannot retrieve udpe record from resulting execution table: %v`, err))
	}

	if record.gNudpeRecord == nil {
		panic("cannot evaluate nudpe boolean value if first udpe does NOT belong to a nudpe")
	}

	nudpeGlobalRecord := record.gNudpeRecord
	return toCustomBool(nudpeGlobalRecord.hasSolutionsFor(context))
}

func (executor *executor) retrieveResults() (results []Position) {
	if DEBUG {
		logger.Printf("RETRIEVE RESULTS")
		for _, er := range executor.resultingExecutionTable.records {
			logger.Printf(" results record: %v", er.String())
			for i := 0; i < er.threads.size; i++ {
				logger.Printf("  resulting thread: %v", er.threads.array[i].String())
			}
		}
	}

	var potentiallyDuplicatedResults []Position
	switch executor.mainQueryType {
	case UDPE:
		mainQueryUDPERecord := executor.resultingExecutionTable.mainQueryRecord()
		potentiallyDuplicatedResults = mainQueryUDPERecord.ctxSols.convertToGroupOfSolutionsPositions()
	case NUDPE:
		mainQueryNUDPERecord := nudpeGlobalTable.mainQueryRecord()
		potentiallyDuplicatedResults = mainQueryNUDPERecord.contextSolutions().convertToGroupOfSolutionsPositions()
	default:
		panic("retrieving results error: unknown main query type")
	}

	var uniqueResults = make(map[string]bool, len(potentiallyDuplicatedResults))

	for _, result := range potentiallyDuplicatedResults {
		resultIdentifier := fmt.Sprintf("(%v,%v)", result.Start(), result.End())
		_, isAlreadyConsidered := uniqueResults[resultIdentifier]
		if !isAlreadyConsidered {
			results = append(results, result)
			uniqueResults[resultIdentifier] = true
		}
	}

	return
}


func PrepareBenchmark(runner *gopapageno.Runner, source []byte, numberOfThreads int) *executor {
	executor := &executor{
		numberOfThreads: numberOfThreads,
		runner:          runner,
		source: source,
	}

	udpeGlobalTable = new(globalUdpeTable)
	nudpeGlobalTable = new(globalNudpeTable)

	return executor
}
 
func (executor *executor) ExecuteBenchmark() []Position {
	debug.SetGCPercent(-1)
	debug.SetMemoryLimit(math.MaxInt64)
	err := executor.executeUDPEsWhileParsing()
	if err != nil {
		return nil
	}

	err = executor.completeExecutionOfUDPEsAndNUDPEs()
	if err != nil {
		return nil
	}

	results := executor.retrieveResults()
	return results
}

func (executor *executor) LoadQuery(queryCode string) (err error) {
	switch queryCode {
	case "A1":
		executor.A1()
	case "A2":
		executor.A2()
	case "A3":
		executor.A3()
	case "A4":
		executor.A4()
	case "A5":
		executor.A5()
	case "A6":
		executor.A6()
	case "A7":
		executor.A7()
	case "A8":
		executor.A8()
	case "B1":
		executor.B1()
	case "B2":
		executor.B2()
	default:
		return fmt.Errorf("unknown query code: %s", queryCode)
	}
	return nil
}

func (executor *executor) A1() {
	executor.mainQueryType = UDPE
	/* /site/closed_auctions/closed_auction/annotation/description/Text/keyword */

	fpeBuilder1 := fpeBuilder{}
	fpeBuilder1.addAxis(child)
	fpeBuilder1.addUdpeTest(newElementTest("site", nil, nil))
	fpeBuilder1.addAxis(child)
	fpeBuilder1.addUdpeTest(newElementTest("closed_auctions", nil, nil))
	fpeBuilder1.addAxis(child)
	fpeBuilder1.addUdpeTest(newElementTest("closed_auction", nil, nil))
	fpeBuilder1.addAxis(child)
	fpeBuilder1.addUdpeTest(newElementTest("annotation", nil, nil))
	fpeBuilder1.addAxis(child)
	fpeBuilder1.addUdpeTest(newElementTest("description", nil, nil))
	fpeBuilder1.addAxis(child)
	fpeBuilder1.addUdpeTest(newElementTest("Text", nil, nil))
	fpeBuilder1.addAxis(child)
	fpeBuilder1.addUdpeTest(newElementTest("keyword", nil, nil))

	udpeGlobalTable.addFpe(fpeBuilder1.end())
}

func (executor *executor) A2() {
	executor.mainQueryType = UDPE
	/* //closed_auction//keyword */

	fpeBuilder1 := fpeBuilder{}
	fpeBuilder1.addAxis(descendantOrSelf)
	fpeBuilder1.addUdpeTest(newElementTest("closed_auction", nil, nil))
	fpeBuilder1.addAxis(descendantOrSelf)
	fpeBuilder1.addUdpeTest(newElementTest("keyword", nil, nil))

	udpeGlobalTable.addFpe(fpeBuilder1.end())
}

func (executor *executor) A3() {
	executor.mainQueryType = UDPE
	/* /site/closed_auctions/closed_auction//keyword */

	fpeBuilder1 := fpeBuilder{}
	fpeBuilder1.addAxis(child)
	fpeBuilder1.addUdpeTest(newElementTest("site", nil, nil))
	fpeBuilder1.addAxis(child)
	fpeBuilder1.addUdpeTest(newElementTest("closed_auctions", nil, nil))
	fpeBuilder1.addAxis(child)
	fpeBuilder1.addUdpeTest(newElementTest("closed_auction", nil, nil))
	fpeBuilder1.addAxis(descendantOrSelf)
	fpeBuilder1.addUdpeTest(newElementTest("keyword", nil, nil))

	udpeGlobalTable.addFpe(fpeBuilder1.end())
}

func (executor *executor) A4() {
	executor.mainQueryType = UDPE

	/* p = /closed_auction/annotation/description/Text/keyword */

	fpeBuilder1 := fpeBuilder{}
	fpeBuilder1.addAxis(child)
	fpeBuilder1.addUdpeTest(newElementTest("closed_auction", nil, nil))
	fpeBuilder1.addAxis(child)
	fpeBuilder1.addUdpeTest(newElementTest("annotation", nil, nil))
	fpeBuilder1.addAxis(child)
	fpeBuilder1.addUdpeTest(newElementTest("description", nil, nil))
	fpeBuilder1.addAxis(child)
	fpeBuilder1.addUdpeTest(newElementTest("Text", nil, nil))
	fpeBuilder1.addAxis(child)
	fpeBuilder1.addUdpeTest(newElementTest("keyword", nil, nil))

	fpeID, _ := udpeGlobalTable.addFpe(fpeBuilder1.end())

	node := predNode{op: atom()}
	p := &predicate{
		root: &node,
		undoneAtoms: map[int]*predNode{fpeID: &node},
	}

	/* /site/closed_auctions/closed_auction[p]/date */

	fpeBuilder2 := fpeBuilder{}
	fpeBuilder2.addAxis(child)
	fpeBuilder2.addUdpeTest(newElementTest("site", nil, nil))
	fpeBuilder2.addAxis(child)
	fpeBuilder2.addUdpeTest(newElementTest("closed_auctions", nil, nil))
	fpeBuilder2.addAxis(child)
	fpeBuilder2.addUdpeTest(newElementTest("closed_auction", nil, p))
	fpeBuilder2.addAxis(child)
	fpeBuilder2.addUdpeTest(newElementTest("date", nil, nil))

	udpeGlobalTable.addFpe(fpeBuilder2.end())
}

func (executor *executor) A5() {
	executor.mainQueryType = UDPE

	/* p = closed_auction//keyword */
	fpeBuilder1 := fpeBuilder{}
	fpeBuilder1.addAxis(child)
	fpeBuilder1.addUdpeTest(newElementTest("closed_auction", nil, nil))
	fpeBuilder1.addAxis(descendantOrSelf)
	fpeBuilder1.addUdpeTest(newElementTest("keyword", nil, nil))

	fpeID, _ := udpeGlobalTable.addFpe(fpeBuilder1.end())

	node := predNode{op: atom()}
	p := &predicate{
		root: &node,
		undoneAtoms: map[int]*predNode{fpeID: &node},
	}

	/* /site/closed_auctions/closed_auction[p]/date */
	fpeBuilder2 := fpeBuilder{}
	fpeBuilder2.addAxis(child)
	fpeBuilder2.addUdpeTest(newElementTest("site", nil, nil))
	fpeBuilder2.addAxis(child)
	fpeBuilder2.addUdpeTest(newElementTest("closed_auctions", nil, nil))
	fpeBuilder2.addAxis(child)
	fpeBuilder2.addUdpeTest(newElementTest("closed_auction", nil, p))
	fpeBuilder2.addAxis(child)
	fpeBuilder2.addUdpeTest(newElementTest("date", nil, nil))

	udpeGlobalTable.addFpe(fpeBuilder2.end())
}

func (executor *executor) A6() {
	executor.mainQueryType = UDPE

	/* /person/profile/gender */
	fpeBuilder1 := fpeBuilder{}
	fpeBuilder1.addAxis(child)
	fpeBuilder1.addUdpeTest(newElementTest("person", nil, nil))
	fpeBuilder1.addAxis(child)
	fpeBuilder1.addUdpeTest(newElementTest("profile", nil, nil))
	fpeBuilder1.addAxis(child)
	fpeBuilder1.addUdpeTest(newElementTest("gender", nil, nil))

	/* /person/profile/age */
	fpeBuilder2 := fpeBuilder{}
	fpeBuilder2.addAxis(child)
	fpeBuilder2.addUdpeTest(newElementTest("person", nil, nil))
	fpeBuilder2.addAxis(child)
	fpeBuilder2.addUdpeTest(newElementTest("profile", nil, nil))
	fpeBuilder2.addAxis(child)
	fpeBuilder2.addUdpeTest(newElementTest("age", nil, nil))

	fpe1ID, _ := udpeGlobalTable.addFpe(fpeBuilder1.end())
	fpe2ID, _ := udpeGlobalTable.addFpe(fpeBuilder2.end())

	node1 := predNode{op: atom()}
	node2 := predNode{op: atom()}
	nodeAnd := predNode{op: and()}
	nodeAnd.left = &node1
	nodeAnd.left = &node2
	node1.parent = &nodeAnd
	node2.parent = &nodeAnd
	p := &predicate{
		root: &nodeAnd,
		undoneAtoms: map[int]*predNode{fpe1ID: &node1, fpe2ID: &node2},
	}

	/* /site/people/person[p]/name */
	fpebuilder3 := fpeBuilder{}
	fpebuilder3.addAxis(child)
	fpebuilder3.addUdpeTest(newElementTest("site", nil, nil))
	fpebuilder3.addAxis(child)
	fpebuilder3.addUdpeTest(newElementTest("people", nil, nil))
	fpebuilder3.addAxis(child)
	fpebuilder3.addUdpeTest(newElementTest("person", nil, p))
	fpebuilder3.addAxis(child)
	fpebuilder3.addUdpeTest(newElementTest("name", nil, nil))

	udpeGlobalTable.addFpe(fpebuilder3.end())
}

func (executor *executor) A7() {
	executor.mainQueryType = UDPE

	/* /person/phone */
	fpeBuilder1 := fpeBuilder{}
	fpeBuilder1.addAxis(child)
	fpeBuilder1.addUdpeTest(newElementTest("person", nil, nil))
	fpeBuilder1.addAxis(child)
	fpeBuilder1.addUdpeTest(newElementTest("phone", nil, nil))

	/* /person/homepage */
	fpeBuilder2 := fpeBuilder{}
	fpeBuilder2.addAxis(child)
	fpeBuilder2.addUdpeTest(newElementTest("person", nil, nil))
	fpeBuilder2.addAxis(child)
	fpeBuilder2.addUdpeTest(newElementTest("homepage", nil, nil))

	fpe1ID, _ := udpeGlobalTable.addFpe(fpeBuilder1.end())
	fpe2ID, _ := udpeGlobalTable.addFpe(fpeBuilder2.end())

	node1 := predNode{op: atom()}
	node2 := predNode{op: atom()}
	nodeOr := predNode{op: or()}
	nodeOr.left = &node1
	nodeOr.left = &node2
	node1.parent = &nodeOr
	node2.parent = &nodeOr
	p := &predicate{
		root: &nodeOr,
		undoneAtoms: map[int]*predNode{fpe1ID: &node1, fpe2ID: &node2},
	}

	/* /site/people/person[p]/name */
	fpebuilder3 := fpeBuilder{}
	fpebuilder3.addAxis(child)
	fpebuilder3.addUdpeTest(newElementTest("site", nil, nil))
	fpebuilder3.addAxis(child)
	fpebuilder3.addUdpeTest(newElementTest("people", nil, nil))
	fpebuilder3.addAxis(child)
	fpebuilder3.addUdpeTest(newElementTest("person", nil, p))
	fpebuilder3.addAxis(child)
	fpebuilder3.addUdpeTest(newElementTest("name", nil, nil))

	udpeGlobalTable.addFpe(fpebuilder3.end())

}

func (executor *executor) A8() {
	executor.mainQueryType = UDPE

	fpeBuilder1 := fpeBuilder{}
	fpeBuilder1.addAxis(child)
	fpeBuilder1.addUdpeTest(newElementTest("person", nil, nil))
	fpeBuilder1.addAxis(child)
	fpeBuilder1.addUdpeTest(newElementTest("address", nil, nil))

	fpeBuilder2 := fpeBuilder{}
	fpeBuilder2.addAxis(child)
	fpeBuilder2.addUdpeTest(newElementTest("person", nil, nil))
	fpeBuilder2.addAxis(child)
	fpeBuilder2.addUdpeTest(newElementTest("phone", nil, nil))

	fpeBuilder3 := fpeBuilder{}
	fpeBuilder3.addAxis(child)
	fpeBuilder3.addUdpeTest(newElementTest("person", nil, nil))
	fpeBuilder3.addAxis(child)
	fpeBuilder3.addUdpeTest(newElementTest("homepage", nil, nil))

	fpeBuilder4 := fpeBuilder{}
	fpeBuilder4.addAxis(child)
	fpeBuilder4.addUdpeTest(newElementTest("person", nil, nil))
	fpeBuilder4.addAxis(child)
	fpeBuilder4.addUdpeTest(newElementTest("creditcard", nil, nil))

	fpeBuilder5 := fpeBuilder{}
	fpeBuilder5.addAxis(child)
	fpeBuilder5.addUdpeTest(newElementTest("person", nil, nil))
	fpeBuilder5.addAxis(child)
	fpeBuilder5.addUdpeTest(newElementTest("profile", nil, nil))

	fpe1ID, _ := udpeGlobalTable.addFpe(fpeBuilder1.end())
	fpe2ID, _ := udpeGlobalTable.addFpe(fpeBuilder2.end())
	fpe3ID, _ := udpeGlobalTable.addFpe(fpeBuilder3.end())
	fpe4ID, _ := udpeGlobalTable.addFpe(fpeBuilder4.end())
	fpe5ID, _ := udpeGlobalTable.addFpe(fpeBuilder5.end())

	n1 := predNode{op: atom()}
	n2 := predNode{op: atom()}
	n3 := predNode{op: atom()}
	n4 := predNode{op: atom()}
	n5 := predNode{op: atom()}

	nAnd1 := predNode{op: and()}
	nAnd2 := predNode{op: and()}
	nOr1 := predNode{op: or()}
	nOr2 := predNode{op: or()}

	nAnd1.left = &n1
	nAnd1.right = &nAnd2
	n1.parent = &nAnd1
	nAnd2.parent = &nAnd1

	nAnd2.left = &nOr1
	nAnd2.right = &nOr2
	nOr1.parent = &nAnd2
	nOr2.parent = &nAnd2

	nOr1.left = &n2
	nOr1.right = &n3
	nOr2.left = &n4
	nOr2.right = &n5

	n2.parent = &nOr1
	n3.parent = &nOr1
	n4.parent = &nOr2
	n5.parent = &nOr2

	p := &predicate{
		root: &nAnd1,
		undoneAtoms: map[int]*predNode{
			fpe1ID: &n1,
			fpe2ID: &n2,
			fpe3ID: &n3,
			fpe4ID: &n4,
			fpe5ID: &n5,
		},
	}

	/* /site/people/person[p]/name */
	fpebuilder6 := fpeBuilder{}
	fpebuilder6.addAxis(child)
	fpebuilder6.addUdpeTest(newElementTest("site", nil, nil))
	fpebuilder6.addAxis(child)
	fpebuilder6.addUdpeTest(newElementTest("people", nil, nil))
	fpebuilder6.addAxis(child)
	fpebuilder6.addUdpeTest(newElementTest("person", nil, p))
	fpebuilder6.addAxis(child)
	fpebuilder6.addUdpeTest(newElementTest("name", nil, nil))

	udpeGlobalTable.addFpe(fpebuilder6.end())
}

func (executor *executor) B1() {
	executor.mainQueryType = UDPE

	rpeBuilder1 := rpeBuilder{}
	rpeBuilder1.addAxis(parent)
	rpeBuilder1.addUdpeTest(newElementTest("namerica", nil, nil))
	rpeBuilder1.addAxis(parent)
	rpeBuilder1.addUdpeTest(newElementTest("samerica", nil, nil))

	rpe1ID, _ := udpeGlobalTable.addRpe(rpeBuilder1.end())

	node := predNode{op: atom()}
	p := &predicate{
		root: &node,
		undoneAtoms: map[int]*predNode{rpe1ID: &node},
	}

	/* /site/regions/namerica/item[p]/name */
	fpebuilder2 := fpeBuilder{}
	fpebuilder2.addAxis(child)
	fpebuilder2.addUdpeTest(newElementTest("site", nil, nil))
	fpebuilder2.addAxis(child)
	fpebuilder2.addUdpeTest(newElementTest("regions", nil, nil))
	fpebuilder2.addAxis(child)
	fpebuilder2.addUdpeTest(newElementTest("*", nil, nil))
	fpebuilder2.addAxis(child)
	fpebuilder2.addUdpeTest(newElementTest("item", nil, p))
	fpebuilder2.addAxis(child)
	fpebuilder2.addUdpeTest(newElementTest("name", nil, nil))

	udpeGlobalTable.addFpe(fpebuilder2.end())
}

func (executor *executor) B2() {
	executor.mainQueryType = NUDPE
	nudpeRec := nudpeGlobalTable.addNudpeRecord(3)

	fpeBuilder1 := fpeBuilder{}
	fpeBuilder1.addAxis(descendantOrSelf)
	fpeBuilder1.addUdpeTest(newElementTest("keyword", nil, nil))

	rpeBuilder1 := rpeBuilder{}
	rpeBuilder1.addAxis(ancestorOrSelf)
	rpeBuilder1.addUdpeTest(newElementTest("listitem", nil, nil))

	fpeBuilder2 := fpeBuilder{}
	fpeBuilder2.addAxis(child)
	fpeBuilder2.addUdpeTest(newElementTest("listitem", nil, nil))
	fpeBuilder2.addAxis(child)
	fpeBuilder2.addUdpeTest(newElementTest("Text", nil, nil))
	fpeBuilder2.addAxis(child)
	fpeBuilder2.addUdpeTest(newElementTest("keyword", nil, nil))

	fpe1 := fpeBuilder1.end()
	_, rec1 := udpeGlobalTable.addFpe(fpe1)
	rec1.setNudpeRecord(nudpeRec)

	rpe1 := rpeBuilder1.end()
	_, rec2 := udpeGlobalTable.addRpe(rpe1)
	rec2.setNudpeRecord(nudpeRec)

	fpe2 := fpeBuilder2.end()
	_, rec3 := udpeGlobalTable.addFpe(fpe2)
	rec3.setNudpeRecord(nudpeRec)
}

type Executor executor
