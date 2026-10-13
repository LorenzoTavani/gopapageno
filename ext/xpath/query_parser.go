package xpath

import (
	"fmt"
	"context"
	"github.com/LorenzoTavani/gopapageno"
)

func parse(xpathQuery string) mainQueryType {
	udpeGlobalTable = new(globalUdpeTable)
	nudpeGlobalTable = new(globalNudpeTable)
	// return parseDPE(xpathQuery, 0, len(xpathQuery))

	r := gopapageno.NewRunner(
		NewLexer(),
		NewGrammar(),
	)
	root, err := r.Run(context.Background(), []byte(xpathQuery))
	if err != nil {
		panic(fmt.Errorf("could not parse xpath query: %v", err))
	}
	// NOTE(vvihorev): last path expression remaining has not been added to global tables yet.
	root.Value.(*peSemValue).end()

	firstUdpeType := udpeGlobalTable.list[0].udpeType() 
	for i := range udpeGlobalTable.size() {
		if udpeGlobalTable.list[i].udpeType() != firstUdpeType {
			// TODO(vvihorev): this might be wrong if NUDPE size is meant as count of UDPEs in topmost NUDPE
			globalNudpe := nudpeGlobalTable.addNudpeRecord(len(udpeGlobalTable.list))
			for i := range udpeGlobalTable.size() {
				udpeGlobalTable.list[i].setNudpeRecord(globalNudpe)
			}
			return NUDPE
		}
	}
	return UDPE
}
