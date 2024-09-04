package data

type Set[T comparable] struct {
	data map[T]struct{}
}

func (s *Set[T]) Has(v T) bool {
	_, exists := s.data[v]
	return exists
}

func (s *Set[T]) Add(v T) {
	s.data[v] = struct{}{}
}

func (s *Set[T]) All() []T {
	values := make([]T, len(s.data))

	i := 0
	for k := range s.data {
		values[i] = k
		i++
	}

	return values
}

func (s *Set[T]) Len() int {
	return len(s.data)
}

func (s *Set[T]) Remove(v T) {
	delete(s.data, v)
}

func NewSet[T comparable]() *Set[T] {
	return &Set[T]{
		data: make(map[T]struct{}),
	}
}
