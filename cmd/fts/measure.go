// Copyright 2024 by Chris Palmer (https://noncombatant.org)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"runtime"
	"time"
)

// MeasureTimeAndSpace attempts to measure the CPU and memory costs of invoking
// f, and returns what it finds: the time duration, the number of heap allocated
// bytes, and the number of heap allocations that invoking f incurred.
//
// If measure is false, MeasureTimeAndSpace is equivalent to just invoking f: no
// measurements are made and only zero values are returned. If gc is true,
// MeasureTimeAndSpace will invoke runtime.GC before measuring memory usage.
//
// TODO: Consider returning either before and after runtime.MemStats, or a diff
// of before and after, or exposing a diff function.
func MeasureTimeAndSpace(measure bool, gc bool, f func()) (d time.Duration, bytes uint64, allocs uint64) {
	if !measure {
		f()
		return time.Duration(0), 0, 0
	}

	if gc {
		runtime.GC()
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	f()
	d = time.Since(start)
	runtime.ReadMemStats(&after)
	return d, after.Alloc - before.Alloc, after.Mallocs - before.Mallocs
}
