// Copyright 2024 by Chris Palmer (https://noncombatant.org)
// SPDX-License-Identifier: Apache-2.0

package simplefts

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/noncombatant/pdf"
	"github.com/noncombatant/zreader"
)

var ignoreExtensions = SetFromSlice([]string{
	".gif",
	".jfif",
	".jpeg",
	".jpg",
	".mkv",
	".mp4",
	".png",
	".tiff",
	".tfx", // Pro Tools plug-in
	".wav",
	".webm",
	".webp",
})

type trType int

const (
	trByte trType = iota
	trID3Reader
	trPDFTextReader
	trZReader
)

type TextReader struct {
	trType
	byteReader    *bytes.Reader
	pdfTextReader *pdf.TextReader
	zReader       *zreader.Reader
}

var nilReader = &TextReader{
	trType:     trByte,
	byteReader: bytes.NewReader(nil),
}

func OpenTextReader(pathname string) (*TextReader, error) {
	ext := strings.ToLower(filepath.Ext(pathname))
	if ignoreExtensions.Contains(ext) {
		return nilReader, nil
	}

	switch ext {
	case ".pdf":
		pdfReader, e := pdf.Open(pathname)
		if e != nil {
			return nil, e
		}
		return &TextReader{
			trType:        trPDFTextReader,
			pdfTextReader: pdf.NewTextReader(pdfReader),
		}, nil
	case ".mp3":
		file, e := os.Open(pathname)
		if e != nil {
			return nil, e
		}
		defer file.Close()
		data := make([]byte, 4096)
		_, e = file.Read(data)
		if e != nil {
			return nil, e
		}
		return &TextReader{
			trType:     trByte,
			byteReader: bytes.NewReader(data),
		}, nil
	}

	zr, e := zreader.Open(pathname)
	if e != nil {
		return nil, e
	}
	return &TextReader{
		trType:  trZReader,
		zReader: zr,
	}, nil
}

func (tr *TextReader) Close() error {
	switch tr.trType {
	case trZReader:
		return tr.zReader.Close()
	}
	return nil
}

func (tr *TextReader) Read(bytes []byte) (int, error) {
	switch tr.trType {
	case trByte:
		return tr.byteReader.Read(bytes)
	case trPDFTextReader:
		return tr.pdfTextReader.Read(bytes)
	case trZReader:
		return tr.zReader.Read(bytes)
	}
	panic("unknown trType")
}
