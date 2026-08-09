// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause
//
// Adapted and expanded from
// https://github.com/tailscale/tailscale/tree/main/util/set

package simplefts

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestSet(t *testing.T) {
	s := Set[int]{}
	s.Add(1)
	s.Add(2)
	if !s.Contains(1) {
		t.Error("missing 1")
	}
	if !s.Contains(2) {
		t.Error("missing 2")
	}
	if s.Contains(3) {
		t.Error("shouldn't have 3")
	}
	if s.Len() != 2 {
		t.Errorf("wrong len %d; want 2", s.Len())
	}

	more := []int{3, 4}
	s.AddSlice(more)
	if !s.Contains(3) {
		t.Error("missing 3")
	}
	if !s.Contains(4) {
		t.Error("missing 4")
	}
	if s.Contains(5) {
		t.Error("shouldn't have 5")
	}
	if s.Len() != 4 {
		t.Errorf("wrong len %d; want 4", s.Len())
	}

	es := s.Slice()
	if len(es) != 4 {
		t.Errorf("slice has wrong len %d; want 4", len(es))
	}
	for _, e := range []int{1, 2, 3, 4} {
		if !slices.Contains(es, e) {
			t.Errorf("slice missing %d (%#v)", e, es)
		}
	}
}

func TestSetFromSlice(t *testing.T) {
	s := SetFromSlice[int]([]int{1, 2, 3, 4, 4, 1})
	if s.Len() != 4 {
		t.Errorf("wrong len %d; want 4", s.Len())
	}
	for _, n := range []int{1, 2, 3, 4} {
		if !s.Contains(n) {
			t.Errorf("should contain %d", n)
		}
	}
}

func TestEqual(t *testing.T) {
	type test struct {
		name   string
		a      Set[int]
		b      Set[int]
		wanted bool
	}
	tests := []test{
		{
			"equal",
			SetFromSlice([]int{1, 2, 3, 4}),
			SetFromSlice([]int{1, 2, 3, 4}),
			true,
		},
		{
			"not equal",
			SetFromSlice([]int{1, 2, 3, 4}),
			SetFromSlice([]int{1, 2, 3, 5}),
			false,
		},
		{
			"different lengths",
			SetFromSlice([]int{1, 2, 3, 4, 5}),
			SetFromSlice([]int{1, 2, 3, 5}),
			false,
		},
	}

	for _, tt := range tests {
		if tt.a.Equal(tt.b) != tt.wanted {
			t.Errorf("%s: failed", tt.name)
		}
	}
}

func TestClone(t *testing.T) {
	s := SetFromSlice[int]([]int{1, 2, 3, 4, 4, 1})
	if s.Len() != 4 {
		t.Errorf("wrong len %d; want 4", s.Len())
	}
	s2 := s.Clone()
	if !s.Equal(s2) {
		t.Error("clone not equal to original")
	}
	s.Add(100)
	if s.Equal(s2) {
		t.Error("clone is not distinct from original")
	}
}

func TestSetJSONRoundTrip(t *testing.T) {
	tests := []struct {
		desc    string
		strings Set[string]
		ints    Set[int]
	}{
		{"empty", make(Set[string]), make(Set[int])},
		{"nil", nil, nil},
		{"one-item", SetFromSlice([]string{"one"}), SetFromSlice([]int{1})},
		{"multiple-items", SetFromSlice([]string{"one", "two", "three"}), SetFromSlice([]int{1, 2, 3})},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			t.Run("strings", func(t *testing.T) {
				buf, err := json.Marshal(tt.strings)
				if err != nil {
					t.Fatalf("json.Marshal: %v", err)
				}
				t.Logf("marshaled: %s", buf)
				var s Set[string]
				if err := json.Unmarshal(buf, &s); err != nil {
					t.Fatalf("json.Unmarshal: %v", err)
				}
				if !s.Equal(tt.strings) {
					t.Errorf("set changed after JSON marshal/unmarshal, before: %v, after: %v", tt.strings, s)
				}
			})
			t.Run("ints", func(t *testing.T) {
				buf, err := json.Marshal(tt.ints)
				if err != nil {
					t.Fatalf("json.Marshal: %v", err)
				}
				t.Logf("marshaled: %s", buf)
				var s Set[int]
				if err := json.Unmarshal(buf, &s); err != nil {
					t.Fatalf("json.Unmarshal: %v", err)
				}
				if !s.Equal(tt.ints) {
					t.Errorf("set changed after JSON marshal/unmarshal, before: %v, after: %v", tt.ints, s)
				}
			})
		})
	}
}

