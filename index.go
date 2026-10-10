// Copyright 2024 by Chris Palmer (https://noncombatant.org), adapted and
// extended from code by Artem Krylysov
// (https://github.com/akrylysov/simplefts).
// SPDX-License-Identifier: Apache-2.0

package simplefts

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"
	"unsafe"
)

// Index maps tokens (strings) to [Document] pathnames.
type Index struct {
	// Index maps tokens (strings) to 1 or more [Document] [ID]s.
	Index map[string][]ID

	indexMutex sync.Mutex

	// Documents contains the [Documents] this Index indexes.
	*Documents

	// TODO: Documents should hold this.
	documentsMutex sync.Mutex

	// InputLimit is the maximum number of bytes’ worth of tokens to read per
	// document. 0 indicates no limit. Refer to [DefaultInputLimit].
	InputLimit uint64
}

// DefaultInputLimit is large enough to hopefully get most of the meaningful
// tokens from a [Document], while still keeping memory consumption reasonable.
// Refer to [Index].
const DefaultInputLimit = uint64(8 * 1024 * 1024)

// NewIndex returns a new [Index] with the given inputLimit.
func NewIndex(inputLimit uint64) *Index {
	return &Index{
		Index:      make(map[string][]ID),
		Documents:  NewDocuments(),
		InputLimit: inputLimit,
	}
}

// Encode serializes index into w.
//
// It currently uses [encoding/gob], but that will surely have to change.
func (index *Index) Encode(w io.Writer) error {
	encoder := gob.NewEncoder(w)
	index.indexMutex.Lock()
	defer index.indexMutex.Unlock()
	return encoder.Encode(index)
}

// DecodeIndex deserializes an [Index] from r.
func DecodeIndex(r io.Reader) (*Index, error) {
	decoder := gob.NewDecoder(r)
	var index Index
	if e := decoder.Decode(&index); e != nil {
		return nil, e
	}
	return &index, nil
}

