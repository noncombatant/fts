// Copyright 2024 by Chris Palmer (https://noncombatant.org), adapted and
// extended from code by Artem Krylysov
// (https://github.com/akrylysov/simplefts).
// SPDX-License-Identifier: Apache-2.0

package simplefts

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIndex(t *testing.T) {
	index := NewIndex(0)

	assert.Nil(t, index.Search("foo"))
	assert.Nil(t, index.Search("cement"))

	cpanPath := "test/data/CPAN"
	index.Add(cpanPath)
	expected := SetFromSlice([]string{cpanPath})
	assert.True(t, expected.Equal(index.Search("a")))
	assert.True(t, expected.Equal(index.Search("perl")))
	assert.True(t, expected.Equal(index.Search("PeRl")))
	assert.True(t, expected.Equal(index.Search("program")))
	assert.Equal(t, 0, index.Search("DoNuts").Len())

	codecPath := "test/data/Codec"
	index.Add(codecPath)
	assert.Nil(t, index.Search("donut"))
	assert.True(t, SetFromSlice([]string{cpanPath, codecPath}).Equal(index.Search("program")))
	assert.True(t, SetFromSlice([]string{cpanPath, codecPath}).Equal(index.Search("a")))
	assert.True(t, SetFromSlice([]string{codecPath}).Equal(index.Search("encodes")))
	// TODO: When we do stemming, enable this:
	//assert.True(t, SetFromSlice([]ID{codecID}).Equal(index.Search("encode")))
	assert.Equal(t, 0, index.Search("DoNuts").Len())

	// Make sure decompression works:
	bz2Path := "test/data/zreader.txt.bz2"
	index.Add(bz2Path)
	gzPath := "test/data/zreader.txt.gz"
	index.Add(gzPath)
	assert.True(t, SetFromSlice([]string{bz2Path, gzPath}).Equal(index.Search("goat")))
	assert.True(t, SetFromSlice([]string{bz2Path, gzPath}).Equal(index.Search("lazy")))

	// Make sure indexing pathnames works:
	cottagePath := "test/data/Craven_Cottage"
	index.Add(cottagePath)
	assert.True(t, SetFromSlice([]string{cottagePath}).Equal(index.Search("craven")))
}

func TestUnicodeNormalization(t *testing.T) {
	// TODO: Create test documents with composing and decomposing characters,
	// and query both with both. Also, create test documents with e.g.
	// superscripts, and make sure querying with normal digits works, and vice
	// versa.
	nfcPath := "test/data/nfc.txt"
	nfdPath := "test/data/nfd.txt"
	compatibilityKPath := "test/data/compatibility-k.txt"
	compatibilityKelvinPath := "test/data/compatibility-kelvin.txt"

	index := NewIndex(0)
	index.Add(nfcPath)
	index.Add(nfdPath)
	index.Add(compatibilityKPath)
	index.Add(compatibilityKelvinPath)

	// Refer to the test files: these query strings look the same but are not.
	// Searching for either should yield both result files.
	assert.True(t, SetFromSlice([]string{nfcPath, nfdPath}).Equal(index.Search("é")))
	assert.True(t, SetFromSlice([]string{nfcPath, nfdPath}).Equal(index.Search("é")))

	// Same thing here:
	assert.True(t, SetFromSlice([]string{compatibilityKPath, compatibilityKelvinPath}).Equal(index.Search("K")))
	assert.True(t, SetFromSlice([]string{compatibilityKPath, compatibilityKelvinPath}).Equal(index.Search("K")))
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

	index := NewIndex(0)
	index.Add("test/data")

	runtime.ReadMemStats(&m2)
	loadBytes := float64(m2.Alloc - m1.Alloc)
	assert.LessOrEqual(b, loadBytes, indexMaxBytes)
	loadMallocs := float64(m2.Mallocs - m1.Mallocs)
	assert.LessOrEqual(b, loadMallocs, indexMaxMallocs)
	b.Logf("index total: %d", m2.Alloc-m1.Alloc)
	b.Logf("index mallocs: %d", m2.Mallocs-m1.Mallocs)
}
