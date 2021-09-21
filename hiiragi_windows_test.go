//
// hiiragi :: hiiragi_windows_test.go
//
//   Copyright (c) 2016-2021 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
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
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	h, err := windows.CreateFile(p, windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)

	a := windows.NsecToFiletime(windows.TimespecToNsec(windows.Timespec(ts[0])))
	w := windows.NsecToFiletime(windows.TimespecToNsec(windows.Timespec(ts[1])))
	return windows.SetFileTime(h, nil, &a, &w)
}
