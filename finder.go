//
// hiiragi :: finder.go
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

func (f *Finder) Walk(root string) error {
	if err := f.db.Begin(); err != nil {
		return err
	}
	defer f.db.Rollback()

	f.p.Show = f.Progress
	filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
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

	return f.db.Commit()
}

func (f *Finder) error(err error) {
	f.p.Clear()
	f.ui.Errorln("error:", err)
}
