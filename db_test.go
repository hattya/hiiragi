//
// hiiragi :: db_test.go
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

package hiiragi_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/hattya/hiiragi"
)

func TestDBCreate(t *testing.T) {
	dir, err := tempDir()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	db, err := hiiragi.Create(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	files, err := ioutil.ReadDir(dir)
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
	files, err = ioutil.ReadDir(dir)
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

func TestDBFiles(t *testing.T) {
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

	test := func(db *hiiragi.DB, v int64) (err error) {
		_, err = db.NextFiles()
		if err != nil {
			return
		}
		done, n, err := db.NFile()
		if err != nil {
			return
		}
		if g, e := n-done, v; g != e {
			err = fmt.Errorf("expected count(file) = %v, got %v", e, g)
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
	// set hash: f2
	h, err := hiiragi.Sum(f2)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SetHash(f2, h); err != nil {
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

func count(db *hiiragi.DB, e int) error {
	_, nf, err := db.NFile()
	if err != nil {
		return err
	}
	if g, e := nf, int64(e); g != e {
		return fmt.Errorf("expected count(info) = %v, got %v", e, g)
	}
	return nil
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
	switch runtime.GOOS {
	case "windows":
		home = `C:\Users\hiiragi`
	default:
		home = "/home/hiiragi"
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

	hiiragi.Sort(files)
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
}
