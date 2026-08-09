// Copyright 2024 by Chris Palmer (https://noncombatant.org), adapted and
// extended from code by Artem Krylysov
// (https://github.com/akrylysov/simplefts).
// SPDX-License-Identifier: Apache-2.0

package simplefts

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenizer(t *testing.T) {
	testCases := []struct {
		name    string
		text    string
		tokens  []string
		stemmer Stemmer
	}{
		{
			name:    "empty",
			text:    "",
			tokens:  []string{},
			stemmer: StemEnglish,
		},
		{
			name:    "a",
			text:    "a",
			tokens:  []string{"a"},
			stemmer: StemEnglish,
		},
		{
			name:    "small wild cat",
			text:    "small wild,cat!",
			tokens:  []string{"small", "wild", "cat"},
			stemmer: StemEnglish,
		},
		{
			name:   "decomposing e acute",
			text:   "sante\u0301",
			tokens: []string{"sante"},
		},
		{
			name:   "composing e acute",
			text:   "sant\u00e9", // NFC
			tokens: []string{"sante"},
		},
		{
			name:    "both composing and decomposing e acute",
			text:    "NFD: sante\u0301 NFC: santé",
			tokens:  []string{"nfd", "sante", "nfc", "sante"},
			stemmer: StemFrench,
		},
		{
			name:    "Kelvin symbol",
			text:    "K", // Kelvin
			tokens:  []string{"k"},
			stemmer: nil,
		},
		{
			name:    "capital K",
			text:    "K", // K
			tokens:  []string{"k"},
			stemmer: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(st *testing.T) {
			assert.EqualValues(st, tc.tokens, scannerToSlice(bytes.NewBuffer([]byte(tc.text))))
		})
	}
}
