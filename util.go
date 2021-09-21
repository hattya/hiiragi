//
// hiiragi :: util.go
//
//   Copyright (c) 2016-2021 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
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
