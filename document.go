// Copyright 2024 by Chris Palmer (https://noncombatant.org)
// SPDX-License-Identifier: Apache-2.0

package simplefts

import (
	"sync"
	"time"
)

// Document contains metadata about an indexed document.
type Document struct {
	// Pathname indicates the pathname of this Document.
	Pathname string

	// IndexedAt is the time at which we last indexed this Document.
	IndexedAt time.Time
}

// Documents maps pathnames to [Document]s.
type Documents struct {
	// IDToDocument is a map of [ID]s to [Document]s.
	IDToDocument map[ID]*Document

	// mutex guards [IDToDocument].
	mutex sync.Mutex

	// pathnames maps pathnames to the [ID] of the corresponding
	// [Document]. This must be kept in sync with the Documents field.
	pathnames map[string]ID

	// g generates the [ID]s for this set of documents.
	g IDGenerator
}

func NewDocuments() *Documents {
	return &Documents{IDToDocument: make(map[ID]*Document), pathnames: make(map[string]ID)}
}