// Add recursively walks a directory of files and subdirectories and adds any
// regular files it finds to index.
func (index *Index) Add(rootPathname string) error {
	info, e := os.Stat(rootPathname)
	if e != nil {
		return e
	}

	var wg sync.WaitGroup
	defer wg.Wait()

	if info.Mode().IsRegular() {
		wg.Go(func() {
			if e := index.addStreamOrTimeout(rootPathname, info); e != nil {
				slog.Error("add", "pathname", rootPathname, "error", e)
			}
		})
	} else if info.Mode().IsDir() {
		filepath.WalkDir(rootPathname, func(pathname string, entry fs.DirEntry, e error) error {
			if e != nil {
				slog.Error("walk", "pathname", pathname, "error", e)
				return nil
			}
			if shouldSkip(entry.Name()) {
				if entry.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if entry.Type().IsRegular() {
				info, e := entry.Info()
				if e != nil {
					slog.Error("Info", "pathname", pathname, "error", e)
					return nil
				}
				wg.Go(func() {
					if e := index.addStreamOrTimeout(pathname, info); e != nil {
						slog.Error("add", "pathname", pathname, "error", e)
					}
				})
				return nil
			}
			return nil
		})
	}
	return nil
}

// String renders an [Index] as a string.
//
// TODO: Improve this to include more information.
func (index *Index) String() string {
	return fmt.Sprintf("Index: %d words in %d documents", len(index.Index), len(index.Documents.Documents))
}

// TODO: This is more of a concern for the crawler, which should be a separate
// component.
var ignoreBasenames = SetFromSlice([]string{
	// Pointless-to-index and often huge directories on macOS:
	"Applications",
	"CachedData",
	"Library",

	// POSIX binaries:
	"bin",
	"lib",
	"libexec",
	"sbin",
})

func shouldSkip(basename string) bool {
	if basename != "." && basename[0] == '.' {
		return true
	}
	return ignoreBasenames.Contains(basename)
}

func (index *Index) addStreamOrTimeout(pathname string, info fs.FileInfo) error {
	return index.addStream(pathname, info)

	//r := make(chan error, 1)
	//go func() {
	//	r <- index.addStream(pathname, info)
	//}()
	//const addStreamDeadline = 60 * time.Second
	//select {
	//case <-time.After(addStreamDeadline):
	//	return errors.New("timed out")
	//case e := <-r:
	//	return e
	//}
}

// addStream adds the regular file named by pathname to the index.
func (index *Index) addStream(pathname string, info fs.FileInfo) error {
	defer func() {
		if r := recover(); r != nil {
			slog.Debug("recovered", "pathname", pathname, "r", r, "stack", debug.Stack())
		}
	}()

	if shouldSkip(pathname) {
		slog.Debug("addStream skipping", "pathname", pathname)
		return nil
	}
	slog.Debug("add", "pathname", pathname)

	var document *Document
	index.documentsMutex.Lock()
	id, ok := index.Documents.pathnames[pathname]
	if ok {
		document = index.Documents.Documents[id]
		if document.IndexedAt.After(info.ModTime()) {
			// We have indexed the latest version of this document,
			// and are done.
			index.documentsMutex.Unlock()
			return nil
		}
	} else {
		// We are about to index a new document, so create it:
		id = index.Documents.g.GetNext()
		document = &Document{Pathname: pathname, IndexedAt: time.Now()}
		index.Documents.Documents[id] = document
		index.pathnames[pathname] = id
	}
	index.documentsMutex.Unlock()

	count := uint64(0)
	scanner := bufio.NewScanner(bytes.NewBuffer([]byte(pathname)))
	scanner.Split(scanWords)
	index.scan(scanner, id, &count)

	r, e := OpenTextReader(pathname)
	if e != nil {
		return e
	}
	defer r.Close()

	scanner = bufio.NewScanner(r)
	scanner.Split(scanWords)
	// TODO: Consider making this public and putting it next to DefaultInputLimit.
	const maxTokenLength = 512
	scanner.Buffer(make([]byte, 0, index.InputLimit), maxTokenLength)
	index.scan(scanner, id, &count)
	return nil
}

func (index *Index) scan(scanner *bufio.Scanner, id ID, byteCount *uint64) {
	for scanner.Scan() {
		token := normalizeToken(scanner.Text())
		if token == "" {
			continue
		}
		if index.checkLimit(token, byteCount) {
			break
		}
		index.indexMutex.Lock()
		ids := index.Index[token]
		// TODO: Keep these unique by using a map[string]struct{} or
		// Set, or at least wrap all this in a helper function.
		if len(ids) > 0 {
			last := ids[len(ids)-1]
			if last != id {
				index.Index[token] = append(ids, id)
			}
		} else {
			index.Index[token] = append(ids, id)
		}
		index.indexMutex.Unlock()
	}
}

func (index *Index) checkLimit(token string, count *uint64) (stop bool) {
	if index.InputLimit != 0 {
		*count += uint64(len(token))
		return *count > index.InputLimit
	}
	return false
}

// Search queries the index for the given text, and returns a [Set] of document
// pathnames.
func (index *Index) Search(text string) Set[string] {
	var r Set[ID]
	for _, token := range scannerToSlice(bytes.NewBuffer([]byte(text))) {
		if ids, ok := index.Index[token]; ok {
			s := SetFromSlice(ids)
			if r.Len() == 0 {
				r = s
			} else {
				r.Intersection(s)
			}
		} else {
			// Token doesn't exist.
			return nil
		}
	}

	// Now map those results into their corresponding pathnames.
	pathnames := make(Set[string])
	for id, _ := range r {
		pathnames.Add(index.Documents.Documents[id].Pathname)
	}
	return pathnames
}

// IndexStatistics estimates the size of an [Index].
type IndexStatistics struct {
	BytesUsed     uint64
	DocumentCount uint64
	TokenCount    uint64
	TokenHits     uint64
}

// Statistics returns an [IndexStatistics] estimating the size of index.
func (index *Index) Statistics() IndexStatistics {
	r := IndexStatistics{
		DocumentCount: uint64(len(index.Documents.Documents)),
		TokenCount:    uint64(len(index.Index)),
	}
	for tk, hits := range index.Index {
		r.BytesUsed += uint64(len(tk))
		r.BytesUsed += uint64(len(hits)) * uint64(unsafe.Sizeof(hits[0]))
		r.TokenHits += uint64(len(hits))
	}
	return r
}
