package xpath

// swapbackArray is an unordered collection with O(1) removal: the removed
// element is replaced by the last one.
type swapbackArray[V any] struct {
	array []V
	size  int
}

func (sa *swapbackArray[V]) append(v V) {
	if sa.size >= len(sa.array) {
		sa.array = append(sa.array, v)
	} else {
		sa.array[sa.size] = v
	}
	sa.size++
}

// removeAt removes the element at position i by moving the last element into
// its place. A loop going upwards must therefore look at position i again.
func (sa *swapbackArray[V]) removeAt(i int) {
	sa.size--
	sa.array[i] = sa.array[sa.size]
}
