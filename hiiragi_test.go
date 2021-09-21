//
// hiiragi :: hiiragi_test.go
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
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/hattya/go.cli"
	"github.com/hattya/hiiragi"
)

var supportsSymlinks = true

func TestDedupAll(t *testing.T) {
	dir, list, err := dedup("all", map[string]interface{}{})
	defer os.RemoveAll(dir)
	if err != nil {
		t.Fatal(err)
	}

	files := list[:8]
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
	syms := list[8:]
	if 0 < len(syms) {
		if sameFile(syms[0], syms[1]) {
			t.Errorf("symlinks should be different")
		}
		if !sameFile(syms[0], syms[2]) {
			t.Errorf("symlinks should be same")
		}
		if sameFile(syms[0], syms[4]) {
			t.Errorf("symlinks should be different")
		}
		if !sameFile(syms[1], syms[3]) {
			t.Errorf("symlinks should be same")
		}
		if sameFile(syms[1], syms[5]) {
			t.Errorf("symlinks should be different")
		}
	}
}

func TestDedupAllPretend(t *testing.T) {
	dir, list, err := dedup("all", map[string]interface{}{
		"pretend": true,
	})
	defer os.RemoveAll(dir)
	if err != nil {
		t.Fatal(err)
	}

	files := list[:8]
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
	syms := list[8:]
	if 0 < len(syms) {
		if sameFile(syms[0], syms[1]) {
			t.Errorf("symlinks should be different")
		}
		if sameFile(syms[0], syms[2]) {
			t.Errorf("symlinks should be different")
		}
		if sameFile(syms[0], syms[4]) {
			t.Errorf("symlinks should be different")
		}
		if sameFile(syms[1], syms[3]) {
			t.Errorf("symlinks should be different")
		}
		if sameFile(syms[1], syms[5]) {
			t.Errorf("symlinks should be different")
		}
	}
}

func TestDedupAllIgnoreAll(t *testing.T) {
	dir, list, err := dedup("all", map[string]interface{}{
		"attrs": false,
		"mtime": hiiragi.Oldest,
		"name":  false,
	})
	defer os.RemoveAll(dir)
	if err != nil {
		t.Fatal(err)
	}

	files := list[:8]
	if !sameFile(files[0], files[1]) {
		t.Errorf("files should be same")
	}
	if !sameFile(files[0], files[2]) {
		t.Errorf("files should be same")
	}
	if !sameFile(files[0], files[4]) {
		t.Errorf("files should be same")
	}
	if !sameFile(files[0], files[6]) {
		t.Errorf("files should be same")
	}
	if !sameFile(files[1], files[3]) {
		t.Errorf("files should be same")
	}
	if !sameFile(files[1], files[5]) {
		t.Errorf("files should be same")
	}
	if !sameFile(files[1], files[7]) {
		t.Errorf("files should be same")
	}
	syms := list[8:]
	if 0 < len(syms) {
		if !sameFile(syms[0], syms[1]) {
			t.Errorf("symlinks should be same")
		}
		if !sameFile(syms[0], syms[2]) {
			t.Errorf("symlinks should be same")
		}
		if !sameFile(syms[0], syms[4]) {
			t.Errorf("symlinks should be same")
		}
		if !sameFile(syms[1], syms[3]) {
			t.Errorf("symlinks should be same")
		}
		if !sameFile(syms[1], syms[5]) {
			t.Errorf("symlinks should be same")
		}
	}
}

