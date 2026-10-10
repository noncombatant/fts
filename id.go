// Copyright 2026 by Chris Palmer (https://noncombatant.org)
// SPDX-License-Identifier: Apache-2.0

package simplefts

import (
	"log/slog"
	"math"
	"os"
	"sync"
)

// ID is a large, opaque numerical value.
type ID uint64

// IDGenerator generates new IDs, unique to the generator. (I.e. you can have
// multiple IDGenerators, each with its own internal state, and the guarantee
// of uniqueness is particular to that generator.)
type IDGenerator struct {
	current uint64
	mutex   sync.Mutex
}

// Generate returns a new, unique [ID]. It will panic if it is unable to
// generate a unique ID.
func (g *IDGenerator) Generate() ID {
	g.mutex.Lock()
	if g.current == math.MaxUint64 {
		slog.Error("IDGenerator.Generate unable to create a unique ID")
		os.Exit(1)
	}
	g.current += 1
	g.mutex.Unlock()
	return ID(g.current)
}
