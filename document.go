// Copyright 2024 by Chris Palmer (https://noncombatant.org)
// SPDX-License-Identifier: Apache-2.0

package simplefts

import (
	"time"

	blake "golang.org/x/crypto/blake2s"
)

// Document contains metadata about an indexed document.
type Document struct {
	// Pathname is the pathname of this Document.
	Pathname string

	// ModTime gives the time this Document was modified.
	//
	// TODO: We don't need this field.
	ModTime time.Time

	// IndexAt gives the time at which we last indexed this Document.
	IndexedAt time.Time
}

// ID is the unique identifier for documents.
type ID [blake.Size]byte

// ComputeID computes the cryptographic hash of the pathname and returns it as
// an [ID].
func ComputeID(pathname string) ID {
	return blake.Sum256([]byte(pathname))
}

// Documents is a map of [ID]s to [Document]s.
type Documents map[ID]Document

// TODO: Not sure why we need the `ID` concept — `Documents` could just be a
// map of pathnames to `Document`s (and then remove `Pathname` from
// `Document`).
//
// TODO: Idea: For string, token, et c. identifiers, use
// lukechampine.com/uint128. have to supply getter/setter, for locking. E.g.
// `GetNewIDForDocument(pathname string) Uint128`. Need a map of Uint128 to
// Document.
