package xpath

import "testing"

func TestGlobalUdpeTable(t *testing.T) {
	t.Run(`size() counts the registered path expressions`, func(t *testing.T) {
		table := new(globalUdpeTable)

		table.addFpe(nil)
		table.addFpe(nil)
		table.addFpe(nil)

		if got := table.size(); got != 3 {
			t.Errorf(`size() = %d | want 3`, got)
		}
	})
}
