//
// hiiragi :: hiiragi.go
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
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/hattya/go.cli"
)

const Version = "0.0+"

type Deduper struct {
	Attrs    bool
	Mtime    When
	Name     bool
	Pretend  bool
	Progress bool

	ui  *cli.CLI
	db  *DB
	p   *counter
	pid int
	i   int
}

func NewDeduper(ui *cli.CLI, db *DB) *Deduper {
	return &Deduper{
		Attrs:    true,
		Name:     true,
		Progress: true,
		ui:       ui,
		db:       db,
		p:        newCounter(ui, "dedup"),
		pid:      os.Getpid(),
	}
}

func (d *Deduper) All(ctx context.Context) error {
	// file
	f, n, err := d.db.NumFiles()
	if err != nil {
		return err
	}
	d.p.N = n
	// symlink
	s, n, err := d.db.NumSymlinks()
	if err != nil {
		return err
	}
	d.p.N += n
	d.p.Set(f + s)

	d.p.Show = d.Progress
	d.p.Update(0)
	defer d.p.Close()

	if err = d.files(ctx); err != nil {
		return err
	}
	return d.symlinks(ctx)
}

func (d *Deduper) Files(ctx context.Context) error {
	// file
	done, n, err := d.db.NumFiles()
	if err != nil {
		return err
	}
	d.p.N = n
	d.p.Set(done)

	d.p.Show = d.Progress
	d.p.Update(0)
	defer d.p.Close()

	return d.files(ctx)
}

func (d *Deduper) files(ctx context.Context) error {
	defer d.db.Rollback()

	mtime, order := d.mtime()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		files, err := d.db.NextFiles(mtime, order)
		switch {
		case err != nil || len(files) == 0:
			return err
		case len(files) == 1:
			if err = d.skip(files[0].Path); err != nil {
				return err
			}
			continue
		}

		if err = d.db.Begin(); err != nil {
			return err
		}

		hash := make(map[string][]FileInfoEx)
		for _, f := range files {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			fi, err := Lstat(f.Path)
			switch {
			case err != nil:
				return err
			case fi.Mode()&os.ModeType != 0 || fi.Size() != f.Size || !fi.ModTime().Equal(f.Mtime):
				if err = d.skip(f.Path); err != nil {
					return err
				}
				continue
			}

			h, err := Sum(f.Path)
			if err != nil {
				return err
			}
			hash[h] = append(hash[h], fi)
		}
		for _, v := range hash {
			if err = d.dedup(ctx, v); err != nil {
				return err
			}
		}

		if err = d.db.Commit(); err != nil {
			return err
		}
	}
}

func (d *Deduper) Symlinks(ctx context.Context) error {
	// symlink
	done, n, err := d.db.NumSymlinks()
	if err != nil {
		return err
	}
	d.p.N = n
	d.p.Set(done)

	d.p.Show = d.Progress
	d.p.Update(0)
	defer d.p.Close()

	return d.symlinks(ctx)
}

func (d *Deduper) symlinks(ctx context.Context) error {
	defer d.db.Rollback()

	mtime, order := d.mtime()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		syms, err := d.db.NextSymlinks(mtime, order)
		switch {
		case err != nil || len(syms) == 0:
			return err
		case len(syms) == 1:
			if err = d.skip(syms[0].Path); err != nil {
				return err
			}
			continue
		}

		if err = d.db.Begin(); err != nil {
			return err
		}

		var v []FileInfoEx
		for _, s := range syms {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			fi, err := Lstat(s.Path)
			switch {
			case err != nil:
				return err
			case fi.Mode()&os.ModeType != os.ModeSymlink || !fi.ModTime().Equal(s.Mtime):
				if err = d.skip(s.Path); err != nil {
					return err
				}
				continue
			}

			switch t, err := os.Readlink(s.Path); {
			case err != nil:
				return err
			case t != s.Target:
				if err = d.skip(s.Path); err != nil {
					return err
				}
				continue
			}
			v = append(v, fi)
		}
		if err = d.dedup(ctx, v); err != nil {
			return err
		}

		if err = d.db.Commit(); err != nil {
			return err
		}
	}
}

func (d *Deduper) mtime() (bool, Order) {
	mtime := d.Mtime == 0
	order := Asc
	if d.Mtime == Latest {
		order = Desc
	}
	return mtime, order
}

func (d *Deduper) dedup(ctx context.Context, list []FileInfoEx) (err error) {
	var src FileInfoEx
	if d.Name {
		sort.SliceStable(list, func(i, j int) bool { return list[i].Name() < list[j].Name() })
	}
	for _, dst := range list {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		switch {
		case src == nil || (d.Name && src.Name() != dst.Name()):
			src = dst
			d.i = 0
		case SameFile(src, dst):
			// skip
		case !d.Attrs || SameAttrs(src, dst):
			if err = d.link(src.Path(), dst.Path()); err != nil {
				return
			}
		}
		if err = d.db.Done(dst.Path()); err != nil {
			return
		}
		d.p.Update(1)
	}
	return
}

func (d *Deduper) link(src, dst string) (err error) {
	d.p.Clear()
	if d.i == 0 {
		d.ui.Println(">>", src)
	}
	d.ui.Println(" +", dst)

	if !d.Pretend {
		var tmp string
		for {
			d.i++
			tmp = fmt.Sprintf("%v.%v_%v", dst, d.pid, d.i)
			if !exists(tmp) {
				break
			}
		}
		defer os.Rename(tmp, dst)

		if err = os.Rename(dst, tmp); err != nil {
			return
		}
		if err = os.Link(src, dst); err != nil {
			return
		}
		err = os.Remove(tmp)
	}
	return
}

func (d *Deduper) skip(name string) error {
	if err := d.db.Done(name); err != nil {
		return err
	}
	d.p.Update(1)
	return nil
}

type When uint

const (
	Oldest When = 1 + iota
	Latest
)

func (w When) String() string {
	switch w {
	case 0:
		return "None"
	case Oldest:
		return "Oldest"
	case Latest:
		return "Latest"
	}
	return fmt.Sprintf("When(%d)", w)
}
