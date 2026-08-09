// Copyright 2024 by Chris Palmer (https://noncombatant.org)
// SPDX-License-Identifier: Apache-2.0

package simplefts

import (
	"time"

	blake "golang.org/x/crypto/blake2s"
)

// Document represents a document.
type Document struct {
	Pathname  string
	ModTime   time.Time
	IndexedAt time.Time
}

// ID is the unique identifier for documents.
type ID [blake.Size]byte

// ComputeID computes the cryptographic hash of the pathname and returns it as
// an [ID].
func ComputeID(pathname string) ID {
	return blake.Sum256([]byte(pathname))
}

type Documents map[ID]Document

// TODO: Not sure why we need the `ID` concept — `Documents` could just be a
// map of pathnames to `Document`s (and then remove `Pathname` from
// `Document`).
