//
// hiiragi :: hiiragi_test.go
//
//   Copyright (c) 2016-2025 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package hiiragi_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/hattya/go.cli"
	"github.com/hattya/hiiragi"
)

var supportsSymlinks = true

func TestDedupAll(t *testing.T) {
	list, err := dedup(t, "all", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}

	files := list[:8]
	if sameFile(files[0], files[1]) {
		t.Error("files should be different")
	}
	if !sameFile(files[0], files[2]) {
		t.Error("files should be same")
	}
	if sameFile(files[0], files[4]) {
		t.Error("files should be different")
	}
	if sameFile(files[0], files[6]) {
		t.Error("files should be different")
	}
	if !sameFile(files[1], files[3]) {
		t.Error("files should be same")
	}
	if sameFile(files[1], files[5]) {
		t.Error("files should be different")
	}
	if sameFile(files[1], files[7]) {
		t.Error("files should be different")
	}
	syms := list[8:]
	if len(syms) > 0 {
		if sameFile(syms[0], syms[1]) {
			t.Error("symlinks should be different")
		}
		if !sameFile(syms[0], syms[2]) {
			t.Error("symlinks should be same")
		}
		if sameFile(syms[0], syms[4]) {
			t.Error("symlinks should be different")
		}
		if !sameFile(syms[1], syms[3]) {
			t.Error("symlinks should be same")
		}
		if sameFile(syms[1], syms[5]) {
			t.Error("symlinks should be different")
		}
	}
}

func TestDedupAllPretend(t *testing.T) {
	list, err := dedup(t, "all", map[string]any{
		"pretend": true,
	})
	if err != nil {
		t.Fatal(err)
	}

	files := list[:8]
	if sameFile(files[0], files[1]) {
		t.Error("files should be different")
	}
	if sameFile(files[0], files[2]) {
		t.Error("files should be different")
	}
	if sameFile(files[0], files[4]) {
		t.Error("files should be different")
	}
	if sameFile(files[0], files[6]) {
		t.Error("files should be different")
	}
	if sameFile(files[1], files[3]) {
		t.Error("files should be different")
	}
	if sameFile(files[1], files[5]) {
		t.Error("files should be different")
	}
	if sameFile(files[1], files[7]) {
		t.Error("files should be different")
	}
	syms := list[8:]
	if len(syms) > 0 {
		if sameFile(syms[0], syms[1]) {
			t.Error("symlinks should be different")
		}
		if sameFile(syms[0], syms[2]) {
			t.Error("symlinks should be different")
		}
		if sameFile(syms[0], syms[4]) {
			t.Error("symlinks should be different")
		}
		if sameFile(syms[1], syms[3]) {
			t.Error("symlinks should be different")
		}
		if sameFile(syms[1], syms[5]) {
			t.Error("symlinks should be different")
		}
	}
}

func TestDedupAllIgnoreAll(t *testing.T) {
	list, err := dedup(t, "all", map[string]any{
		"attrs": false,
		"mtime": hiiragi.Oldest,
		"name":  false,
	})
	if err != nil {
		t.Fatal(err)
	}

	files := list[:8]
	if !sameFile(files[0], files[1]) {
		t.Error("files should be same")
	}
	if !sameFile(files[0], files[2]) {
		t.Error("files should be same")
	}
	if !sameFile(files[0], files[4]) {
		t.Error("files should be same")
	}
	if !sameFile(files[0], files[6]) {
		t.Error("files should be same")
	}
	if !sameFile(files[1], files[3]) {
		t.Error("files should be same")
	}
	if !sameFile(files[1], files[5]) {
		t.Error("files should be same")
	}
	if !sameFile(files[1], files[7]) {
		t.Error("files should be same")
	}
	syms := list[8:]
	if len(syms) > 0 {
		if !sameFile(syms[0], syms[1]) {
			t.Error("symlinks should be same")
		}
		if !sameFile(syms[0], syms[2]) {
			t.Error("symlinks should be same")
		}
		if !sameFile(syms[0], syms[4]) {
			t.Error("symlinks should be same")
		}
		if !sameFile(syms[1], syms[3]) {
			t.Error("symlinks should be same")
		}
		if !sameFile(syms[1], syms[5]) {
			t.Error("symlinks should be same")
		}
	}
}

