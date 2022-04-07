//
// hiiragi/cmd/hrg :: hrg.go
//
//   Copyright (c) 2016-2022 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"

	"github.com/cloudfoundry/gosigar"
	"github.com/hattya/go.cli"
	"github.com/hattya/hiiragi"
	"github.com/mattn/go-colorable"
	"golang.org/x/term"
)

var app = cli.NewCLI()

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	if err := app.Run(os.Args[1:]); err != nil {
		switch err.(type) {
		case cli.FlagError:
			os.Exit(2)
		case cli.Interrupt:
			os.Exit(128 + 2)
		}
		os.Exit(1)
	}
}

func init() {
	mem := sigar.Mem{}
	if err := mem.Get(); err != nil {
		panic(err)
	}

	app.Version = hiiragi.Version
	app.Usage = "[options] PATH"
	app.Desc = "Create hard links for duplicate files that are under the specified directory."
	app.Flags.Bool("a, attrs", false, "ignore file attributes")
	app.Flags.String("c, cache", "hiiragi.db", "cache file (default: %q)")
	app.Flags.PrefixChoice("m, mtime", hiiragi.When(0), map[string]interface{}{
		"oldest": hiiragi.Oldest,
		"latest": hiiragi.Latest,
	}, `ignore mtime. <when> is either "oldest" or "latest"`)
	app.Flags.MetaVar("mtime", " <when>")
	app.Flags.Bool("n, name", false, "ignore file name")
	app.Flags.Bool("p, pretend", false, "show what will be done")
	app.Flags.Bool("r, resume", false, "resume dedup with the specified cache file")
	app.Flags.Int64("s, size", -int64(mem.Total/2/1024), "cache size for SQLite (default: 50%% of system memory)")
	app.Action = cli.Simple(dedup)
	app.Stdout = colorable.NewColorable(os.Stdout)
	app.Stderr = colorable.NewColorable(os.Stderr)
}

func dedup(ctx *cli.Context) error {
	progress := false
	if f, ok := ctx.UI.Stdout.(*os.File); ok {
		progress = term.IsTerminal(int(f.Fd()))
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig
		ctx.Interrupt()
	}()

	c := ctx.String("cache")
	open := hiiragi.Create
	if ctx.Bool("resume") {
		open = hiiragi.Open
	} else {
		if _, err := os.Lstat(c); err == nil {
			return fmt.Errorf("'%v' already exists!", c)
		}
	}
	db, err := open(c)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.SetCacheSize(ctx.Int64("size")); err != nil {
		return err
	}

	if !ctx.Bool("resume") {
		f := hiiragi.NewFinder(ctx.UI, db)
		f.Progress = progress
		for _, p := range ctx.Args {
			p, err := filepath.Abs(p)
			if err != nil {
				return err
			}
			if err := f.Walk(ctx.Context(), p); err != nil {
				return err
			}
		}
		f.Close()
	}

	if ctx.Bool("pretend") {
		// close master
		if err := db.Close(); err != nil {
			return err
		}
		// copy master → temporary
		f, err := os.Open(ctx.String("cache"))
		if err != nil {
			return err
		}
		defer f.Close()
		t, err := os.CreateTemp("", "hiiragi")
		if err != nil {
			return err
		}
		defer os.Remove(t.Name())
		if _, err := io.Copy(t, f); err != nil {
			return err
		}
		// dedup with temprary
		db, err = hiiragi.Open(t.Name())
		if err != nil {
			return err
		}
		defer db.Close()
		if err := db.SetCacheSize(ctx.Int64("size")); err != nil {
			return err
		}
	}

	d := hiiragi.NewDeduper(ctx.UI, db)
	d.Attrs = !ctx.Bool("attrs")
	d.Mtime = ctx.Value("mtime").(hiiragi.When)
	d.Name = !ctx.Bool("name")
	d.Pretend = ctx.Bool("pretend")
	d.Progress = progress
	return d.All(ctx.Context())
}
