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
	"slices"
	"sync"
	"time"
	"unsafe"
)

type stats struct {
	bytes   uint64
	elapsed time.Duration
	mutex   sync.Mutex
}

func (s *stats) add(bytes uint64, elapsed time.Duration) {
	s.mutex.Lock()
	s.bytes += bytes
	s.elapsed += elapsed
	s.mutex.Unlock()
}

// Index maps tokens (strings) to [Document] pathnames.
type Index struct {
	// TokenToIDs maps tokens to 1 or more [Document] [ID]s.
	TokenToIDs map[string][]ID

	ttdMutex sync.Mutex

	// Documents contains the [Documents] this Index indexes.
	*Documents

	// InputLimit is the maximum number of bytes’ worth of tokens to read per
	// document. 0 indicates no limit. Refer to [DefaultInputLimit].
	InputLimit uint64

	stats
}

// DefaultInputLimit is large enough to hopefully get most of the meaningful
// tokens from a [Document], while still keeping memory consumption reasonable.
// 0 indicates no limit. Refer to [Index].
const DefaultInputLimit = uint64(8 * 1024 * 1024)

// MaxTokenLength is the maximum size in bytes, not runes, of a token. (See
// [bufio.Scanner].)
const MaxTokenLength = 48

// `awk '{ print(length($0), $0) }' /usr/share/dict/words | sort -n | tail`
// yields "28 antidisestablishmentarianism".
// https://www.merriam-webster.com/wordplay/longest-words-ever has
// "Pneumonoultramicroscopicsilicovolcanoconiosis" (45 letters), and
// https://en.wikipedia.org/wiki/Longest_word_in_French has
// "hippopotomonstrosesquippedaliophobique" (38). So, 48 seems ok for now.

// NewIndex returns a new [Index] with the given inputLimit.
func NewIndex(inputLimit uint64) *Index {
	return &Index{
		TokenToIDs: make(map[string][]ID),
		Documents:  NewDocuments(),
		InputLimit: inputLimit,
	}
}

// Encode serializes index into w.
//
// It currently uses [encoding/gob], but that will surely have to change.
func (index *Index) Encode(w io.Writer) error {
	encoder := gob.NewEncoder(w)
	index.ttdMutex.Lock()
	e := encoder.Encode(index)
	index.ttdMutex.Unlock()
	return e
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
	wg.Wait()
	return nil
}

// String renders an [Index] as a string.
//
// TODO: Improve this to include more information.
func (index *Index) String() string {
	return fmt.Sprintf("Index: %d tokens, %d bytes in %d documents, indexed in %v", len(index.TokenToIDs), index.bytes, len(index.Documents.IDToDocument), index.elapsed)
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

	_, id := index.Documents.Upsert(pathname, info)

	count := uint64(0)
	scanner := bufio.NewScanner(bytes.NewBuffer([]byte(pathname)))
	scanner.Split(scanWords)
	index.scan(scanner, id, &count)

	r, e := OpenTextReader(pathname)
	if e != nil {
		return e
	}
	scanner = bufio.NewScanner(r)
	scanner.Split(scanWords)
	scanner.Buffer(make([]byte, 0, index.InputLimit), MaxTokenLength)
	index.scan(scanner, id, &count)
	r.Close()
	return nil
}

func (index *Index) scan(scanner *bufio.Scanner, id ID, byteCount *uint64) {
	for scanner.Scan() {
		start := time.Now()
		text := scanner.Text()
		token := normalizeToken(text)
		if token == "" {
			continue
		}
		if index.checkLimit(token, byteCount) {
			break
		}

		index.ttdMutex.Lock()
		ids := index.TokenToIDs[token]
		// TODO: Keep these unique by using a map or Set — don't love
		// this linear search. However, when we make this change, we
		// get a deadlock. Don't understand why yet.
		if !slices.Contains(ids, id) {
			index.TokenToIDs[token] = append(ids, id)
		}
		index.ttdMutex.Unlock()
		index.stats.add(uint64(len(text)), time.Since(start))
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
		if ids, ok := index.TokenToIDs[token]; ok {
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
	for id := range r {
		pathnames.Add(index.Documents.IDToDocument[id].Pathname)
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
		DocumentCount: uint64(len(index.Documents.IDToDocument)),
		TokenCount:    uint64(len(index.TokenToIDs)),
	}
	for tk, hits := range index.TokenToIDs {
		r.BytesUsed += uint64(len(tk))
		r.BytesUsed += uint64(len(hits)) * uint64(unsafe.Sizeof(hits[0]))
		r.TokenHits += uint64(len(hits))
	}
	return r
}
