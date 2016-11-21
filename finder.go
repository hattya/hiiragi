//
// hiiragi :: finder.go
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

package hiiragi

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/hattya/go.cli"
)

type Finder struct {
	ui *cli.CLI
	db *DB
	c  chan FileInfoEx
	wg sync.WaitGroup
	mu sync.Mutex
	p  *counter
}

func NewFinder(ui *cli.CLI, db *DB) *Finder {
	f := &Finder{
		ui: ui,
		db: db,
		c:  make(chan FileInfoEx),
		p:  newCounter(ui, "scan"),
	}
	go f.update()
	return f
}

func (f *Finder) update() {
	f.wg.Add(1)
	for fi := range f.c {
		f.mu.Lock()
		f.p.Update(1)
		if err := f.db.Update(fi); err != nil {
			f.p.Clear()
			f.ui.Errorln("error:", err)
		}
		f.mu.Unlock()
	}
	f.wg.Done()
}

func (f *Finder) Close() {
	close(f.c)
	f.wg.Wait()

	f.p.Update(0)
	f.p.Close()
}

func (f *Finder) Walk(root string) {
	filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		switch {
		case err != nil:
			f.mu.Lock()
			f.p.Clear()
			f.ui.Errorln("error:", err)
			f.mu.Unlock()
		case fi.Mode()&os.ModeType == 0:
			f.c <- &fileStatEx{
				FileInfo: fi,
				path:     path,
			}
		}
		return nil
	})
}
