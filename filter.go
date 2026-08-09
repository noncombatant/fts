// Copyright 2024 by Chris Palmer (https://noncombatant.org), adapted and
// extended from code by Artem Krylysov
// (https://github.com/akrylysov/simplefts).
// SPDX-License-Identifier: Apache-2.0

package simplefts

import (
	"strings"

	snowballeng "github.com/kljensen/snowball/english"
)

// MapToLower returns a slice of tokens normalized to lower case.
func MapToLower(tokens []string) []string {
	r := make([]string, len(tokens))
	for i, token := range tokens {
		r[i] = strings.ToLower(token)
	}
	return r
}

var EnglishStopwords = SetFromSlice([]string{
	"a", "and", "be", "have", "i", "in", "of", "that", "the", "to",
})

// FilterStopwords returns a slice of tokens with stopwords removed.
func FilterStopwords(tokens []string, stopwords Set[string]) []string {
	r := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if !stopwords.Contains(token) {
			r = append(r, token)
		}
	}
	return r
}

// MapTokensToStems returns a slice of stemmed tokens.
func MapTokensToStems(tokens []string) []string {
	r := make([]string, len(tokens))
	for i, token := range tokens {
		r[i] = snowballeng.Stem(token, false)
	}
	return r
}

// FilterStopwordsAndMapToStems returns a slice containing no stopwords, and the
// stemmed tokens.
func FilterStopwordsAndMapToStems(tokens []string, stopwords Set[string]) []string {
	r := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if !stopwords.Contains(token) {
			r = append(r, snowballeng.Stem(token, false))
		}
	}
	return r
}

var stemMemo = make(map[string]string)

// FilterStopwordsAndMapToStemsMemoized returns a slice containing no stopwords,
// and the stemmed tokens.
func FilterStopwordsAndMapToStemsMemoized(tokens []string, stopwords Set[string]) []string {
	r := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if !stopwords.Contains(token) {
			stem, ok := stemMemo[token]
			if !ok {
				stem = snowballeng.Stem(token, false)
				// BUG TODO: Update the memo
			}
			r = append(r, stem)
		}
	}
	return r
}