func TestDedupAllInterrupt(t *testing.T) {
	dir, _, err := dedup("all", map[string]interface{}{
		"interrupt": true,
	})
	if err != context.Canceled {
		t.Error(err)
	}
	os.RemoveAll(dir)
}

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f := hiiragi.NewFinder(ui, db)
	if err := f.Walk(ctx, root); err != nil {
		t.Fatal(err)
	}
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
	if err := d.Files(ctx); err != nil {
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

func TestDedupFilesIgnoreAll(t *testing.T) {
	dir, files, err := dedup("files", map[string]interface{}{
		"attrs": false,
		"mtime": hiiragi.Oldest,
		"name":  false,
	})
	defer os.RemoveAll(dir)
	if err != nil {
		t.Fatal(err)
	}

	if !sameFile(files[0], files[1]) {
		t.Errorf("files should be same")
	}
	if !sameFile(files[0], files[2]) {
		t.Errorf("files should be same")
	}
	if !sameFile(files[0], files[4]) {
		t.Errorf("files should be same")
	}
	if !sameFile(files[0], files[6]) {
		t.Errorf("files should be same")
	}
	if !sameFile(files[1], files[3]) {
		t.Errorf("files should be same")
	}
	if !sameFile(files[1], files[5]) {
		t.Errorf("files should be same")
	}
	if !sameFile(files[1], files[7]) {
		t.Errorf("files should be same")
	}
}

func TestDedupFilesIgnoreAttrs(t *testing.T) {
	dir, files, err := dedup("files", map[string]interface{}{
		"attrs": false,
	})
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
	if !sameFile(files[0], files[6]) {
		t.Errorf("files should be same")
	}
	if !sameFile(files[1], files[3]) {
		t.Errorf("files should be same")
	}
	if sameFile(files[1], files[5]) {
		t.Errorf("files should be different")
	}
	if !sameFile(files[1], files[7]) {
		t.Errorf("files should be same")
	}
}

func TestDedupFilesIgnoreMtime(t *testing.T) {
	dir, files, err := dedup("files", map[string]interface{}{
		"mtime": hiiragi.Oldest,
	})
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
	if !sameFile(files[0], files[4]) {
		t.Errorf("files should be same")
	}
	if sameFile(files[0], files[6]) {
		t.Errorf("files should be different")
	}
	if !sameFile(files[1], files[3]) {
		t.Errorf("files should be same")
	}
	if !sameFile(files[1], files[5]) {
		t.Errorf("files should be same")
	}
	if sameFile(files[1], files[7]) {
		t.Errorf("files should be different")
	}
}

func TestDedupFilesIgnoreName(t *testing.T) {
	dir, files, err := dedup("files", map[string]interface{}{
		"name": false,
	})
	defer os.RemoveAll(dir)
	if err != nil {
		t.Fatal(err)
	}

	if !sameFile(files[0], files[1]) {
		t.Errorf("files should be same")
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

func TestDedupFilesInterrupt(t *testing.T) {
	dir, _, err := dedup("files", map[string]interface{}{
		"interrupt": true,
	})
	if err != context.Canceled {
		t.Error(err)
	}
	os.RemoveAll(dir)
}

func TestDedupNoSymlinks(t *testing.T) {
	if !supportsSymlinks {
		t.Skipf("skipping on %v", runtime.GOOS)
	}

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
	tgt := filepath.Join(dir, "target")
	if err := touch(tgt); err != nil {
		t.Fatal(err)
	}
	var syms []string
	now := time.Now().Truncate(time.Second)
	ts := now.Add(-3 * time.Second)
	for i := 1; i < 4; i++ {
		n := filepath.Join(root, fmt.Sprintf("%v", i))
		if err := os.Symlink(tgt, n); err != nil {
			t.Fatal(err)
		}
		if err := lchtimes(n, ts, ts); err != nil {
			t.Fatal(err)
		}
		syms = append(syms, n)
		ts = now
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
	f.Close()
	if err := count(db, len(syms)); err != nil {
		t.Error(err)
	}
	// update mtime
	ts = now.Add(-3 * time.Second)
	if err := lchtimes(syms[1], ts, ts); err != nil {
		t.Fatal(err)
	}
	// change link target
	ts = now
	tgt = filepath.Join(dir, "another")
	if err := touch(tgt); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(syms[2]); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(tgt, syms[2]); err != nil {
		t.Fatal(err)
	}
	if err := lchtimes(syms[2], ts, ts); err != nil {
		t.Fatal(err)
	}

	d := hiiragi.NewDeduper(ui, db)
	if err := d.Symlinks(ctx); err != nil {
		t.Fatal(err)
	}
	if sameFile(syms[0], syms[1]) {
		t.Errorf("symlinks should be different")
	}
	if sameFile(syms[0], syms[2]) {
		t.Errorf("symlinks should be different")
	}
	if sameFile(syms[1], syms[2]) {
		t.Errorf("symlinks should be different")
	}
}

func TestDedupSymlinks(t *testing.T) {
	if !supportsSymlinks {
		t.Skipf("skipping on %v", runtime.GOOS)
	}

	dir, syms, err := dedup("symlinks", map[string]interface{}{})
	defer os.RemoveAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	if sameFile(syms[0], syms[1]) {
		t.Errorf("symlinks should be different")
	}
	if !sameFile(syms[0], syms[2]) {
		t.Errorf("symlinks should be same")
	}
	if sameFile(syms[0], syms[4]) {
		t.Errorf("symlinks should be different")
	}
	if !sameFile(syms[1], syms[3]) {
		t.Errorf("symlinks should be same")
	}
	if sameFile(syms[1], syms[5]) {
		t.Errorf("symlinks should be different")
	}
}

func TestDedupSymlinksIgnoreAll(t *testing.T) {
	if !supportsSymlinks {
		t.Skipf("skipping on %v", runtime.GOOS)
	}

	dir, syms, err := dedup("symlinks", map[string]interface{}{
		"attrs": false,
		"mtime": hiiragi.Oldest,
		"name":  false,
	})
	defer os.RemoveAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !sameFile(syms[0], syms[1]) {
		t.Errorf("symlinks should be same")
	}
	if !sameFile(syms[0], syms[2]) {
		t.Errorf("symlinks should be same")
	}
	if !sameFile(syms[0], syms[4]) {
		t.Errorf("symlinks should be same")
	}
	if !sameFile(syms[1], syms[3]) {
		t.Errorf("symlinks should be same")
	}
	if !sameFile(syms[1], syms[5]) {
		t.Errorf("symlinks should be same")
	}
}

func TestDedupSymlinksIgnoreName(t *testing.T) {
	if !supportsSymlinks {
		t.Skipf("skipping on %v", runtime.GOOS)
	}

	dir, syms, err := dedup("symlinks", map[string]interface{}{
		"name": false,
	})
	defer os.RemoveAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !sameFile(syms[0], syms[1]) {
		t.Errorf("symlinks should be same")
	}
	if !sameFile(syms[0], syms[2]) {
		t.Errorf("symlinks should be same")
	}
	if sameFile(syms[0], syms[4]) {
		t.Errorf("symlinks should be different")
	}
	if !sameFile(syms[1], syms[3]) {
		t.Errorf("symlinks should be same")
	}
	if sameFile(syms[1], syms[5]) {
		t.Errorf("symlinks should be different")
	}
}

func TestDedupSymlinksPretend(t *testing.T) {
	if !supportsSymlinks {
		t.Skipf("skipping on %v", runtime.GOOS)
	}

	dir, syms, err := dedup("symlinks", map[string]interface{}{
		"pretend": true,
	})
	defer os.RemoveAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	if sameFile(syms[0], syms[1]) {
		t.Errorf("symlinks should be different")
	}
	if sameFile(syms[0], syms[2]) {
		t.Errorf("symlinks should be different")
	}
	if sameFile(syms[0], syms[4]) {
		t.Errorf("symlinks should be different")
	}
	if sameFile(syms[1], syms[3]) {
		t.Errorf("symlinks should be different")
	}
	if sameFile(syms[1], syms[5]) {
		t.Errorf("symlinks should be different")
	}
}

func TestDedupSymlinksInterrupt(t *testing.T) {
	if !supportsSymlinks {
		t.Skipf("skipping on %v", runtime.GOOS)
	}

	dir, _, err := dedup("symlinks", map[string]interface{}{
		"interrupt": true,
	})
	if err != context.Canceled {
		t.Error(err)
	}
	os.RemoveAll(dir)
}

func dedup(action string, opts map[string]interface{}) (dir string, list []string, err error) {
	dir, err = tempDir()
	if err != nil {
		return
	}
	root := filepath.Join(dir, "root")
	switch action {
	case "all":
		var files, syms []string
		files, err = createFiles(filepath.Join(root, "file"))
		if err != nil {
			return
		}
		if supportsSymlinks {
			syms, err = createSymlinks(dir, filepath.Join(root, "sym"))
			if err != nil {
				return
			}
		}
		list = make([]string, len(files)+len(syms))
		copy(list, files)
		copy(list[len(files):], syms)
	case "files":
		list, err = createFiles(root)
		if err != nil {
			return
		}
	case "symlinks":
		list, err = createSymlinks(dir, root)
		if err != nil {
			return
		}
	}

	ui := cli.NewCLI()
	ui.Stdout = ioutil.Discard
	ui.Stderr = ioutil.Discard

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := hiiragi.Create(filepath.Join(dir, "hiiragi.db"))
	if err != nil {
		return
	}
	defer db.Close()

	f := hiiragi.NewFinder(ui, db)
	if err = f.Walk(ctx, root); err != nil {
		return
	}
	f.Close()
	if err = count(db, len(list)); err != nil {
		return
	}

	d := hiiragi.NewDeduper(ui, db)
	if v, ok := opts["attrs"]; ok {
		d.Attrs = v.(bool)
	}
	if v, ok := opts["mtime"]; ok {
		d.Mtime = v.(hiiragi.When)
	}
	if v, ok := opts["name"]; ok {
		d.Name = v.(bool)
	}
	if v, ok := opts["pretend"]; ok {
		d.Pretend = v.(bool)
	}
	if v, ok := opts["interrupt"]; ok && v.(bool) {
		cancel()
	}
	switch action {
	case "all":
		err = d.All(ctx)
	case "files":
		err = d.Files(ctx)
	case "symlinks":
		err = d.Symlinks(ctx)
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

func createSymlinks(path, root string) (syms []string, err error) {
	now := time.Now().Truncate(time.Second)
	tgt := filepath.Join(path, "target")
	if err = mkdir(path); err != nil {
		return
	}
	if err = touch(tgt); err != nil {
		return
	}
	if err = mkdir(root); err != nil {
		return
	}

	for _, dir := range []string{
		"",
		"a", // same
		"b", // mtime is differ
	} {
		dir = filepath.Join(root, dir)
		if err = mkdir(dir); err != nil {
			return
		}
		for i := 1; i < 3; i++ {
			n := filepath.Join(dir, string('0'+i))
			if err = os.Symlink(tgt, n); err != nil {
				return
			}
			if err = lchtimes(n, now, now); err != nil {
				return
			}
			syms = append(syms, n)
		}
	}

	ts := now.Add(3 * time.Second)
	for i := 1; i < 3; i++ {
		n := filepath.Join(root, "b", string('0'+i))
		if err = lchtimes(n, ts, ts); err != nil {
			return
		}
	}
	return
}

func mkdir(path string) error {
	return os.MkdirAll(path, 0777)
}

func lchtimes(name string, atime time.Time, mtime time.Time) error {
	ts := []syscall.Timespec{
		syscall.NsecToTimespec(atime.UnixNano()),
		syscall.NsecToTimespec(mtime.UnixNano()),
	}
	if err := lutimesNano(name, ts); err != nil {
		return &os.PathError{
			Op:   "lchtimes",
			Path: name,
			Err:  err,
		}
	}
	return nil
}

func sameFile(a, b string) bool {
	fi1, err := hiiragi.Lstat(a)
	if err != nil {
		return false
	}
	fi2, err := hiiragi.Lstat(b)
	if err != nil {
		return false
	}
	return hiiragi.SameFile(fi1, fi2)
}

func tempDir() (string, error) {
	return ioutil.TempDir("", "hiiragi.test")
}

func touch(name string) error {
	return ioutil.WriteFile(name, []byte{}, 0666)
}

var whenTests = []struct {
	in  hiiragi.When
	out string
}{
	{hiiragi.When(0), "None"},
	{hiiragi.Oldest, "Oldest"},
	{hiiragi.Latest, "Latest"},
	{hiiragi.When(9), "When(9)"},
}

func TestWhen(t *testing.T) {
	for _, tt := range whenTests {
		if g, e := tt.in.String(), tt.out; g != e {
			t.Errorf("expected %q, got %q", e, g)
		}
	}
}
