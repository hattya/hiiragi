//
// hiiragi :: finder.go
//
//   Copyright (c) 2016-2021 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package hiiragi

import (
	"context"
	"os"
	"path/filepath"

	"github.com/hattya/go.cli"
)

type Finder struct {
	Progress bool

	ui *cli.CLI
	db *DB
	p  *counter
}

func NewFinder(ui *cli.CLI, db *DB) *Finder {
	f := &Finder{
		Progress: true,
		ui:       ui,
		db:       db,
		p:        newCounter(ui, "scan"),
	}
	return f
}

func (f *Finder) Close() {
	f.p.Update(0)
	f.p.Close()
}

func (f *Finder) Walk(ctx context.Context, root string) error {
	if err := f.db.Begin(); err != nil {
		return err
	}
	defer f.db.Rollback()

	f.p.Show = f.Progress
	err := filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		switch {
		case err != nil:
			f.error(err)
		case fi.Mode()&(os.ModeType^os.ModeSymlink) == 0:
			fi := &fileStatEx{
				FileInfo: fi,
				path:     path,
			}
			if err := f.db.Update(fi); err != nil {
				switch err.(type) {
				case *os.PathError:
					f.error(err)
				default:
					return err
				}
			}
			f.p.Update(1)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return f.db.Commit()
}

func (f *Finder) error(err error) {
	f.p.Clear()
	f.ui.Errorln("error:", err)
}
