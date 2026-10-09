// Copyright 2026 by Chris Palmer (https://noncombatant.org)
// SPDX-License-Identifier: Apache-2.0

package simplefts

import (
	"testing"
)

func TestIDGenerator(t *testing.T) {
	var g IDGenerator
	if g.current != 0 {
		t.Errorf("new current is not zero; got %v", g.current)
	}
	for i := uint64(1); i <= 10; i++ {
		n := g.GetNext()
		expect := i
		if n != ID(i) {
			t.Errorf("got %v, expected %v", n, expect)
		}
	}
}
