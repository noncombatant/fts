// Copyright 2024 by Chris Palmer (https://noncombatant.org)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	fts "github.com/noncombatant/simplefts"
)

func main() {
	flag.Parse()
	for _, pathname := range flag.Args() {
		xr, e := fts.OpenTextReader(pathname)
		if e != nil {
			panic(e)
		}
		if _, e := io.Copy(os.Stdout, xr); e != nil {
			panic(e)
		}
		fmt.Println()
		if e := xr.Close(); e != nil {
			panic(e)
		}
	}
}
