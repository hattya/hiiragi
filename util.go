//
// hiiragi :: util.go
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

package hiiragi

import (
	"crypto"
	_ "crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"time"
)

func exists(name string) bool {
	_, err := os.Lstat(name)
	return err == nil
}

func Sum(name string) (hash string, err error) {
	f, err := os.Open(name)
	if err != nil {
		return
	}
	defer f.Close()

	h := crypto.SHA256.New()
	if _, err = io.Copy(h, f); err != nil {
		return
	}
	hash = hex.EncodeToString(h.Sum(nil))
	return
}

type FileInfoEx interface {
	os.FileInfo

	Path() string
	Dev() (uint64, error)
	Nlink() (uint64, error)
}

func Lstat(name string) (FileInfoEx, error) {
	fi, err := os.Lstat(name)
	if err != nil {
		return nil, err
	}
	return &fileStatEx{
		FileInfo: fi,
		path:     name,
	}, nil
}

func Stat(name string) (FileInfoEx, error) {
	fi, err := os.Stat(name)
	if err != nil {
		return nil, err
	}
	return &fileStatEx{
		FileInfo: fi,
		path:     name,
	}, nil
}

func (fs *fileStatEx) ModTime() time.Time {
	return fs.FileInfo.ModTime().Truncate(time.Second)
}

func (fs *fileStatEx) Path() string {
	return fs.path
}

type FileInfoExSlice []FileInfoEx

func (p FileInfoExSlice) Len() int           { return len(p) }
func (p FileInfoExSlice) Less(i, j int) bool { return p[i].Name() < p[j].Name() }
func (p FileInfoExSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
