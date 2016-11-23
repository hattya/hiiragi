//
// hiiragi :: hiiragi_test.go
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
	"testing"
	"time"

	"github.com/hattya/go.cli"
	"github.com/hattya/hiiragi"
)

func TestDedupNoFiles(t *testing.T) {
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
	var files []string
	now := time.Now().Truncate(time.Second)
	ts := now.Add(-3 * time.Second)
	for i := 1; i < 4; i++ {
		n := filepath.Join(root, fmt.Sprintf("%v", i))
		if err := touch(n); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(n, ts, ts); err != nil {
			t.Fatal(err)
		}
		files = append(files, n)
		ts = now
	}

	ui := cli.NewCLI()
	ui.Stdout = ioutil.Discard
	ui.Stderr = ioutil.Discard

	f := hiiragi.NewFinder(ui, db)
	f.Walk(root)
	f.Close()
	if err := count(db, len(files)); err != nil {
		t.Error(err)
	}
	// update mtime
	ts = now.Add(-3 * time.Second)
	if err := os.Chtimes(files[1], ts, ts); err != nil {
		t.Fatal(err)
	}
	// change data
	ts = now
	if err := ioutil.WriteFile(files[2], []byte("data\n"), 0666); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(files[2], ts, ts); err != nil {
		t.Fatal(err)
	}

	d := hiiragi.NewDeduper(ui, db)
	if err := d.Files(); err != nil {
		t.Fatal(err)
	}
	if sameFile(files[0], files[1]) {
		t.Errorf("files should be different")
	}
	if sameFile(files[0], files[2]) {
		t.Errorf("files should be different")
	}
	if sameFile(files[1], files[2]) {
		t.Errorf("files should be different")
	}
}

func TestDedupFiles(t *testing.T) {
	dir, files, err := dedup("files", map[string]interface{}{})
	defer os.RemoveAll(dir)
	if err != nil {
		t.Fatal(err)
	}

	if sameFile(files[0], files[1]) {
		t.Errorf("files should be different")
	}
	if !sameFile(files[0], files[2]) {
		t.Errorf("files should be same")
	}
	if sameFile(files[0], files[4]) {
		t.Errorf("files should be different")
	}
	if sameFile(files[0], files[6]) {
		t.Errorf("files should be different")
	}
	if !sameFile(files[1], files[3]) {
		t.Errorf("files should be same")
	}
	if sameFile(files[1], files[5]) {
		t.Errorf("files should be different")
	}
	if sameFile(files[1], files[7]) {
		t.Errorf("files should be different")
	}
}

func TestDedupFilesPretend(t *testing.T) {
	dir, files, err := dedup("files", map[string]interface{}{
		"pretend": true,
	})
	defer os.RemoveAll(dir)
	if err != nil {
		t.Fatal(err)
	}

	if sameFile(files[0], files[1]) {
		t.Errorf("files should be different")
	}
	if sameFile(files[0], files[2]) {
		t.Errorf("files should be different")
	}
	if sameFile(files[0], files[4]) {
		t.Errorf("files should be different")
	}
	if sameFile(files[0], files[6]) {
		t.Errorf("files should be different")
	}
	if sameFile(files[1], files[3]) {
		t.Errorf("files should be different")
	}
	if sameFile(files[1], files[5]) {
		t.Errorf("files should be different")
	}
	if sameFile(files[1], files[7]) {
		t.Errorf("files should be different")
	}
}

func dedup(action string, opts map[string]interface{}) (dir string, list []string, err error) {
	dir, err = tempDir()
	if err != nil {
		return
	}
	root := filepath.Join(dir, "root")
	switch action {
	case "files":
		list, err = createFiles(root)
		if err != nil {
			return
		}
	}

	ui := cli.NewCLI()
	ui.Stdout = ioutil.Discard
	ui.Stderr = ioutil.Discard

	db, err := hiiragi.Create(filepath.Join(dir, "hiiragi.db"))
	if err != nil {
		return
	}
	defer db.Close()

	f := hiiragi.NewFinder(ui, db)
	f.Walk(root)
	f.Close()
	if err = count(db, len(list)); err != nil {
		return
	}

	d := hiiragi.NewDeduper(ui, db)
	if v, ok := opts["pretend"]; ok {
		d.Pretend = v.(bool)
	}
	switch action {
	case "files":
		err = d.Files()
	}
	return
}

func createFiles(root string) (files []string, err error) {
	now := time.Now().Truncate(time.Second)
	if err = mkdir(root); err != nil {
		return
	}

	for _, dir := range []string{
		"",
		"a", // same
		"b", // mtime is differ
		"c", // attrs are differ
	} {
		dir = filepath.Join(root, dir)
		if err = mkdir(dir); err != nil {
			return
		}
		for i := 1; i < 3; i++ {
			n := filepath.Join(dir, string('0'+i))
			if err = touch(n); err != nil {
				return
			}
			if err = os.Chtimes(n, now, now); err != nil {
				return
			}
			files = append(files, n)
		}
	}

	ts := now.Add(3 * time.Second)
	for i := 1; i < 3; i++ {
		n := filepath.Join(root, "b", string('0'+i))
		if err = os.Chtimes(n, ts, ts); err != nil {
			return
		}

		n = filepath.Join(root, "c", string('0'+i))
		if err = os.Chmod(n, 0444); err != nil {
			return
		}
	}
	return
}

func mkdir(path string) error {
	return os.MkdirAll(path, 0777)
}

func tempDir() (string, error) {
	return ioutil.TempDir("", "hiiragi.test")
}

func touch(name string) error {
	return ioutil.WriteFile(name, []byte{}, 0666)
}

func sameFile(a, b string) bool {
	fi1, err := os.Lstat(a)
	if err != nil {
		return false
	}
	fi2, err := os.Lstat(b)
	if err != nil {
		return false
	}
	return os.SameFile(fi1, fi2)
}
