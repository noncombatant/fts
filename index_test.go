// Copyright 2024 by Chris Palmer (https://noncombatant.org), adapted and
// extended from code by Artem Krylysov
// (https://github.com/akrylysov/simplefts).
// SPDX-License-Identifier: Apache-2.0

package simplefts

import (
	"io"
	"log"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

var discard = log.New(io.Discard, "", 0)

func TestIndex(t *testing.T) {
	idx := NewIndex(0)

	assert.Nil(t, idx.Search("foo"))
	assert.Nil(t, idx.Search("cement"))

	cpanPath := "test/data/CPAN"
	cpanID := ComputeID(cpanPath)
	idx.Add(cpanPath, discard)
	expected := SetFromSlice([]ID{cpanID})
	assert.True(t, expected.Equal(idx.Search("a")))
	assert.True(t, expected.Equal(idx.Search("perl")))
	assert.True(t, expected.Equal(idx.Search("PeRl")))
	assert.True(t, expected.Equal(idx.Search("program")))
	assert.Equal(t, 0, idx.Search("DoNuts").Len())

	codecPath := "test/data/Codec"
	codecID := ComputeID(codecPath)
	idx.Add(codecPath, discard)
	assert.Nil(t, idx.Search("donut"))
	assert.True(t, SetFromSlice([]ID{cpanID, codecID}).Equal(idx.Search("program")))
	assert.True(t, SetFromSlice([]ID{cpanID, codecID}).Equal(idx.Search("a")))
	assert.True(t, SetFromSlice([]ID{codecID}).Equal(idx.Search("encodes")))
	// TODO: When we do stemming, enable this:
	//assert.True(t, SetFromSlice([]ID{codecID}).Equal(idx.Search("encode")))
	assert.Equal(t, 0, idx.Search("DoNuts").Len())

	// Make sure decompression works:
	bz2Path := "test/data/zreader.txt.bz2"
	bz2ID := ComputeID(bz2Path)
	idx.Add(bz2Path, discard)
	gzPath := "test/data/zreader.txt.gz"
	gzID := ComputeID(gzPath)
	idx.Add(gzPath, discard)
	assert.True(t, SetFromSlice([]ID{bz2ID, gzID}).Equal(idx.Search("goat")))
	assert.True(t, SetFromSlice([]ID{bz2ID, gzID}).Equal(idx.Search("lazy")))

	// Make sure indexing pathnames works:
	cottagePath := "test/data/Craven_Cottage"
	cottageID := ComputeID(cottagePath)
	idx.Add(cottagePath, discard)
	assert.True(t, SetFromSlice([]ID{cottageID}).Equal(idx.Search("craven")))
}

func TestUnicodeNormalization(t *testing.T) {
	// TODO: Create test documents with composing and decomposing characters,
	// and query both with both. Also, create test documents with e.g.
	// superscripts, and make sure querying with normal digits works, and vice
	// versa.
	nfcPath := "test/data/nfc.txt"
	nfcID := ComputeID(nfcPath)
	nfdPath := "test/data/nfd.txt"
	nfdID := ComputeID(nfdPath)
	compatibilityKPath := "test/data/compatibility-k.txt"
	compatibilityKID := ComputeID(compatibilityKPath)
	compatibilityKelvinPath := "test/data/compatibility-kelvin.txt"
	compatibilityKelvinID := ComputeID(compatibilityKelvinPath)

	idx := NewIndex(0)
	idx.Add(nfcPath, discard)
	idx.Add(nfdPath, discard)
	idx.Add(compatibilityKPath, discard)
	idx.Add(compatibilityKelvinPath, discard)

	// Refer to the test files: these query strings look the same but are not.
	// Searching for either should yield both result files.
	assert.True(t, SetFromSlice([]ID{nfcID, nfdID}).Equal(idx.Search("é")))
	assert.True(t, SetFromSlice([]ID{nfcID, nfdID}).Equal(idx.Search("é")))

	// Same thing here:
	assert.True(t, SetFromSlice([]ID{compatibilityKID, compatibilityKelvinID}).Equal(idx.Search("K")))
	assert.True(t, SetFromSlice([]ID{compatibilityKID, compatibilityKelvinID}).Equal(idx.Search("K")))
}

var (
	// The space efficiency numbers to beat, again on my 2020 Intel MBP.
	tolerance       = float64(1.2)
	indexMaxBytes   = tolerance * float64(95716520)
	indexMaxMallocs = tolerance * float64(181959886)

	// TODO: Use the tolerance factor for time efficiency too. Run the tests
	// multiple times and average the results. Write a script to generate all
	// the numbers and draw graphs for every revision, to see trends and
	// improvements.
)

func BenchmarkIndexMemory(b *testing.B) {
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	idx := NewIndex(0)
	idx.Add("test/data", discard)

	runtime.ReadMemStats(&m2)
	loadBytes := float64(m2.Alloc - m1.Alloc)
	assert.LessOrEqual(b, loadBytes, indexMaxBytes)
	loadMallocs := float64(m2.Mallocs - m1.Mallocs)
	assert.LessOrEqual(b, loadMallocs, indexMaxMallocs)
	b.Logf("index total: %d", m2.Alloc-m1.Alloc)
	b.Logf("index mallocs: %d", m2.Mallocs-m1.Mallocs)
}
