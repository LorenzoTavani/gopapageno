package xpath

import (
	"fmt"
	"testing"
)

// parseOutcome runs the query parser and reports what happened.
func parseOutcome(query string) (outcome string) {
	defer func() {
		if r := recover(); r != nil {
			outcome = fmt.Sprintf("PANIC %v", r)
		}
	}()
	kind := "UDPE"
	if parse(query) == NUDPE {
		kind = "NUDPE"
	}
	return fmt.Sprintf("OK    %s with %d path expression(s)", kind, udpeGlobalTable.size())
}

// This is a measurement, not an assertion: it logs which queries the query
// parser accepts.
func TestQueryParserCoverage(t *testing.T) {
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
		`//div`,
		`/html//p`,
		`//div[@class]`,
		`//div[@class="col"]`,
		`//div[@class="col" or @class="row"]`,
		`//div[@class="col" or not @class="row"]`,
		`/html//p\\div`,
		`/body[//p\\div]`,
		`/a/b[/@c and /d\\e]`,
		`//div[@class="a" or @class="b" or @class="c"]`,
		`//div[@a and @b and @c]`,
	}
	for i, q := range queries {
		t.Run(fmt.Sprintf("q%02d", i), func(t *testing.T) {
			t.Logf("%s    %s", parseOutcome(q), q)
		})
	}
}
