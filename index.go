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
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"
	"unsafe"
)

// Index maps tokens (strings) to [Document] pathnames.
type Index struct {
	// Index maps tokens (strings) to 1 or more [Document] pathnames.
	Index map[string][]string

	indexMutex sync.Mutex

	// Documents contains the [Documents] this Index indexes.
	Documents

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
		Index:      make(map[string][]string),
		Documents:  make(Documents),
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
// regular files it finds to index. Along the way, it logs informative messages
// to verbose.
//
// Note: Add uses [log], not verbose, for errors.
//
// TODO: Get rid of verbose, and just use slog throughout.
func (index *Index) Add(rootPathname string, verbose *log.Logger) error {
	info, e := os.Stat(rootPathname)
	if e != nil {
		return e
	}

	var wg sync.WaitGroup
	defer wg.Wait()

	if info.Mode().IsRegular() {
		wg.Go(func() {
			if e := index.addStreamOrTimeout(rootPathname, info, verbose); e != nil {
				log.Printf("%q: %v", rootPathname, e)
			}
		})
	} else if info.Mode().IsDir() {
		filepath.WalkDir(rootPathname, func(pathname string, entry fs.DirEntry, e error) error {
			if e != nil {
				log.Printf("%q: %v", pathname, e)
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
					log.Printf("%q: %v", pathname, e)
					return nil
				}
				wg.Go(func() {
					if e := index.addStreamOrTimeout(pathname, info, verbose); e != nil {
						log.Printf("%q: %v", pathname, e)
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
	return fmt.Sprintf("Index: %d words in %d documents", len(index.Index), len(index.Documents))
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

func (index *Index) addStreamOrTimeout(pathname string, info fs.FileInfo, verbose *log.Logger) error {
	return index.addStream(pathname, info, verbose)

	//r := make(chan error, 1)
	//go func() {
	//	r <- index.addStream(pathname, info, verbose)
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
func (index *Index) addStream(pathname string, info fs.FileInfo, verbose *log.Logger) error {
	defer func() {
		if r := recover(); r != nil {
			verbose.Printf("recovered %q: %v %s", pathname, r, debug.Stack())
		}
	}()

	if shouldSkip(pathname) {
		verbose.Printf("addStream skipping %q", pathname)
		return nil
	}
	verbose.Printf("%q", pathname)

	index.documentsMutex.Lock()
	document, ok := index.Documents[pathname]
	if ok && document.IndexedAt.After(info.ModTime()) {
		index.documentsMutex.Unlock()
		return nil
	}
	index.documentsMutex.Unlock()

	document = Document{IndexedAt: time.Now()}
	count := uint64(0)

	scanner := bufio.NewScanner(bytes.NewBuffer([]byte(pathname)))
	scanner.Split(scanWords)
	index.scan(scanner, pathname, &count)

	r, e := OpenTextReader(pathname)
	if e != nil {
		return e
	}
	defer r.Close()

	scanner = bufio.NewScanner(r)
	scanner.Split(scanWords)
	const maxTokenLength = 512
	scanner.Buffer(make([]byte, 0, index.InputLimit), maxTokenLength)
	index.scan(scanner, pathname, &count)
	index.documentsMutex.Lock()
	index.Documents[pathname] = document
	index.documentsMutex.Unlock()
	return nil
}

func (index *Index) scan(scanner *bufio.Scanner, pathname string, byteCount *uint64) {
	for scanner.Scan() {
		token := normalizeToken(scanner.Text())
		if token == "" {
			continue
		}
		if index.checkLimit(token, byteCount) {
			break
		}
		index.indexMutex.Lock()
		pathnames := index.Index[token]
		// TODO: Keep these unique by using a map[string]struct{} or Set.
		index.Index[token] = append(pathnames, pathname)
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
	var r Set[string]
	for _, token := range scannerToSlice(bytes.NewBuffer([]byte(text))) {
		if pathnames, ok := index.Index[token]; ok {
			s := SetFromSlice(pathnames)
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
	return r
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
		DocumentCount: uint64(len(index.Documents)),
		TokenCount:    uint64(len(index.Index)),
	}
	for tk, hits := range index.Index {
		r.BytesUsed += uint64(len(tk))
		r.BytesUsed += uint64(len(hits)) * uint64(unsafe.Sizeof(hits[0]))
		r.TokenHits += uint64(len(hits))
	}
	return r
}
