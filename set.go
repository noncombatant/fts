// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause
//
// Adapted and expanded from
// https://github.com/tailscale/tailscale/tree/main/util/set

package simplefts

import (
	"encoding/json"
	"maps"
)

// Set is a set of T.
type Set[T comparable] map[T]struct{}

// SetFromSlice returns a new set constructed from the elements in slice.
func SetFromSlice[T comparable](slice []T) Set[T] {
	s := make(Set[T])
	s.AddSlice(slice)
	return s
}

// Clone returns a new set cloned from the elements in s.
func (s Set[T]) Clone() Set[T] {
	return maps.Clone(s)
}

// Add adds e to s.
func (s Set[T]) Add(e T) { s[e] = struct{}{} }

// AddSlice adds each element of es to s.
func (s Set[T]) AddSlice(es []T) {
	for _, e := range es {
		s.Add(e)
	}
}

// Slice returns the elements of the set as a slice. The elements will not be
// in any particular order.
func (s Set[T]) Slice() []T {
	es := make([]T, 0, s.Len())
	for k := range s {
		es = append(es, k)
	}
	return es
}

// Delete removes e from the set.
func (s Set[T]) Delete(e T) { delete(s, e) }

// Contains reports whether s contains e.
func (s Set[T]) Contains(e T) bool {
	_, ok := s[e]
	return ok
}

// Len reports the number of items in s.
func (s Set[T]) Len() int { return len(s) }

// Equal reports whether s is equal to other.
func (s Set[T]) Equal(other Set[T]) bool {
	return maps.Equal(s, other)
}

// MarshalJSON returns s represented as a JSON array.
func (s Set[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Slice())
}

// UnmarshalJSON updates s to contain the given JSON array.
func (s *Set[T]) UnmarshalJSON(buf []byte) error {
	var ss []T
	if err := json.Unmarshal(buf, &ss); err != nil {
		return err
	}
	*s = SetFromSlice(ss)
	return nil
}

// Union adds the elements of other to s.
func (s Set[T]) Union(other Set[T]) {
	for e := range other {
		s.Add(e)
	}
}

// Intersection deletes from s elements that are not also in other.
func (s Set[T]) Intersection(other Set[T]) {
	for e := range s {
		if !other.Contains(e) {
			s.Delete(e)
		}
	}
}

// Difference deletes from s all elements that are in s and in other.
func (s Set[T]) Difference(other Set[T]) {
	for e := range s {
		if other.Contains(e) {
			s.Delete(e)
		}
	}
}

// Union returns a new Set containing all elements which are in either s1 or s2
// (or both).
func Union[T comparable](s1, s2 Set[T]) Set[T] {
	union := make(Set[T])
	for e := range s1 {
		union.Add(e)
	}
	for e := range s2 {
		union.Add(e)
	}
	return union
}

func intersection[T comparable](larger, smaller Set[T]) Set[T] {
	i := make(Set[T])
	for e := range smaller {
		if larger.Contains(e) {
			i.Add(e)
		}
	}
	return i
}

// Intersection returns a new Set containing all elements which are in both s1
// and s2.
func Intersection[T comparable](s1, s2 Set[T]) Set[T] {
	if s1.Len() > s2.Len() {
		return intersection(s1, s2)
	}
	return intersection(s2, s1)
}

// Difference returns a new Set containing all elements which are in s1 but not
// in s2.
func Difference[T comparable](s1, s2 Set[T]) Set[T] {
	difference := make(Set[T])
	for e := range s1 {
		if !s2.Contains(e) {
			difference.Add(e)
		}
	}
	return difference
}
