//
// hiiragi :: hiiragi_windows_test.go
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
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows"
)

func init() {
	dir, err := tempDir()
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	if err = os.Symlink("target", filepath.Join(dir, "symlink")); err != nil {
		switch err.(*os.LinkError).Err {
		case syscall.EWINDOWS, syscall.ERROR_PRIVILEGE_NOT_HELD:
			supportsSymlinks = false
		}
	}
}

func lutimesNano(path string, ts []syscall.Timespec) error {
	h, err := createFile(path)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)

	a := windows.NsecToFiletime(windows.TimespecToNsec(windows.Timespec(ts[0])))
	w := windows.NsecToFiletime(windows.TimespecToNsec(windows.Timespec(ts[1])))
	return windows.SetFileTime(h, nil, &a, &w)
}

func sameFile(a, b string) bool {
	h1, err := createFile(a)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h1)

	h2, err := createFile(b)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h2)

	var fi1, fi2 windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(h1, &fi1); err != nil {
		return false
	}
	if err := windows.GetFileInformationByHandle(h2, &fi2); err != nil {
		return false
	}
	return fi1.VolumeSerialNumber == fi2.VolumeSerialNumber && fi1.FileIndexHigh == fi2.FileIndexHigh && fi1.FileIndexLow == fi2.FileIndexLow
}

func createFile(path string) (windows.Handle, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return windows.InvalidHandle, err
	}
	return windows.CreateFile(p, windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
}