func TestDedupAllInterrupt(t *testing.T) {
	_, err := dedup(t, "all", map[string]any{
		"interrupt": true,
	})
	if err != context.Canceled {
		t.Error(err)
	}
}

func TestDedupNoFiles(t *testing.T) {
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
	var files []string
	now := time.Now().Truncate(time.Second)
	ts := now.Add(-3 * time.Second)
	for i := rune('1'); i < '4'; i++ {
		n := filepath.Join(root, string(i))
		if err := touch(n); err != nil {
			t.Fatal(err)
		}
		if err := lutimes(n, ts, ts); err != nil {
			t.Fatal(err)
		}
		files = append(files, n)
		ts = now
	}

	ui := cli.NewCLI()
	ui.Stdout = io.Discard
	ui.Stderr = io.Discard

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
	if err := lutimes(files[1], ts, ts); err != nil {
		t.Fatal(err)
	}
	// change data
	ts = now
	if err := file(files[2], "data\n"); err != nil {
		t.Fatal(err)
	}
	if err := lutimes(files[2], ts, ts); err != nil {
		t.Fatal(err)
	}

	d := hiiragi.NewDeduper(ui, db)
	if err := d.Files(ctx); err != nil {
		t.Fatal(err)
	}
	if sameFile(files[0], files[1]) {
		t.Error("files should be different")
	}
	if sameFile(files[0], files[2]) {
		t.Error("files should be different")
	}
	if sameFile(files[1], files[2]) {
		t.Error("files should be different")
	}
}

func TestDedupFiles(t *testing.T) {
	files, err := dedup(t, "files", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}

	if sameFile(files[0], files[1]) {
		t.Error("files should be different")
	}
	if !sameFile(files[0], files[2]) {
		t.Error("files should be same")
	}
	if sameFile(files[0], files[4]) {
		t.Error("files should be different")
	}
	if sameFile(files[0], files[6]) {
		t.Error("files should be different")
	}
	if !sameFile(files[1], files[3]) {
		t.Error("files should be same")
	}
	if sameFile(files[1], files[5]) {
		t.Error("files should be different")
	}
	if sameFile(files[1], files[7]) {
		t.Error("files should be different")
	}
}

