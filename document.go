// Copyright 2024 by Chris Palmer (https://noncombatant.org)
// SPDX-License-Identifier: Apache-2.0

package simplefts

import (
	"time"
)

// Document contains metadata about an indexed document.
type Document struct {
	// Pathname is the pathname of this Document.
	Pathname string

	// IndexAt gives the time at which we last indexed this Document.
	IndexedAt time.Time
}

// Documents is a map of pathnames to [Document]s.
type Documents map[string]Document

// TODO: Idea: For string, token, et c. identifiers, use
// lukechampine.com/uint128. have to supply getter/setter, for locking. E.g.
// `GetNewIDForDocument(pathname string) Uint128`. Need a map of Uint128 to
// Document.
