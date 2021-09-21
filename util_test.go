//
// hiiragi :: util_test.go
//
//   Copyright (c) 2016-2021 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package hiiragi_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hattya/hiiragi"
)

func TestSum(t *testing.T) {
	dir, err := tempDir()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	// file
	n := filepath.Join(dir, "1")
	if err := touch(n); err != nil {
		t.Fatal(err)
	}
	h, err := hiiragi.Sum(n)
	if err != nil {
		t.Fatal(err)
	}
	if g, e := h, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"; g != e {
		t.Errorf("expected %v, got %v", e, g)
	}
	// directory
	n = dir
	_, err = hiiragi.Sum(n)
	if err == nil {
		t.Error("expected error")
	}
	// not exist
	n = filepath.Join(dir, "2")
	_, err = hiiragi.Sum(n)
	if err == nil {
		t.Error("expected error")
	}
}

func TestStat(t *testing.T) {
	dir, err := tempDir()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	f1 := filepath.Join(dir, "1")
	if err := touch(f1); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{f1, dir} {
		fi1, err := hiiragi.Stat(n)
		if err != nil {
			t.Fatal(err)
		}
		fi2, err := hiiragi.Lstat(n)
		if err != nil {
			t.Fatal(err)
		}
		if g, e := hiiragi.SameAttrs(fi1, fi2), true; g != e {
			t.Errorf("expected %v, got %v", e, g)
		}
	}
	// not exist
	f2 := filepath.Join(dir, "2")
	_, err = hiiragi.Stat(f2)
	if err == nil {
		t.Error("expected error")
	}
	_, err = hiiragi.Lstat(f2)
	if err == nil {
		t.Error("expected error")
	}
}
