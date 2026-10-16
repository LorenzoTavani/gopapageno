package xpath

import (
	"fmt"
)

// TODO(vvihorev): further simplify the call to reduction
func Reduce(reducedNT, generativeNT, wrappedNT *NonTerminal) {
	var updatingExecutionTable executionTable
	if wrappedNT != nil {
		updatingExecutionTable = wrappedNT.executionTable
	} else {
		updatingExecutionTable = udpeGlobalTable.newExecutionTable()
	}

	if DEBUG {
		logger.Printf("REDUCTION: %v <-- %v <> %v </>", reducedNT.String(), generativeNT.String(), wrappedNT.String())
	}

	// iterate over all global udpe records and execute main phases
	for id, gr := range udpeGlobalTable.list {
		if DEBUG {
			logger.Printf(" UDPE %d", id)
		}

		// Phase 1 checks if the udpe's entry point matches current
		// Reduction, without updating the path pattern, and, if it
		// maches, creates new execution threads accordingly
		{
			if DEBUG {
				logger.Printf("  PHASE 1")
			}
			udpe := gr.udpe()
			entryPoint := udpe.entryPoint()

			if _, _, ok := entryPoint.matchWithReductionOf(reducedNT.Node(), false); ok {
				switch udpeType := gr.udpeType(); udpeType {
				case FPE:
					// When a node matches an FPE entry point, the thread has found a solution
					if DEBUG {
						et := updatingExecutionTable.records[id].addExecutionThread(nil, reducedNT, entryPoint)
						logger.Printf("   adding execution thread: %v", et.String())
					} else {
						updatingExecutionTable.records[id].addExecutionThread(nil, reducedNT, entryPoint)
					}
				case RPE:
					if wrappedNT != nil {
						// When a node matches an RPE entry point, the thread has found a context
						if DEBUG {
							et := updatingExecutionTable.records[id].addExecutionThread(wrappedNT, nil, entryPoint)
							logger.Printf("   adding execution thread: %v", et.String())
						} else {
							updatingExecutionTable.records[id].addExecutionThread(wrappedNT, nil, entryPoint)
						}
						childrenOfWrappedNT := wrappedNT.Children()
						for _, child := range childrenOfWrappedNT {
							updatingExecutionTable.records[id].addExecutionThread(child, nil, udpe.entryPoint())
						}
					}
				default:
					panic(fmt.Sprintf(`adding new execution threads to execution record: unknown udpe type %q`, udpeType))
				}
			}
		}

		// Phase 2 - Update all execution threads
		{
			var predicate *predicate
			var newPathPattern pathPattern
			var ok bool

			if DEBUG {
				logger.Printf("  PHASE 2")
			}
			for i := 0; i < updatingExecutionTable.records[id].threads.size; i++ {
				etPathPattern := updatingExecutionTable.records[id].threads.array[i].pp

				//The path pattern of the execution thread may be empty if the thread is speculative
				//and it's not completed because of some unchecked speculation
				if etPathPattern.isEmpty() {
					continue
				}

				if DEBUG {
					ppBefore := etPathPattern.String()
					predicate, newPathPattern, ok = etPathPattern.matchWithReductionOf(reducedNT.Node(), true)
					logger.Printf("   updating execution thread: %v -> %v", ppBefore, etPathPattern.String())
				} else {
					predicate, newPathPattern, ok = etPathPattern.matchWithReductionOf(reducedNT.Node(), true)
				}

				if ok {
					if newPathPattern != nil {
						etReceivingSpeculation := updatingExecutionTable.records[id].addExecutionThread(updatingExecutionTable.records[id].threads.array[i].ctx, updatingExecutionTable.records[id].threads.array[i].sol, newPathPattern)
						updatingExecutionTable.records[id].threads.array[i].pp = newPathPattern
						if predicate != nil {
							etReceivingSpeculation.addSpeculation(predicate, *reducedNT)
						}
					} else {
						if predicate != nil {
							updatingExecutionTable.records[id].threads.array[i].addSpeculation(predicate, *reducedNT)
						}
					}

				} else {
					updatingExecutionTable.records[id].removeExecutionThreadAt(i)
					i--
				}
			}
		}

		// Phase 3 - Stop speculative threads, store context-solution pairs for compeleted threads
		{
			if DEBUG {
				logger.Printf("  PHASE 3")
			}
			for i := 0; i < updatingExecutionTable.records[id].threads.size; i++ {
				// If a thread speculation is unfounded, stop the speculative execution thread, and all its Children recursively
				if areSpeculationsFounded := updatingExecutionTable.records[id].threads.array[i].checkAndUpdateSpeculations(updatingExecutionTable.evaluateID); !areSpeculationsFounded {
					updatingExecutionTable.records[id].removeExecutionThreadAt(i)
					i--
				}
			}

			for i := 0; i < updatingExecutionTable.records[id].threads.size; i++ {
				// for all completed threads save the input NonTerminal as either context or solution.
				// By the time at which the execution thread is completed, it might not be able
				// to produce context-solution items because of running speculations.
				// Even if the execution thread can not produce context-solution items, it has to save the non
				// terminal whose reducetion caused the execution thread to complete
				if updatingExecutionTable.records[id].threads.array[i].pp.isEmpty() {
					if updatingExecutionTable.records[id].threads.array[i].ctx == nil {
						updatingExecutionTable.records[id].threads.array[i].ctx = reducedNT
					} else if updatingExecutionTable.records[id].threads.array[i].sol == nil {
						updatingExecutionTable.records[id].threads.array[i].sol = reducedNT
					}
				}
			}

			for i := 0; i < updatingExecutionTable.records[id].threads.size; i++ {
				// produce context solutions out of completed non speculative execution threads
				if DEBUG {
					logger.Printf("   checking context-solutions for: %v", updatingExecutionTable.records[id].threads.array[i].String())
				}

				if updatingExecutionTable.records[id].threads.array[i].pp.isEmpty() && !updatingExecutionTable.records[id].threads.array[i].isSpeculative() {
					updatingExecutionTable.records[id].ctxSols.addContextSolution(updatingExecutionTable.records[id].threads.array[i].ctx.Position().Start(), updatingExecutionTable.records[id].threads.array[i].ctx.Position().End(), updatingExecutionTable.records[id].threads.array[i].sol)
					if DEBUG {
						logger.Printf("   added a context-solution: [ %v, %v ] to record: %d %v", updatingExecutionTable.records[id].threads.array[i].ctx.String(), updatingExecutionTable.records[id].threads.array[i].sol.String(), i, updatingExecutionTable.records[id].String())
					}
					updatingExecutionTable.records[id].removeExecutionThreadAt(i)
					i--
				}
			}
		}
	}

	if DEBUG {
		logger.Printf(" MERGE")
	}

	if generativeNT != nil {
		// merge updating execution table with unchanged execution table
		unchangedExecutionTable := generativeNT.executionTable
		ok := updatingExecutionTable.merge(&unchangedExecutionTable)
		if !ok {
			panic(`Reduction Handle Node error: can NOT merge execution tables`)
		}
	}

	if DEBUG {
		for i, er := range updatingExecutionTable.records {
			if generativeNT == nil {
				logger.Printf("  skipping merge %d %v", i, er.String())
			} else {
				logger.Printf("  after merge %d %v", i, er.String())
			}

			for i := 0; i < er.threads.size; i++ {
				logger.Printf("  own thread: %v", er.threads.array[i].String())
			}
		}
	}
	// propagate updating execution table to reduced NT
	reducedNT.executionTable = updatingExecutionTable
}
