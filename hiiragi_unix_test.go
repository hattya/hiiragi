//
// hiiragi :: hiiragi_unix_test.go
//
//   Copyright (c) 2016-2023 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

//go:build unix

package hiiragi_test

import (
	"time"

	"golang.org/x/sys/unix"
)

func lutimes(name string, atime, mtime time.Time) error {
	return unix.Lutimes(name, []unix.Timeval{
		unix.NsecToTimeval(atime.UnixNano()),
		unix.NsecToTimeval(mtime.UnixNano()),
	})
}
