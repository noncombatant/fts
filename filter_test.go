// Copyright 2024 by Chris Palmer (https://noncombatant.org), adapted and
// extended from code by Artem Krylysov
// (https://github.com/akrylysov/simplefts).
// SPDX-License-Identifier: Apache-2.0

package simplefts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapToLower(t *testing.T) {
	var (
		in  = []string{"Cat", "DOG", "fish"}
		out = []string{"cat", "dog", "fish"}
	)
	assert.Equal(t, out, MapToLower(in))
}

func TestFilterStopwords(t *testing.T) {
	var (
		in  = []string{"i", "am", "the", "cat"}
		out = []string{"am", "cat"}
	)
	assert.Equal(t, out, FilterStopwords(in, EnglishStopwords))
}

func TestMapTokensToStems(t *testing.T) {
	var (
		in  = []string{"cat", "cats", "fish", "fishing", "fished", "airline"}
		out = []string{"cat", "cat", "fish", "fish", "fish", "airlin"}
	)
	assert.Equal(t, out, MapTokensToStems(in))
}

func TestFilterStopwordsAndMapToStems(t *testing.T) {
	var (
		in  = []string{"i", "am", "the", "cat", "cats", "fish", "fishing", "fished", "airline"}
		out = []string{"am", "cat", "cat", "fish", "fish", "fish", "airlin"}
	)
	assert.Equal(t, out, FilterStopwordsAndMapToStems(in, EnglishStopwords))
	assert.Equal(t, out, FilterStopwordsAndMapToStemsMemoized(in, EnglishStopwords))
}
