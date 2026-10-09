// Copyright 2024 by Chris Palmer (https://noncombatant.org)
// SPDX-License-Identifier: Apache-2.0

package simplefts

import (
	"time"
)

// Document contains metadata about an indexed document.
type Document struct {
	// IndexAt gives the time at which we last indexed this Document.
	IndexedAt time.Time
}

// Documents maps pathnames to [Document]s.
type Documents struct {
	// Documents is a map of pathnames to [Document]s.
	Documents map[string]Document

	// IDGenerator generates the [ID]s for this set of documents.
	g IDGenerator
}

func NewDocuments() *Documents {
	return &Documents{Documents: make(map[string]Document)}
}
