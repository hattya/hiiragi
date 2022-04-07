//
// hiiragi :: db_test.go
//
//   Copyright (c) 2016-2022 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package hiiragi_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/hattya/hiiragi"
)

func TestDBCreate(t *testing.T) {
	dir := t.TempDir()

	db, err := hiiragi.Create(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if g, e := len(files), 0; g != e {
		t.Errorf("expected len(files) = %v, got %v", e, g)
	}

	db, err = hiiragi.Create(filepath.Join(dir, "hiiragi.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	files, err = os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if g, e := len(files), 1; g != e {
		t.Fatalf("expected len(files) = %v, got %v", e, g)
	}
	if g, e := files[0].Name(), "hiiragi.db"; g != e {
		t.Errorf("expected %v, got %v", e, g)
	}
}

func TestDBOpen(t *testing.T) {
	dir := t.TempDir()

	db, err := hiiragi.Create(filepath.Join(dir, "hiiragi.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if g, e := len(files), 1; g != e {
		t.Errorf("expected len(files) = %v, got %v", e, g)
	}

	db, err = hiiragi.Open(filepath.Join(dir, "hiiragi.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestDBCacheSize(t *testing.T) {
	db, err := hiiragi.Create(filepath.Join(t.TempDir(), "hiiragi.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.SetCacheSize(2000 * 1024); err != nil {
		t.Error(err)
	}

	if err := db.SetCacheSize(-2000); err != nil {
		t.Error(err)
	}
}

func TestDBFiles(t *testing.T) {
	dir := t.TempDir()
	db, err := hiiragi.Create(filepath.Join(dir, "hiiragi.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	root := filepath.Join(dir, "root")
	if err := mkdir(root); err != nil {
		t.Fatal(err)
	}

	test := func(db *hiiragi.DB, v int64) (err error) {
		_, err = db.NextFiles(context.Background(), true, hiiragi.Asc)
		if err != nil {
			return
		}
		done, n, err := db.NumFiles()
		if err != nil {
			return
		} else if g, e := n-done, v; g != e {
			err = fmt.Errorf("expected count(file) = %v, got %v", e, g)
			return
		}
		done, n, err = db.NumSymlinks()
		if err != nil {
			return
		} else if g, e := n-done, int64(0); g != e {
			err = fmt.Errorf("expected count(symlink) = %v, got %v", e, g)
		}
		return
	}

	if err := test(db, 0); err != nil {
		t.Fatal(err)
	}
	// create
	f1 := filepath.Join(root, "1")
	if err := touch(f1); err != nil {
		t.Fatal(err)
	}
	f2 := filepath.Join(root, "2")
	if err := touch(f2); err != nil {
		t.Fatal(err)
	}
	if err := update(db, f1); err != nil {
		t.Fatal(err)
	}
	if err := update(db, f2); err != nil {
		t.Fatal(err)
	}
	if err := test(db, 2); err != nil {
		t.Fatal(err)
	}
	// mark done: f1
	if err := db.Done(f1); err != nil {
		t.Fatal(err)
	}
	if err := test(db, 1); err != nil {
		t.Fatal(err)
	}
	// mark done: f1
	if err := db.Done(f2); err != nil {
		t.Fatal(err)
	}
	if err := test(db, 0); err != nil {
		t.Fatal(err)
	}
	// update
	if err := update(db, f1); err != nil {
		t.Fatal(err)
	}
	if err := update(db, f2); err != nil {
		t.Fatal(err)
	}
	if err := test(db, 2); err != nil {
		t.Fatal(err)
	}
}

func TestDBSymlinks(t *testing.T) {
	if !supportsSymlinks {
		t.Skipf("skipping on %v", runtime.GOOS)
	}

	dir := t.TempDir()
	db, err := hiiragi.Create(filepath.Join(dir, "hiiragi.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	root := filepath.Join(dir, "root")
	if err := mkdir(root); err != nil {
		t.Fatal(err)
	}

	test := func(db *hiiragi.DB, v int64) (err error) {
		_, err = db.NextSymlinks(context.Background(), true, hiiragi.Asc)
		if err != nil {
			return
		}
		done, n, err := db.NumSymlinks()
		if err != nil {
			return
		} else if g, e := n-done, v; g != e {
			err = fmt.Errorf("expected count(symlink) = %v, got %v", e, g)
			return
		}
		done, n, err = db.NumFiles()
		if err != nil {
			return
		} else if g, e := n-done, int64(0); g != e {
			err = fmt.Errorf("expected count(file) = %v, got %v", e, g)
		}
		return
	}

	if err := test(db, 0); err != nil {
		t.Fatal(err)
	}
	// create
	s1 := filepath.Join(root, "1")
	if err := touch(filepath.Join(root, "a")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("a", s1); err != nil {
		t.Fatal(err)
	}
	s2 := filepath.Join(root, "2")
	if err := touch(filepath.Join(root, "b")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("b", s2); err != nil {
		t.Fatal(err)
	}
	if err := update(db, s1); err != nil {
		t.Fatal(err)
	}
	if err := update(db, s2); err != nil {
		t.Fatal(err)
	}
	if err := test(db, 2); err != nil {
		t.Fatal(err)
	}
	// mark done: s1
	if err := db.Done(s1); err != nil {
		t.Fatal(err)
	}
	if err := test(db, 1); err != nil {
		t.Fatal(err)
	}
	// mark done: s2
	if err := db.Done(s2); err != nil {
		t.Fatal(err)
	}
	if err := test(db, 0); err != nil {
		t.Fatal(err)
	}
	// update
	if err := update(db, s1); err != nil {
		t.Fatal(err)
	}
	if err := update(db, s2); err != nil {
		t.Fatal(err)
	}
	if err := test(db, 2); err != nil {
		t.Fatal(err)
	}
}

func TestDBUpdate(t *testing.T) {
	if !supportsSymlinks {
		t.Skipf("skipping on %v", runtime.GOOS)
	}

	dir := t.TempDir()
	db, err := hiiragi.Create(filepath.Join(dir, "hiiragi.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	root := filepath.Join(dir, "root")
	if err := mkdir(root); err != nil {
		t.Fatal(err)
	}

	test := func(db *hiiragi.DB, v int64) (err error) {
		_, nf, err := db.NumFiles()
		if err != nil {
			return
		}
		_, ns, err := db.NumSymlinks()
		if err != nil {
			return
		} else if g, e := nf+ns, v; g != e {
			err = fmt.Errorf("expected count(info) = %v, got %v", e, g)
		}
		return
	}

	if err := test(db, 0); err != nil {
		t.Fatal(err)
	}
	// create files
	p1 := filepath.Join(root, "1")
	if err := touch(p1); err != nil {
		t.Fatal(err)
	}
	p2 := filepath.Join(root, "2")
	if err := touch(p2); err != nil {
		t.Fatal(err)
	}
	if err := update(db, p1); err != nil {
		t.Fatal(err)
	}
	if err := update(db, p2); err != nil {
		t.Fatal(err)
	}
	if err := test(db, 2); err != nil {
		t.Fatal(err)
	}
	// update file → symlink: p1
	if err := os.Remove(p1); err != nil {
		t.Fatal(err)
	}
	if err := touch(filepath.Join(root, "a")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("a", p1); err != nil {
		t.Fatal(err)
	}
	if err := update(db, p1); err != nil {
		t.Fatal(err)
	}
	if err := test(db, 2); err != nil {
		t.Fatal(err)
	}
	// update file → symlink: p2
	if err := os.Remove(p2); err != nil {
		t.Fatal(err)
	}
	if err := touch(filepath.Join(root, "b")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("b", p2); err != nil {
		t.Fatal(err)
	}
	if err := update(db, p2); err != nil {
		t.Fatal(err)
	}
	if err := test(db, 2); err != nil {
		t.Fatal(err)
	}
}

func count(db *hiiragi.DB, e int) (err error) {
	_, nf, err := db.NumFiles()
	if err != nil {
		return
	}
	_, ns, err := db.NumSymlinks()
	if err != nil {
		return
	}
	if g, e := nf+ns, int64(e); g != e {
		err = fmt.Errorf("expected count(info) = %v, got %v", e, g)
	}
	return
}

func update(db *hiiragi.DB, path string) error {
	fi, err := hiiragi.Lstat(path)
	if err != nil {
		return err
	}
	return db.Update(fi)
}

var home string

func init() {
	var err error
	home, err = os.UserHomeDir()
	if err != nil {
		panic(err)
	}
}

func TestSortFiles(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	files := []*hiiragi.File{
		{
			Path:  filepath.Join(home, "a", "2"),
			Nlink: 1,
			Mtime: now,
		},
		{
			Path:  filepath.Join(home, "b", "1"),
			Nlink: 1,
			Mtime: now,
		},
		{
			Path:  filepath.Join(home, "a", "1"),
			Nlink: 1,
			Mtime: now,
		},
		{
			Path:  filepath.Join(home, "4"),
			Nlink: 1,
			Mtime: now,
		},
		{
			Path:  filepath.Join(home, "2"),
			Nlink: 2,
			Mtime: now,
		},
		{
			Path:  filepath.Join(home, "3"),
			Nlink: 1,
			Mtime: now,
		},
		{
			Path:  filepath.Join(home, "1"),
			Nlink: 1,
			Mtime: now.Add(-3 * time.Second),
		},
	}

	hiiragi.Sort(files, hiiragi.Asc)
	for i, e := range []string{
		filepath.Join(home, "1"),
		filepath.Join(home, "2"),
		filepath.Join(home, "3"),
		filepath.Join(home, "4"),
		filepath.Join(home, "a", "1"),
		filepath.Join(home, "a", "2"),
		filepath.Join(home, "b", "1"),
	} {
		if g := files[i].Path; g != e {
			t.Errorf("expected files[%v] = %q, got %q", i, e, g)
		}
	}

	hiiragi.Sort(files, hiiragi.Desc)
	for i, e := range []string{
		filepath.Join(home, "2"),
		filepath.Join(home, "3"),
		filepath.Join(home, "4"),
		filepath.Join(home, "a", "1"),
		filepath.Join(home, "a", "2"),
		filepath.Join(home, "b", "1"),
		filepath.Join(home, "1"),
	} {
		if g := files[i].Path; g != e {
			t.Errorf("expected files[%v] = %q, got %q", i, e, g)
		}
	}
}

func TestSortSymlinks(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	syms := []*hiiragi.Symlink{
		{
			Path:  filepath.Join(home, "a", "2"),
			Nlink: 1,
			Mtime: now,
		},
		{
			Path:  filepath.Join(home, "b", "1"),
			Nlink: 1,
			Mtime: now,
		},
		{
			Path:  filepath.Join(home, "a", "1"),
			Nlink: 1,
			Mtime: now,
		},
		{
			Path:  filepath.Join(home, "4"),
			Nlink: 1,
			Mtime: now,
		},
		{
			Path:  filepath.Join(home, "2"),
			Nlink: 2,
			Mtime: now,
		},
		{
			Path:  filepath.Join(home, "3"),
			Nlink: 1,
			Mtime: now,
		},
		{
			Path:  filepath.Join(home, "1"),
			Nlink: 1,
			Mtime: now.Add(-3 * time.Second),
		},
	}

	hiiragi.Sort(syms, hiiragi.Asc)
	for i, e := range []string{
		filepath.Join(home, "1"),
		filepath.Join(home, "2"),
		filepath.Join(home, "3"),
		filepath.Join(home, "4"),
		filepath.Join(home, "a", "1"),
		filepath.Join(home, "a", "2"),
		filepath.Join(home, "b", "1"),
	} {
		if g := syms[i].Path; g != e {
			t.Errorf("expected symlinks[%v] = %q, got %q", i, e, g)
		}
	}

	hiiragi.Sort(syms, hiiragi.Desc)
	for i, e := range []string{
		filepath.Join(home, "2"),
		filepath.Join(home, "3"),
		filepath.Join(home, "4"),
		filepath.Join(home, "a", "1"),
		filepath.Join(home, "a", "2"),
		filepath.Join(home, "b", "1"),
		filepath.Join(home, "1"),
	} {
		if g := syms[i].Path; g != e {
			t.Errorf("expected symlinks[%v] = %q, got %q", i, e, g)
		}
	}
}
