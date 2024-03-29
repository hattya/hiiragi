//
// hiiragi :: util_unix.go
//
//   Copyright (c) 2016-2023 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

//go:build unix

package hiiragi

import (
	"io/fs"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func Link(oldname, newname string) error {
	return unix.Linkat(unix.AT_FDCWD, oldname, unix.AT_FDCWD, newname, 0)
}

func SameAttrs(fi1, fi2 FileInfoEx) bool {
	sys1 := fi1.Sys().(*syscall.Stat_t)
	sys2 := fi2.Sys().(*syscall.Stat_t)
	return sys1.Dev == sys2.Dev && sys1.Mode == sys2.Mode && sys1.Uid == sys2.Uid && sys1.Gid == sys2.Gid
}

func SameFile(fi1, fi2 FileInfoEx) bool {
	fs1, ok1 := fi1.(*fileStatEx)
	fs2, ok2 := fi2.(*fileStatEx)
	if !ok1 || !ok2 {
		return false
	}
	return os.SameFile(fs1.FileInfo, fs2.FileInfo)
}

type fileStatEx struct {
	fs.FileInfo

	path string
}

func (fs *fileStatEx) Dev() (uint64, error) {
	return uint64(fs.Sys().(*syscall.Stat_t).Dev), nil
}

func (fs *fileStatEx) Nlink() (uint64, error) {
	return uint64(fs.Sys().(*syscall.Stat_t).Nlink), nil
}
