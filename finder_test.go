//
// hiiragi :: finder_test.go
//
//   Copyright (c) 2016-2017 Akinori Hattori <hattya@gmail.com>
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

package hiiragi_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/hattya/go.cli"
	"github.com/hattya/hiiragi"
)

func TestFinder(t *testing.T) {
	dir, err := tempDir()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	db, err := hiiragi.Create(filepath.Join(dir, "hiiragi.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	root := filepath.Join(dir, "root")
	if err := mkdir(root); err != nil {
		t.Fatal(err)
	}
	for i := 1; i < 5; i++ {
		if err := touch(filepath.Join(root, fmt.Sprintf("file%v", i))); err != nil {
			t.Fatal(err)
		}
	}

	ui := cli.NewCLI()
	ui.Stdout = ioutil.Discard
	ui.Stderr = ioutil.Discard

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f := hiiragi.NewFinder(ui, db)
	if err := f.Walk(ctx, root); err != nil {
		t.Fatal(err)
	}
	if err := f.Walk(ctx, filepath.Join(root, "dir")); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if err := count(db, 4); err != nil {
		t.Error(err)
	}
}

func TestFinderInterrupt(t *testing.T) {
	dir, err := tempDir()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	db, err := hiiragi.Create(filepath.Join(dir, "hiiragi.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	root := filepath.Join(dir, "root")
	if err := mkdir(root); err != nil {
		t.Fatal(err)
	}
	for i := 1; i < 5; i++ {
		if err := touch(filepath.Join(root, fmt.Sprintf("file%v", i))); err != nil {
			t.Fatal(err)
		}
	}

	ui := cli.NewCLI()
	ui.Stdout = ioutil.Discard
	ui.Stderr = ioutil.Discard

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	f := hiiragi.NewFinder(ui, db)
	if err := f.Walk(ctx, root); err != context.Canceled {
		t.Fatal(err)
	}
	f.Close()
	if err := count(db, 0); err != nil {
		t.Error(err)
	}
}
