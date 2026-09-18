package algorithm

import "iter"

type Set[E comparable] struct {
	m map[E]struct{}
}

func NewSet[E comparable]() Set[E] {
	return Set[E]{m: make(map[E]struct{})}
}

func (s Set[E]) Add(e E) {
	s.m[e] = struct{}{}
}

func (s Set[E]) Remove(e E) {
	delete(s.m, e)
}

func (s Set[E]) Contains(e E) bool {
	_, ok := s.m[e]
	return ok
}

func (s Set[E]) All() iter.Seq[E] {
	return func(yield func(E) bool) {
		for v := range s.m {
			if !yield(v) {
				return
			}
		}
	}
}
