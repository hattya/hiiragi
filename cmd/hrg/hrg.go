//
// hrg :: hrg.go
//
//   Copyright (c) 2016 Akinori Hattori <hattya@gmail.com>
//
//   Permission is hereby granted, free of charge, to any person
//   obtaining a copy of this software and associated documentation files
//   (the "Software"), to deal in the Software without restriction,
//   including without limitation the rights to use, copy, modify, merge,
//   publish, distribute, sublicense, and/or sell copies of the Software,
//   and to permit persons to whom the Software is furnished to do so,
//   subject to the following conditions:
//
//   The above copyright notice and this permission notice shall be
//   included in all copies or substantial portions of the Software.
//
//   THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
//   EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
//   MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
//   NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS
//   BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN
//   ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
//   CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
//   SOFTWARE.
//

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"

	"github.com/hattya/go.cli"
	"github.com/hattya/hiiragi"
	"github.com/mattn/go-colorable"
)

var app = cli.NewCLI()

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	if err := app.Run(os.Args[1:]); err != nil {
		switch err.(type) {
		case cli.FlagError:
			os.Exit(2)
		}
		os.Exit(1)
	}
}

func init() {
	app.Version = hiiragi.Version
	app.Usage = "[options] PATH"
	app.Desc = "Create hard links for duplicate files that are under the specified directory."
	app.Flags.String("c, cache", "hiiragi.db", "cache file (default: %q)")
	app.Flags.Bool("p, pretend", false, "show what will be done")
	app.Action = cli.Simple(dedup)
	app.Stdout = colorable.NewColorable(os.Stdout)
	app.Stderr = colorable.NewColorable(os.Stderr)
}

func dedup(ctx *cli.Context) error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig
		ctx.UI.Printf("\x1b[?25h")
		os.Exit(128 + 2)
	}()

	c := ctx.String("cache")
	if _, err := os.Lstat(c); err == nil {
		return fmt.Errorf("'%v' already exists!", c)
	}
	db, err := hiiragi.Create(c)
	if err != nil {
		return err
	}
	defer db.Close()

	f := hiiragi.NewFinder(ctx.UI, db)
	for _, p := range ctx.Args {
		p, err := filepath.Abs(p)
		if err != nil {
			return err
		}
		f.Walk(p)
	}
	f.Close()

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
		t, err := ioutil.TempFile("", "hiiragi")
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
	}

	d := hiiragi.NewDeduper(ctx.UI, db)
	d.Pretend = ctx.Bool("pretend")
	return d.Files()
}
