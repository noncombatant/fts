// Copyright 2024 by Chris Palmer (https://noncombatant.org), adapted and
// extended from code by Artem Krylysov
// (https://github.com/akrylysov/simplefts).
// SPDX-License-Identifier: Apache-2.0

package simplefts

import (
	"bufio"
	"io"
	"log"
	"strings"
	"unicode"
	"unicode/utf8"

	// TODO: Make use of the other languages snowball supports!
	snowballen "github.com/kljensen/snowball/english"
	snowballfr "github.com/kljensen/snowball/french"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

type Stemmer func(word string, stemStopWords bool) string

var (
	StemEnglish = snowballen.Stem
	StemFrench  = snowballfr.Stem

	normalizer = transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFKC)
)

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r)
}

// scanWords is a split function for a [bufio.Scanner] that returns each word of
// text. It will never return an empty string. Refer to [isWordRune] and
// [addStream].
func scanWords(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// Skip leading non-word runes.
	start := 0
	for width := 0; start < len(data); start += width {
		var r rune
		r, width = utf8.DecodeRune(data[start:])
		if isWordRune(r) {
			break
		}
	}
	// Scan until non-word rune, marking end of word.
	for width, i := 0, start; i < len(data); i += width {
		var r rune
		r, width = utf8.DecodeRune(data[i:])
		if !isWordRune(r) {
			return i + width, data[start:i], nil
		}
	}
	// If we're at EOF, we have a final, non-empty, non-terminated word. Return it.
	if atEOF && len(data) > start {
		return len(data), data[start:], nil
	}
	// Request more data.
	return start, nil, nil
}

func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r) // Mn: nonspacing marks
}

// normalizeToken normalizes t and returns it, or returns an empty string if t
// is not a valid token.
func normalizeToken(t string) string {
	result, _, e := transform.String(normalizer, t)
	if e != nil {
		log.Print(e)
	}
	return strings.ToLower(result)
}

// scannerToSlice scans r with [scanWords] and returns a slice of the words.
//
// Using this on large inputs will be space-inefficient.
func scannerToSlice(r io.Reader) []string {
	s := make([]string, 0)
	scanner := bufio.NewScanner(r)
	scanner.Split(scanWords)
	for scanner.Scan() {
		t := normalizeToken(scanner.Text())
		if t == "" {
			continue
		}
		s = append(s, t)
	}
	return s
}
