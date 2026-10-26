package xpath

import (
	"testing"
)

// attrs builds a linked list of attributes from key, value pairs.
func attrs(kv ...string) *Attribute {
	var head *Attribute
	for i := len(kv) - 2; i >= 0; i -= 2 {
		head = &Attribute{Key: kv[i], Value: kv[i+1], Next: head}
	}
	return head
}

func TestElementTest(t *testing.T) {
	t.Run(`elementTest without Attribute`, func(t *testing.T) {
		elementTest := newElementTest("a", nil, nil)

		var tests = []struct {
			input interface{}
			want  bool
		}{
			{newText("some Text", nil), false},
			{newElement("not-a", nil, nil), false},
			{newElement("not-a", attrs("key1", "value1"), nil), false},
			{newElement("a", nil, nil), true},
			{newElement("a", attrs("key1", "value1", "key2", "value2"), nil), true},
		}

		for _, test := range tests {
			if got := elementTest.test(test.input); got != test.want {
				t.Errorf(`et.test(%v)=%v | want %v`, test.input, got, test.want)
			}
		}
	})

	t.Run(`elementTest with Attribute`, func(t *testing.T) {
		elementTest := newElementTest("a", attrs("key-want", "value-want"), nil)

		var tests = []struct {
			input interface{}
			want  bool
		}{
			{newText("some Text", nil), false},
			{newElement("not-a", nil, nil), false},
			{newElement("not-a", attrs("key1", "value1"), nil), false},
			{newElement("a", attrs("key1", "value1", "key2", "value2"), nil), false},
			{newElement("a", attrs("key1", "value1", "key-want", "value-want"), nil), true},
		}

		for _, test := range tests {
			if got := elementTest.test(test.input); got != test.want {
				t.Errorf(`et.test(%v)=%v | want %v`, test.input, got, test.want)
			}
		}
	})

	t.Run(`elementTest with predicate`, func(t *testing.T) {
		predicate := newAtom(0)
		elementTest := newElementTest("a", nil, predicate)

		etPredicate := elementTest.predicate()
		if etPredicate == nil || etPredicate == predicate {
			t.Errorf(`et.predicate() = %p | want a copy of %p`, etPredicate, predicate)
		}
	})
}

func TestTextTest(t *testing.T) {
	t.Run(`textTest with empty data`, func(t *testing.T) {
		textTest := newTextTest("")

		var tests = []struct {
			input interface{}
			want  bool
		}{
			{newElement("a", nil, nil), false},
			{newText("some Text", nil), true},
		}

		for _, test := range tests {
			if got := textTest.test(test.input); got != test.want {
				t.Errorf(`tt.test(%v)=%v | want %v`, test.input, got, test.want)
			}
		}
	})

	t.Run(`textTest with data`, func(t *testing.T) {
		textTest := newTextTest("some Text")

		var tests = []struct {
			input interface{}
			want  bool
		}{
			{newElement("a", nil, nil), false},
			{newText("not some Text", nil), false},
			{newText("some Text", nil), true},
		}
		for _, test := range tests {
			if got := textTest.test(test.input); got != test.want {
				t.Errorf(`tt.test(%v)=%v | want %v`, test.input, got, test.want)
			}
		}
	})
}
