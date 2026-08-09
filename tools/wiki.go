// Copyright 2024 by Chris Palmer (https://noncombatant.org), adapted and
// extended from code by Artem Krylysov
// (https://github.com/akrylysov/simplefts).
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"os"
	"path"
)

// Document represents a Wikipedia abstract XML dump file.
type Document struct {
	Title string `xml:"title"`
	URL   string `xml:"url"`
	Text  string `xml:"abstract"`
}

// extractDocuments loads a Wikipedia abstract dump and writes the documents to
// files. Dump example:
// https://dumps.wikimedia.org/enwiki/latest/enwiki-latest-abstract1.xml.gz
func extractDocuments(pathname string) error {
	f, e := os.Open(pathname)
	if e != nil {
		return e
	}
	defer f.Close()
	gz, e := gzip.NewReader(f)
	if e != nil {
		return e
	}
	defer gz.Close()
	dec := xml.NewDecoder(gz)
	dump := struct {
		Documents []Document `xml:"doc"`
	}{}
	if e := dec.Decode(&dump); e != nil {
		return e
	}

	for _, d := range dump.Documents[3000:3025] {
		file, e := os.OpenFile(path.Base(d.URL), os.O_CREATE|os.O_WRONLY, 0644)
		if e != nil {
			return e
		}
		if _, e := file.Write([]byte(d.Text)); e != nil {
			file.Close()
			return nil
		}
		if e := file.Close(); e != nil {
			file.Close()
			return nil
		}
	}
	return nil
}

func main() {
	if e := extractDocuments("enwiki-latest-abstract1.xml.gz"); e != nil {
		fmt.Fprintln(os.Stderr, e)
	}
}
