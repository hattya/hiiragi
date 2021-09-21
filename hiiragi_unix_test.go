//
// hiiragi :: hiiragi_unix_test.go
//
//   Copyright (c) 2016-2021 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

// +build !plan9,!windows

package hiiragi_test

import (
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
