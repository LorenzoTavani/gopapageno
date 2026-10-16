package xpath

import (
	"fmt"
)

type executionTableIterableCallback func(id int, er executionRecord) (doBreak bool)

type executionTable struct {
	records []executionRecord
}

func (et *executionTable) recordByID(id int) (execRecord *executionRecord, err error) {
	defer func() {
		if r := recover(); r != nil {
			execRecord = nil
			err = fmt.Errorf("execution tfble lookup: can NOT get execution record with id %d", id)
			return
		}
	}()
	execRecord = &et.records[id]
	return
}

func (et *executionTable) mainQueryRecord() executionRecord {
	return et.records[len(et.records)-1]
}

// merge joins the incoming execution table to the receiving execution table and returns
// the receiving execution table
func (et *executionTable) merge(incoming *executionTable) (ok bool) {
	ok = true

	for id := range len(et.records) {
		if DEBUG {
			logger.Printf("  merging record %v", et.records[id].String())
			for i := 0; i < et.records[id].threads.size; i++ {
				logger.Printf("   own thread: %v", et.records[id].threads.array[i].String())
			}
		}

		incomingRecord, err := incoming.recordByID(id)
		if err != nil {
			ok = false
			break
		}

		if incoming != nil {
			for i := 0; i < incomingRecord.threads.size; i++ {
				if DEBUG {
					logger.Printf("incoming thread: %v", incomingRecord.threads.array[i].String())
				}
				et.records[id].threads.append(incomingRecord.threads.array[i])
			}
		}
		et.records[id].ctxSols.merge(incomingRecord.ctxSols)
	}
	return
}

// evaluateID returns the boolean value of an udpe with a certain id w.r.t a specific context.
func (et *executionTable) evaluateID(udpeID int, context *NonTerminal, evaluationsCount int) customBool {
	record, err := et.recordByID(udpeID)
	if err != nil {
		panic(fmt.Sprintf(`udpe boolean evaluation error for id: %v`, err))
	}

	//Do not consider the effective boolean value of all those UDPEs which are
	//the initial UDPE of some NUDPE.
	//The boolean value of a NUDPE is computed at the end of the entire parsing of the document.
	if record.gNudpeRecord != nil {
		return Undefined
	}

	if record.ctxSols.hasSolutionsFor(context) {
		return True
	}

	//FPEs are always bounded w.r.t. a specific context and their boolean value
	//is always available as soon as the context has been synthesized.
	if record.expType == FPE {
		return False
	}

	//RPEs may be unbounded w.r.t. a certain context and their boolean values
	//may be undefined at the time when the context is synthesized.
	if evaluationsCount == 0 || record.hasExecutionThreadRunningFor(context) {
		return Undefined
	}

	return False
}

type executionRecord struct {
	threads      swapbackArray[executionThread]
	ctxSols      contextSolutionsMap
	gNudpeRecord *globalNudpeRecord
	expType      udpeType
}

func (er *executionRecord) String() string {
	return fmt.Sprintf("{ %v | %v | #%d }", er.expType, er.ctxSols, er.threads.size)
}

func (er *executionRecord) addExecutionThread(ctx, sol *NonTerminal, pp pathPattern) (et executionThread) {
	et = executionThread{
		ctx: ctx,
		sol: sol,
		pp:  pp,
		speculations: swapbackArray[speculation]{
			array: make([]speculation, 0),
		},
	}
	er.threads.append(et)
	return
}

func (er *executionRecord) removeExecutionThreadAt(i int) {
	er.threads.removeAt(i)
}

func (er *executionRecord) hasExecutionThreadRunningFor(ctx *NonTerminal) bool {
	for i := 0; i < er.threads.size; i++ {
		found := er.threads.array[i].ctx == ctx
		if found {
			return true
		}
	}
	return false
}
