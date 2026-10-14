// Copyright 2024 by Chris Palmer (https://noncombatant.org)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/klauspost/compress/zstd"
	fts "github.com/noncombatant/simplefts"
	"github.com/noncombatant/zreader"
)

func createIndex(index *fts.Index, rootPathname string) {
	if e := index.Add(rootPathname); e != nil {
		slog.Error("Add", "rootPathname", rootPathname, "error", e)
	}
}

func fatal(message string, args ...any) {
	slog.Error(message, args...)
	// TODO: Use sysexits
	os.Exit(1)
}

func saveIndex(index *fts.Index, pathname string) {
	f, e := os.Create(pathname)
	if e != nil {
		fatal("Create", "pathname", pathname, "error", e)
	}
	w, e := zstd.NewWriter(f)
	if e != nil {
		fatal("zstd.NewWriter", "error", e)
	}
	if e := index.Encode(w); e != nil {
		fatal("Encode", "error", e)
	}
	if e := w.Close(); e != nil {
		fatal("Close", "error", e)
	}
	if e := f.Close(); e != nil {
		fatal("Close", "error", e)
	}
}

func doSearch(idx *fts.Index, query string) {
	start := time.Now()
	results := idx.Search(query)
	elapsed := time.Now().Sub(start)
	for pathname := range results {
		slog.Info("hit", "URL", pathname)
	}
	slog.Info("results", "count", len(results), "elapsed", elapsed)
}

func atRoot(previous, name string) bool {
	here, e := os.Stat(previous)
	if e != nil {
		fatal("Stat", "previous", previous, "error", e)
	}
	parent, e := os.Stat(name)
	if e != nil {
		fatal("Stat", "name", name, "error", e)
	}
	hereStatus := here.Sys().(*syscall.Stat_t)
	parentStatus := parent.Sys().(*syscall.Stat_t)
	return hereStatus.Dev == parentStatus.Dev && hereStatus.Ino == parentStatus.Ino
}

func findIndexPathname(pathname string) string {
	if strings.ContainsRune(pathname, os.PathSeparator) {
		return pathname
	}
	name := path.Base(pathname)
	previous := ""
	for {
		if _, e := os.Stat(name); e == nil {
			return name
		}
		previous = name
		name = path.Join("..", name)
		if atRoot(path.Dir(previous), path.Dir(name)) {
			return ""
		}
	}
}

type subcommands map[string]*flag.FlagSet

func printHelp(isError bool, subcommands subcommands) {
	status := 0
	w := os.Stdout
	if isError {
		status = 1
		w = os.Stderr
	}

	fmt.Fprint(w, `Usage:
  fts help
  fts index [options] [pathname...]
  fts query [options] [term...]
`)

	keys := make([]string, 0, len(subcommands))
	for k := range subcommands {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if k == "help" {
			continue
		}
		fs := subcommands[k]
		fs.SetOutput(w)
		fmt.Fprintf(w, "\nOptions for subcommand `%s`:\n", k)
		fs.PrintDefaults()
	}
	os.Exit(status)
}

func main() {
	commands := subcommands{
		"help":  flag.NewFlagSet("help", flag.ExitOnError),
		"index": flag.NewFlagSet("index", flag.ExitOnError),
		"query": flag.NewFlagSet("query", flag.ExitOnError),
	}

	defaultIndexPathname := ".index.gob.zstd"

	indexCommand := commands["index"]
	indexIndexPathname := indexCommand.String("i", defaultIndexPathname, "index file pathname")
	indexDebug := indexCommand.Bool("v", false, "log debug output")
	indexInputLimit := indexCommand.Uint64("l", fts.DefaultInputLimit, "input size limit (bytes of tokens)")
	indexCPUProfilePathname := indexCommand.String("cpu", "", "CPU profile pathname")
	indexMemoryProfilePathname := indexCommand.String("memory", "", "memory profile pathname")

	queryCommand := commands["query"]
	queryIndexPathname := queryCommand.String("i", defaultIndexPathname, "index file pathname")

	if len(os.Args) < 2 {
		printHelp(true, commands)
	}
	switch os.Args[1] {
	case "help":
		printHelp(false, commands)
	case "index":
		indexCommand.Parse(os.Args[2:])
		if *indexCPUProfilePathname != "" {
			f, e := os.Create(*indexCPUProfilePathname)
			if e != nil {
				fatal("CPU profile", "error", e)
			}
			defer f.Close()
			if e := pprof.StartCPUProfile(f); e != nil {
				fatal("StartCPUProfile", "error", e)
			}
			defer pprof.StopCPUProfile()
		}
		pathnames := indexCommand.Args()
		if len(pathnames) == 0 {
			pathnames, _ = filepath.Glob("[^.]*")
		}
		slog.SetLogLoggerLevel(slog.LevelInfo)
		if *indexDebug {
			slog.SetLogLoggerLevel(slog.LevelDebug)
		}
		idx := fts.NewIndex(*indexInputLimit)
		for _, pathname := range pathnames {
			createIndex(idx, pathname)
		}
		saveIndex(idx, *indexIndexPathname)
		if *indexMemoryProfilePathname != "" {
			f, e := os.Create(*indexMemoryProfilePathname)
			if e != nil {
				fatal("memory profile", "error", e)
			}
			defer f.Close()
			runtime.GC()
			if e := pprof.WriteHeapProfile(f); e != nil {
				fatal("WriteHeapProfile", "error", e)
			}
		}
	case "query":
		queryCommand.Parse(os.Args[2:])
		indexPathname := findIndexPathname(*queryIndexPathname)
		if indexPathname == "" {
			fatal("findIndexPathname", "pathname", *queryIndexPathname)
		}
		zr, e := zreader.Open(indexPathname)
		if e != nil {
			fatal("open zReader", "pathname", indexPathname, "error", e)
		}
		defer zr.Close()
		idx, e := fts.DecodeIndex(zr)
		if e != nil {
			fatal("DecodeIndex", "error", e)
		}
		query := strings.Join(queryCommand.Args(), " ")
		if query != "" {
			doSearch(idx, query)
		} else {
			slog.Info("query server ready")
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				doSearch(idx, scanner.Text())
			}
		}
	default:
		printHelp(true, commands)
	}
}
