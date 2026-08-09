// Copyright 2024 by Chris Palmer (https://noncombatant.org)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/gob"
	"testing"

	fts "github.com/noncombatant/simplefts"
	"github.com/noncombatant/zreader"
	"github.com/stretchr/testify/assert"
)

func TestZstdGob(t *testing.T) {
	r, e := zreader.Open("testdata/foo.gob.zstd")
	assert.Nil(t, e)
	decoder := gob.NewDecoder(r)
	var index fts.Index
	e = decoder.Decode(&index)
	assert.Nil(t, e)
	e = r.Close()
	assert.Nil(t, e)
}
