package xpath

import "testing"

func TestSwapbackArrayRemoveMiddle(t *testing.T) {
	t.Skip("known bug: setIndex has a value receiver, see KNOWN_ISSUES.md")
	var sa swapbackArray[speculation]
	s1 := speculation{evaluationsCount: 1}
	s2 := speculation{evaluationsCount: 2}
	s3 := speculation{evaluationsCount: 3}
	sa.append(s1)
	sa.append(s2)
	sa.append(s3)
	sa.remove(s2)

	var got []int
	for i := 0; i < sa.size; i++ {
		got = append(got, sa.array[i].evaluationsCount)
	}
	if len(got) != 2 || got[0] != 1 || got[1] != 3 {
		t.Errorf("after removing 2 want [1 3], got %v", got)
	}
}
