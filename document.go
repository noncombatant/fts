// Copyright 2024 by Chris Palmer (https://noncombatant.org)
// SPDX-License-Identifier: Apache-2.0

package simplefts

import (
	"io/fs"
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

// Upsert either retrieves an existing [Document] (as identified by its
// pathname) or adds a new Document to ds. Either way, it returns the Document
// and its [ID].
//
// Upsert is concurrency-safe.
func (ds *Documents) Upsert(pathname string, info fs.FileInfo) (*Document, ID) {
	var document *Document
	ds.mutex.Lock()
	id, ok := ds.pathnames[pathname]
	if ok {
		document = ds.IDToDocument[id]
		// TODO: Uncomment this when we're ready to do something in the
		// body.
		//
		//if document.IndexedAt.Before(info.ModTime()) {
		//	// TODO: At this point, we have a Document that has
		//	// been indexed before, and which is about to be
		//	// indexed again. Since its contents may have changed
		//	// substantially, we need to remove the Document's ID
		//	// everywhere it appears in Index.TokenToIDs (!); or,
		//	// alternatively, to give the Document a new ID and
		//	// mark the old ID as stale (and then vacuum the data
		//	// structures asynchronously).
		//}
	} else {
		// We are about to index a new document, so create it:
		id = ds.g.Generate()
		document = &Document{Pathname: pathname, IndexedAt: time.Now()}
		ds.IDToDocument[id] = document
		ds.pathnames[pathname] = id
	}
	ds.mutex.Unlock()
	return document, id
}
