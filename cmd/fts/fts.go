// Copyright 2024 by Chris Palmer (https://noncombatant.org)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sort"
	"strings"
	"syscall"

	"github.com/klauspost/compress/zstd"
	fts "github.com/noncombatant/simplefts"
	"github.com/noncombatant/zreader"
)

func createIndex(index *fts.Index, rootPathname string, verbose *log.Logger) {
	if e := index.Add(rootPathname, verbose); e != nil {
		log.Print(rootPathname, e)
	}
}

func saveIndex(index *fts.Index, pathname string) {
	f, e := os.Create(pathname)
	if e != nil {
		log.Fatal(e)
	}
	w, e := zstd.NewWriter(f)
	if e != nil {
		log.Fatal(e)
	}
	if e := index.Encode(w); e != nil {
		log.Fatal(e)
	}
	if e := w.Close(); e != nil {
		log.Fatal(e)
	}
	if e := f.Close(); e != nil {
		log.Fatal(e)
	}
}

func doSearch(idx *fts.Index, query string) {
	matchedIDs := idx.Search(query)
	for id := range matchedIDs {
		doc, ok := idx.Documents[id]
		if ok {  // BUG TODO: Figure out why this is necessary.
			fmt.Println(doc.Pathname)
		}
	}
}

func atRoot(previous, name string) bool {
	here, e := os.Stat(previous)
	if e != nil {
		log.Fatal(e)
	}
	parent, e := os.Stat(name)
	if e != nil {
		log.Fatal(e)
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
	log.SetFlags(0)
	commands := subcommands{
		"help":  flag.NewFlagSet("help", flag.ExitOnError),
		"index": flag.NewFlagSet("index", flag.ExitOnError),
		"query": flag.NewFlagSet("query", flag.ExitOnError),
	}

	defaultIndexPathname := ".index.gob.zstd"

	indexCommand := commands["index"]
	indexIndexPathname := indexCommand.String("i", defaultIndexPathname, "index file pathname")
	indexVerbose := indexCommand.Bool("v", false, "print verbose output")
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
				log.Fatal(e)
			}
			defer f.Close()
			if e := pprof.StartCPUProfile(f); e != nil {
				log.Fatal(e)
			}
			defer pprof.StopCPUProfile()
		}
		pathnames := indexCommand.Args()
		if len(pathnames) == 0 {
			pathnames, _ = filepath.Glob("[^.]*")
		}
		verbose := log.New(io.Discard, "", 0)
		if *indexVerbose {
			verbose = log.New(os.Stdout, "", 0)
		}
		idx := fts.NewIndex(*indexInputLimit)
		for _, pathname := range pathnames {
			createIndex(idx, pathname, verbose)
		}
		saveIndex(idx, *indexIndexPathname)
		if *indexMemoryProfilePathname != "" {
			f, e := os.Create(*indexMemoryProfilePathname)
			if e != nil {
				log.Fatal(e)
			}
			defer f.Close()
			runtime.GC()
			if e := pprof.WriteHeapProfile(f); e != nil {
				log.Fatal(e)
			}
		}
	case "query":
		queryCommand.Parse(os.Args[2:])
		indexPathname := findIndexPathname(*queryIndexPathname)
		if indexPathname == "" {
			log.Fatalf("Could not find index file %q", *queryIndexPathname)
		}
		zr, e := zreader.Open(indexPathname)
		if e != nil {
			log.Fatal(e)
		}
		defer zr.Close()
		idx, e := fts.DecodeIndex(zr)
		if e != nil {
			log.Fatal(e)
		}
		query := strings.Join(queryCommand.Args(), " ")
		if query != "" {
			doSearch(idx, query)
		} else {
			fmt.Println("Enter your queries, 1 per line:")
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				doSearch(idx, scanner.Text())
			}
		}
	default:
		printHelp(true, commands)
	}
}
