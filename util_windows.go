//
// hiiragi :: util_windows.go
//
//   Copyright (c) 2016-2021 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
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

	mu    sync.Mutex
	path  string
	vol   uint32
	nlink uint32
	idx   uint64
}

func (fs *fileStatEx) load() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

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
