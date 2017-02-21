//
// hiiragi :: util_unix.go
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

// +build !plan9,!windows

package hiiragi

import (
	"os"
	"syscall"
)

func SameAttrs(fi1, fi2 FileInfoEx) bool {
	sys1 := fi1.Sys().(*syscall.Stat_t)
	sys2 := fi2.Sys().(*syscall.Stat_t)
	return sys1.Dev == sys2.Dev && sys1.Mode == sys2.Mode && sys1.Uid == sys2.Uid && sys1.Gid == sys2.Gid
}

type fileStatEx struct {
	os.FileInfo

	path string
}

func (fs *fileStatEx) Dev() (uint64, error) {
	return uint64(fs.Sys().(*syscall.Stat_t).Dev), nil
}

func (fs *fileStatEx) Nlink() (uint64, error) {
	return uint64(fs.Sys().(*syscall.Stat_t).Nlink), nil
}