func TestDedupFilesIgnoreAll(t *testing.T) {
	files, err := dedup(t, "files", map[string]any{
		"attrs": false,
		"mtime": hiiragi.Oldest,
		"name":  false,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !sameFile(files[0], files[1]) {
		t.Error("files should be same")
	}
	if !sameFile(files[0], files[2]) {
		t.Error("files should be same")
	}
	if !sameFile(files[0], files[4]) {
		t.Error("files should be same")
	}
	if !sameFile(files[0], files[6]) {
		t.Error("files should be same")
	}
	if !sameFile(files[1], files[3]) {
		t.Error("files should be same")
	}
	if !sameFile(files[1], files[5]) {
		t.Error("files should be same")
	}
	if !sameFile(files[1], files[7]) {
		t.Error("files should be same")
	}
}

func TestDedupFilesIgnoreAttrs(t *testing.T) {
	files, err := dedup(t, "files", map[string]any{
		"attrs": false,
	})
	if err != nil {
		t.Fatal(err)
	}

	if sameFile(files[0], files[1]) {
		t.Error("files should be different")
	}
	if !sameFile(files[0], files[2]) {
		t.Error("files should be same")
	}
	if sameFile(files[0], files[4]) {
		t.Error("files should be different")
	}
	if !sameFile(files[0], files[6]) {
		t.Error("files should be same")
	}
	if !sameFile(files[1], files[3]) {
		t.Error("files should be same")
	}
	if sameFile(files[1], files[5]) {
		t.Error("files should be different")
	}
	if !sameFile(files[1], files[7]) {
		t.Error("files should be same")
	}
}

func TestDedupFilesIgnoreMtime(t *testing.T) {
	files, err := dedup(t, "files", map[string]any{
		"mtime": hiiragi.Oldest,
	})
	if err != nil {
		t.Fatal(err)
	}

	if sameFile(files[0], files[1]) {
		t.Error("files should be different")
	}
	if !sameFile(files[0], files[2]) {
		t.Error("files should be same")
	}
	if !sameFile(files[0], files[4]) {
		t.Error("files should be same")
	}
	if sameFile(files[0], files[6]) {
		t.Error("files should be different")
	}
	if !sameFile(files[1], files[3]) {
		t.Error("files should be same")
	}
	if !sameFile(files[1], files[5]) {
		t.Error("files should be same")
	}
	if sameFile(files[1], files[7]) {
		t.Error("files should be different")
	}
}

func TestDedupFilesIgnoreName(t *testing.T) {
	files, err := dedup(t, "files", map[string]any{
		"name": false,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !sameFile(files[0], files[1]) {
		t.Error("files should be same")
	}
	if !sameFile(files[0], files[2]) {
		t.Error("files should be same")
	}
	if sameFile(files[0], files[4]) {
		t.Error("files should be different")
	}
	if sameFile(files[0], files[6]) {
		t.Error("files should be different")
	}
	if !sameFile(files[1], files[3]) {
		t.Error("files should be same")
	}
	if sameFile(files[1], files[5]) {
		t.Error("files should be different")
	}
	if sameFile(files[1], files[7]) {
		t.Error("files should be different")
	}
}

func TestDedupFilesPretend(t *testing.T) {
	files, err := dedup(t, "files", map[string]any{
		"pretend": true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if sameFile(files[0], files[1]) {
		t.Error("files should be different")
	}
	if sameFile(files[0], files[2]) {
		t.Error("files should be different")
	}
	if sameFile(files[0], files[4]) {
		t.Error("files should be different")
	}
	if sameFile(files[0], files[6]) {
		t.Error("files should be different")
	}
	if sameFile(files[1], files[3]) {
		t.Error("files should be different")
	}
	if sameFile(files[1], files[5]) {
		t.Error("files should be different")
	}
	if sameFile(files[1], files[7]) {
		t.Error("files should be different")
	}
}

func TestDedupFilesInterrupt(t *testing.T) {
	_, err := dedup(t, "files", map[string]any{
		"interrupt": true,
	})
	if err != context.Canceled {
		t.Error(err)
	}
}

func TestDedupNoSymlinks(t *testing.T) {
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
	tgt := filepath.Join(dir, "target")
	if err := touch(tgt); err != nil {
		t.Fatal(err)
	}
	var syms []string
	now := time.Now().Truncate(time.Second)
	ts := now.Add(-3 * time.Second)
	for i := rune('1'); i < '4'; i++ {
		n := filepath.Join(root, string(i))
		if err := os.Symlink(tgt, n); err != nil {
			t.Fatal(err)
		}
		if err := lutimes(n, ts, ts); err != nil {
			t.Fatal(err)
		}
		syms = append(syms, n)
		ts = now
	}

	ui := cli.NewCLI()
	ui.Stdout = io.Discard
	ui.Stderr = io.Discard

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
	if err := lutimes(syms[1], ts, ts); err != nil {
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
	if err := lutimes(syms[2], ts, ts); err != nil {
		t.Fatal(err)
	}

	d := hiiragi.NewDeduper(ui, db)
	if err := d.Symlinks(ctx); err != nil {
		t.Fatal(err)
	}
	if sameFile(syms[0], syms[1]) {
		t.Error("symlinks should be different")
	}
	if sameFile(syms[0], syms[2]) {
		t.Error("symlinks should be different")
	}
	if sameFile(syms[1], syms[2]) {
		t.Error("symlinks should be different")
	}
}

func TestDedupSymlinks(t *testing.T) {
	if !supportsSymlinks {
		t.Skipf("skipping on %v", runtime.GOOS)
	}

	syms, err := dedup(t, "symlinks", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if sameFile(syms[0], syms[1]) {
		t.Error("symlinks should be different")
	}
	if !sameFile(syms[0], syms[2]) {
		t.Error("symlinks should be same")
	}
	if sameFile(syms[0], syms[4]) {
		t.Error("symlinks should be different")
	}
	if !sameFile(syms[1], syms[3]) {
		t.Error("symlinks should be same")
	}
	if sameFile(syms[1], syms[5]) {
		t.Error("symlinks should be different")
	}
}

func TestDedupSymlinksIgnoreAll(t *testing.T) {
	if !supportsSymlinks {
		t.Skipf("skipping on %v", runtime.GOOS)
	}

	syms, err := dedup(t, "symlinks", map[string]any{
		"attrs": false,
		"mtime": hiiragi.Oldest,
		"name":  false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !sameFile(syms[0], syms[1]) {
		t.Error("symlinks should be same")
	}
	if !sameFile(syms[0], syms[2]) {
		t.Error("symlinks should be same")
	}
	if !sameFile(syms[0], syms[4]) {
		t.Error("symlinks should be same")
	}
	if !sameFile(syms[1], syms[3]) {
		t.Error("symlinks should be same")
	}
	if !sameFile(syms[1], syms[5]) {
		t.Error("symlinks should be same")
	}
}

func TestDedupSymlinksIgnoreName(t *testing.T) {
	if !supportsSymlinks {
		t.Skipf("skipping on %v", runtime.GOOS)
	}

	syms, err := dedup(t, "symlinks", map[string]any{
		"name": false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !sameFile(syms[0], syms[1]) {
		t.Error("symlinks should be same")
	}
	if !sameFile(syms[0], syms[2]) {
		t.Error("symlinks should be same")
	}
	if sameFile(syms[0], syms[4]) {
		t.Error("symlinks should be different")
	}
	if !sameFile(syms[1], syms[3]) {
		t.Error("symlinks should be same")
	}
	if sameFile(syms[1], syms[5]) {
		t.Error("symlinks should be different")
	}
}

func TestDedupSymlinksPretend(t *testing.T) {
	if !supportsSymlinks {
		t.Skipf("skipping on %v", runtime.GOOS)
	}

	syms, err := dedup(t, "symlinks", map[string]any{
		"pretend": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if sameFile(syms[0], syms[1]) {
		t.Error("symlinks should be different")
	}
	if sameFile(syms[0], syms[2]) {
		t.Error("symlinks should be different")
	}
	if sameFile(syms[0], syms[4]) {
		t.Error("symlinks should be different")
	}
	if sameFile(syms[1], syms[3]) {
		t.Error("symlinks should be different")
	}
	if sameFile(syms[1], syms[5]) {
		t.Error("symlinks should be different")
	}
}

func TestDedupSymlinksInterrupt(t *testing.T) {
	if !supportsSymlinks {
		t.Skipf("skipping on %v", runtime.GOOS)
	}

	_, err := dedup(t, "symlinks", map[string]any{
		"interrupt": true,
	})
	if err != context.Canceled {
		t.Error(err)
	}
}

func dedup(t *testing.T, action string, opts map[string]any) (list []string, err error) {
	dir := t.TempDir()
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
	ui.Stdout = io.Discard
	ui.Stderr = io.Discard

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
		for i := rune('1'); i < '3'; i++ {
			n := filepath.Join(dir, string(i))
			if err = touch(n); err != nil {
				return
			}
			if err = lutimes(n, now, now); err != nil {
				return
			}
			files = append(files, n)
		}
	}

	ts := now.Add(3 * time.Second)
	for i := rune('1'); i < '3'; i++ {
		n := filepath.Join(root, "b", string(i))
		if err = lutimes(n, ts, ts); err != nil {
			return
		}

		n = filepath.Join(root, "c", string(i))
		if err = os.Chmod(n, 0o444); err != nil {
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
		for i := rune('1'); i < '3'; i++ {
			n := filepath.Join(dir, string(i))
			if err = os.Symlink(tgt, n); err != nil {
				return
			}
			if err = lutimes(n, now, now); err != nil {
				return
			}
			syms = append(syms, n)
		}
	}

	ts := now.Add(3 * time.Second)
	for i := rune('1'); i < '3'; i++ {
		n := filepath.Join(root, "b", string(i))
		if err = lutimes(n, ts, ts); err != nil {
			return
		}
	}
	return
}

func mkdir(path string) error {
	return os.MkdirAll(path, 0o777)
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

func file(name, data string) error {
	return os.WriteFile(name, []byte(data), 0o666)
}

func touch(name string) error {
	return os.WriteFile(name, []byte{}, 0o666)
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
