//
// hiiragi :: finder_test.go
//
//   Copyright (c) 2016-2021 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
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