func TestUnionMethod(t *testing.T) {
	s1 := SetFromSlice([]int{1, 2, 3})
	s2 := SetFromSlice([]int{3, 4, 5})
	union := Union(s1, s2)
	s1.Union(s2)
	if !union.Equal(s1) {
		t.Errorf("Union method produces result not equal to Union function: %v vs %v", s1, union)
	}
}

func TestUnion(t *testing.T) {
	s1 := SetFromSlice([]int{1, 2, 3})
	s2 := SetFromSlice([]int{3, 4, 5})
	union := Union(s1, s2)
	wanted := SetFromSlice([]int{1, 2, 3, 4, 5})
	if !union.Equal(wanted) {
		t.Errorf("%v != %v", union, wanted)
	}
}

func TestIntersectionMethod(t *testing.T) {
	s1 := SetFromSlice([]int{1, 2, 3})
	s2 := SetFromSlice([]int{3, 4, 5, 6, 7})
	intersection := Intersection(s1, s2)
	s1.Intersection(s2)
	if !intersection.Equal(s1) {
		t.Errorf("Intersection method produces result not equal to Intersection function: %v vs %v", s1, intersection)
	}

	// The Intersection function has an optimization: it iterates over the
	// smaller set. So, try it both ways:
	s1 = SetFromSlice([]int{1, 2, 3, 6, 7})
	s2 = SetFromSlice([]int{3, 4, 5})
	intersection = Intersection(s1, s2)
	s1.Intersection(s2)
	if !intersection.Equal(s1) {
		t.Errorf("Intersection method produces result not equal to Intersection function: %v vs %v", s1, intersection)
	}
}

func TestIntersection(t *testing.T) {
	s1 := SetFromSlice([]int{1, 2, 3})
	s2 := SetFromSlice([]int{2, 3, 4, 5, 6})
	intersection := Intersection(s1, s2)
	wanted := SetFromSlice([]int{2, 3})
	if !intersection.Equal(wanted) {
		t.Errorf("%v != %v", intersection, wanted)
	}

	// Intersection has an optimization: it iterates over the smaller set. So,
	// try it both ways:
	s1 = SetFromSlice([]int{1, 2, 3, 5, 6})
	s2 = SetFromSlice([]int{2, 3, 4})
	intersection = Intersection(s1, s2)
	wanted = SetFromSlice([]int{2, 3})
	if !intersection.Equal(wanted) {
		t.Errorf("%v != %v", intersection, wanted)
	}
}

func TestDifferenceMethod(t *testing.T) {
	s1 := SetFromSlice([]int{1, 2, 3})
	s2 := SetFromSlice([]int{3, 4, 5})
	difference := Difference(s1, s2)
	s1.Difference(s2)
	if !difference.Equal(s1) {
		t.Errorf("Difference method produces result not equal to Difference function: %v vs %v", s1, difference)
	}
}

func TestDifference(t *testing.T) {
	s1 := SetFromSlice([]int{1, 2, 3, 4, 5})
	s2 := SetFromSlice([]int{4, 5, 6})
	difference := Difference(s1, s2)
	wanted := SetFromSlice([]int{1, 2, 3})
	if !difference.Equal(wanted) {
		t.Errorf("%v != %v", difference, wanted)
	}
}
