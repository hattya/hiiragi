//
// hiiragi :: util_windows.go
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

package hiiragi

import (
	"os"
	"sync"
	"syscall"
)

func SameAttrs(fi1, fi2 FileInfoEx) bool {
	fs1, ok1 := fi1.(*fileStatEx)
	fs2, ok2 := fi2.(*fileStatEx)
	if !ok1 || !ok2 {
		return false
	}
	if err := fs1.load(); err != nil {
		return false
	}
	if err := fs2.load(); err != nil {
		return false
	}
	sys1 := fi1.Sys().(*syscall.Win32FileAttributeData)
	sys2 := fi2.Sys().(*syscall.Win32FileAttributeData)
	return fs1.vol == fs2.vol && sys1.FileAttributes == sys2.FileAttributes
}

func SameFile(fi1, fi2 FileInfoEx) bool {
	fs1, ok1 := fi1.(*fileStatEx)
	fs2, ok2 := fi2.(*fileStatEx)
	if !ok1 || !ok2 {
		return false
	}
	if err := fs1.load(); err != nil {
		return false
	}
	if err := fs2.load(); err != nil {
		return false
	}
	return fs1.vol == fs2.vol && fs1.idx == fs2.idx
}

type fileStatEx struct {
	os.FileInfo

	sync.Mutex
	path  string
	vol   uint32
	nlink uint32
	idx   uint64
}

func (fs *fileStatEx) load() error {
	fs.Lock()
	defer fs.Unlock()
	if fs.nlink != 0 {
		return nil
	}

	p, err := syscall.UTF16PtrFromString(fs.path)
	if err != nil {
		return err
	}
	h, err := syscall.CreateFile(p, 0, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_BACKUP_SEMANTICS|syscall.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return &os.PathError{
			Op:   "CreateFile",
			Path: fs.path,
			Err:  err,
		}
	}
	defer syscall.CloseHandle(h)

	var fi syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(h, &fi); err != nil {
		return &os.PathError{
			Op:   "GetFileInformationByHandle",
			Path: fs.path,
			Err:  err,
		}
	}
	fs.vol = fi.VolumeSerialNumber
	fs.nlink = fi.NumberOfLinks
	fs.idx = uint64(fi.FileIndexHigh)<<32 | uint64(fi.FileIndexLow)
	return nil
}

func (fs *fileStatEx) Dev() (uint64, error) {
	if err := fs.load(); err != nil {
		return 0, err
	}
	return uint64(fs.vol), nil
}

func (fs *fileStatEx) Nlink() (uint64, error) {
	if err := fs.load(); err != nil {
		return 0, err
	}
	return uint64(fs.nlink), nil
}
