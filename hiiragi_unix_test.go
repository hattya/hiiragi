//
// hiiragi :: hiiragi_unix_test.go
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

// +build !plan9,!windows

package hiiragi_test

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func lutimesNano(path string, ts []syscall.Timespec) error {
	ut := []unix.Timespec{
		unix.Timespec(ts[0]),
		unix.Timespec(ts[1]),
	}
	return unix.UtimesNanoAt(unix.AT_FDCWD, path, ut, unix.AT_SYMLINK_NOFOLLOW)
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
