//
// hiiragi :: util_test.go
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
		if e, g := hiiragi.SameAttrs(fi1, fi2), true; g != e {
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
