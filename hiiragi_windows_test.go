//
// hiiragi :: hiiragi_windows_test.go
//
//   Copyright (c) 2016-2022 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package hiiragi_test

import (
	"os"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

func init() {
	dir, err := os.MkdirTemp("", "hiiragi.test")
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

func lutimes(name string, atime, mtime time.Time) error {
	p, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	h, err := windows.CreateFile(p, windows.FILE_WRITE_ATTRIBUTES, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	a := windows.NsecToFiletime(atime.UnixNano())
	w := windows.NsecToFiletime(mtime.UnixNano())
	return windows.SetFileTime(h, nil, &a, &w)
}
