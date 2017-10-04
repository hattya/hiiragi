//
// hiiragi :: db.go
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
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/hattya/go.cli"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	db    *sql.DB
	stmt  map[string]*sql.Stmt
	stack []*scope
}

func Create(name string) (*DB, error) {
	if name != ":memory:" {
		os.Remove(name)
	}
	return Open(name)
}

func Open(name string) (*DB, error) {
	db, err := open(name)
	if err != nil {
		return nil, err
	}
	return &DB{
		db:    db,
		stmt:  make(map[string]*sql.Stmt),
		stack: nil,
	}, nil
}

func (db *DB) Close() error {
	return db.db.Close()
}

func (db *DB) SetCacheSize(size int64) error {
	_, err := db.db.Exec(fmt.Sprintf(`PRAGMA cache_size = %v`, size))
	return err
}

func (db *DB) Begin() error {
	tx, err := db.db.Begin()
	db.stack = append(db.stack, &scope{
		tx:   tx,
		stmt: make(map[string]*sql.Stmt),
	})
	return err
}

func (db *DB) Commit() error {
	s := db.pop()
	if s == nil {
		return nil
	}
	return s.tx.Commit()
}

func (db *DB) Rollback() error {
	s := db.pop()
	if s == nil {
		return nil
	}
	return s.tx.Rollback()
}

func (db *DB) pop() *scope {
	n := len(db.stack) - 1
	if n < 0 {
		return nil
	}
	s := db.stack[n]
	db.stack = db.stack[:n]
	return s
}

func (db *DB) NumFiles() (int64, int64, error) {
	return db.count(`file`)
}

func (db *DB) NumSymlinks() (int64, int64, error) {
	return db.count(`symlink`)
}

func (db *DB) count(t string) (done, n int64, err error) {
	k := "count." + t
	stmt, ok := db.stmt[k]
	if !ok {
		q := cli.Dedent(`
			SELECT done,
			       total
			  FROM master
			 WHERE type = ?
		`)
		if stmt, err = db.prepare(k, q); err != nil {
			return
		}
	}
	err = stmt.QueryRow(t).Scan(&done, &n)
	return
}

func (db *DB) NextFiles(mtime bool, order Order) ([]*File, error) {
	list, err := db.next(&File{}, mtime, order)
	return list.([]*File), err
}

func (db *DB) NextSymlinks(mtime bool, order Order) ([]*Symlink, error) {
	list, err := db.next(&Symlink{}, mtime, order)
	return list.([]*Symlink), err
}

func (db *DB) next(t interface{}, mtime bool, order Order) (list interface{}, err error) {
	tt := reflect.Indirect(reflect.ValueOf(t)).Type()
	col := tt.Field(tt.NumField() - 1).Name
	lv := reflect.MakeSlice(reflect.SliceOf(reflect.PtrTo(tt)), 0, 0)
	list = lv.Interface() // make type assertion simple

	k := "next." + tt.Name()
	if mtime {
		k += ".mtime"
	}
	stmt, ok := db.stmt[k]
	if !ok {
		var b bytes.Buffer
		fmt.Fprintf(&b, cli.Dedent(`
			  WITH next (
			         value,
			         dev,
			         mtime
			       ) AS (
			         SELECT %v,
			                i.dev,
			                i.mtime
			          FROM  %v
			                INNER JOIN info AS i
			                   ON info_id = i.id
			          LIMIT 1
			       )
			SELECT i.path,
			       i.dev,
			       i.nlink,
			       i.mtime,
			       %[1]v
			  FROM %v
			       INNER JOIN info AS i
			          ON info_id = i.id,
			       next AS n
			 WHERE %[1]v   =  n.value
			   AND i.dev   =  n.dev
		`), strings.ToLower(col), strings.ToLower(tt.Name()))
		if mtime {
			b.WriteString(cli.Dedent(`
			   AND i.mtime =  n.mtime
			`))
		}
		if stmt, err = db.prepare(k, b.String()); err != nil {
			return
		}
	}
	rows, err := stmt.Query()
	if err != nil {
		return
	}
	defer rows.Close()

	lv = reflect.MakeSlice(reflect.SliceOf(reflect.PtrTo(tt)), 0, 0)
	for rows.Next() {
		v := reflect.New(tt).Elem()
		dst := make([]interface{}, tt.NumField())
		for i := 0; i < tt.NumField(); i++ {
			dst[i] = v.Field(i).Addr().Interface()
		}
		if err = rows.Scan(dst...); err != nil {
			return
		}
		lv = reflect.Append(lv, v.Addr())
	}
	list = lv.Interface()
	err = rows.Err()
	sortEntries(list, order)
	return
}

func (db *DB) prepare(name, query string) (*sql.Stmt, error) {
	stmt, err := db.db.Prepare(query)
	if err == nil {
		db.stmt[name] = stmt
	}
	return stmt, err
}

func (db *DB) Done(path string) error {
	return db.withTx(func() (err error) {
		s := db.scope()
		for _, t := range []string{"file", "symlink"} {
			k := "Done." + t
			stmt, ok := s.stmt[k]
			if !ok {
				q := fmt.Sprintf(cli.Dedent(`
					DELETE FROM %v
					 WHERE info_id IN (
					         SELECT i.id
					           FROM info AS i
					          WHERE i.path = ?
					       )
				`), t)
				if stmt, err = s.prepare(k, q); err != nil {
					return
				}
			}
			if _, err = stmt.Exec(path); err != nil {
				return
			}
		}
		return
	})
}

func (db *DB) Update(fi FileInfoEx) error {
	dev, err := fi.Dev()
	if err != nil {
		return err
	}
	nlink, err := fi.Nlink()
	if err != nil {
		return err
	}

	return db.withTx(func() (err error) {
		s := db.scope()
		i := "Update.INSERT.info"
		if _, ok := s.stmt[i]; !ok {
			q := cli.Dedent(`
				INSERT INTO info (
				         dev,
				         nlink,
				         mtime,
				         path
				       )
				VALUES (
				         ?,
				         ?,
				         ?,
				         ?
				       )
			`)
			if _, err = s.prepare(i, q); err != nil {
				return
			}
		}
		u := "Update.UPDATE.info"
		if _, ok := s.stmt[u]; !ok {
			q := cli.Dedent(`
				UPDATE info
				   SET dev   = ?,
				       nlink = ?,
				       mtime = ?
				 WHERE path = ?
			`)
			if _, err = s.prepare(u, q); err != nil {
				return
			}
		}
		if err = db.upsert(i, u, dev, nlink, fi.ModTime(), fi.Path()); err != nil {
			return
		}

		var t, col string
		a := make([]interface{}, 2)
		a[1] = fi.Path()
		if fi.Mode()&os.ModeType == 0 {
			// file
			t = "file"
			col = "size"
			a[0] = fi.Size()
		} else {
			// symlink
			t = "symlink"
			col = "target"
			if a[0], err = os.Readlink(fi.Path()); err != nil {
				return
			}
		}
		i = "Update.INSERT." + t
		if _, ok := s.stmt[i]; !ok {
			q := fmt.Sprintf(cli.Dedent(`
				INSERT INTO %v (
				         info_id,
				         %v
				       )
				SELECT id,
				       ?
				  FROM info
				 WHERE path = ?
			`), t, col)
			if _, err = s.prepare(i, q); err != nil {
				return
			}
		}
		u = "Update.UPDATE." + t
		if _, ok := s.stmt[u]; !ok {
			q := fmt.Sprintf(cli.Dedent(`
				UPDATE %v
				   SET %v = ?
				 WHERE info_id IN (
				         SELECT i.id
				           FROM info AS i
				          WHERE i.path = ?
				       )
			`), t, col)
			if _, err = s.prepare(u, q); err != nil {
				return
			}
		}
		return db.upsert(i, u, a...)
	})
}

func (db *DB) scope() *scope {
	n := len(db.stack) - 1
	if n < 0 {
		return nil
	}
	return db.stack[n]
}

func (db *DB) upsert(insert, update string, a ...interface{}) (err error) {
	s := db.scope()
	res, err := s.stmt[update].Exec(a...)
	if err != nil {
		return
	}
	n, err := res.RowsAffected()
	if err == nil && n < 1 {
		res, err = s.stmt[insert].Exec(a...)
	}
	return
}

func (db *DB) withTx(fn func() error) (err error) {
	auto := db.scope() == nil
	if auto {
		if err = db.Begin(); err != nil {
			return
		}
		defer db.Rollback()
	}
	if err = fn(); err != nil {
		return
	}
	if auto {
		return db.Commit()
	}
	return nil
}

type scope struct {
	tx   *sql.Tx
	stmt map[string]*sql.Stmt
}

func (s *scope) prepare(name, query string) (*sql.Stmt, error) {
	stmt, err := s.tx.Prepare(query)
	if err == nil {
		s.stmt[name] = stmt
	}
	return stmt, err
}

type File struct {
	Path  string
	Dev   uint64
	Nlink uint64
	Mtime time.Time
	Size  int64
}

type Symlink struct {
	Path   string
	Dev    uint64
	Nlink  uint64
	Mtime  time.Time
	Target string
}

func sortEntries(list interface{}, order Order) interface{} {
	lv := reflect.ValueOf(list)
	n := lv.Len()
	s := make([]*entry, n)
	for i := 0; i < n; i++ {
		v := lv.Index(i).Elem()
		path := v.FieldByName("Path").Interface().(string)
		vol := filepath.VolumeName(path)
		s[i] = &entry{
			Vol:   strings.Split(vol, `\`),
			Path:  strings.Split(path[len(vol):], string(os.PathSeparator)),
			Nlink: v.FieldByName("Nlink").Interface().(uint64),
			Mtime: v.FieldByName("Mtime").Interface().(time.Time),
			Data:  v.Addr(),
		}
	}
	less := func(a, b []string) bool {
		if len(a) != len(b) {
			return len(a) < len(b)
		}
		for i := range a {
			if a[i] != b[i] {
				return a[i] < b[i]
			}
		}
		return false
	}
	sort.Slice(s, func(i, j int) bool {
		a := s[i]
		b := s[j]
		if !a.Mtime.Equal(b.Mtime) {
			if order == Desc {
				return a.Mtime.After(b.Mtime)
			}
			return a.Mtime.Before(b.Mtime)
		}
		if a.Nlink != b.Nlink {
			return a.Nlink > b.Nlink
		}
		if less(a.Vol, b.Vol) {
			// local disk < shared folder
			return true
		}
		return less(a.Path, b.Path)
	})
	for i, e := range s {
		lv.Index(i).Set(e.Data.(reflect.Value))
	}
	return list
}

type entry struct {
	Vol   []string
	Path  []string
	Nlink uint64
	Mtime time.Time
	Data  interface{}
}

type Order uint

const (
	Asc Order = iota
	Desc
)

var (
	pragma   map[string]string
	table    map[string][]string
	index    map[string][][]string
	triggers []string
)

func init() {
	pragma = map[string]string{
		"auto_vacuum":   "FULL",
		"cache_size":    "-65536",
		"foreign_keys":  "ON",
		"journal_mode":  "WAL",
		"secure_delete": "ON",
		"synchronous":   "NORMAL",
	}

	table = make(map[string][]string)
	table["info"] = []string{
		"id      INTEGER   NOT NULL PRIMARY KEY",
		"path    TEXT      NOT NULL UNIQUE",
		"dev     INTEGER   NOT NULL CHECK (0 < dev)",
		"nlink   INTEGER   NOT NULL CHECK (0 < nlink) DEFAULT 1",
		"mtime   TIMESTAMP NOT NULL",
	}
	table["file"] = []string{
		"id      INTEGER   NOT NULL PRIMARY KEY",
		"info_id INTEGER   NOT NULL REFERENCES info (id) ON DELETE CASCADE UNIQUE",
		"size    INTEGER   NOT NULL CHECK (0 <= size)",
	}
	table["symlink"] = []string{
		"id      INTEGER   NOT NULL PRIMARY KEY",
		"info_id INTEGER   NOT NULL REFERENCES info (id) ON DELETE CASCADE UNIQUE",
		"target  TEXT      NOT NULL",
	}

	table["master"] = []string{
		"id      INTEGER NOT NULL PRIMARY KEY",
		"type    TEXT    NOT NULL UNIQUE",
		"done    INTEGER NOT NULL CHECK (0 <= done)  DEFAULT 0",
		"total   INTEGER NOT NULL CHECK (0 <= total) DEFAULT 0",
	}

	index = make(map[string][][]string)
	index["info"] = [][]string{
		{"dev", "mtime"},
	}
	index["file"] = [][]string{
		{"info_id", "size"},
	}
	index["symlink"] = [][]string{
		{"info_id", "target"},
	}

	for _, a := range [][]interface{}{
		{"file", "symlink"},
		{"symlink", "file"},
	} {
		triggers = append(triggers, fmt.Sprintf(cli.Dedent(`
			CREATE TRIGGER IF NOT EXISTS %v_insert
			  AFTER INSERT ON %[1]v
			  FOR EACH ROW
			  BEGIN
			    DELETE FROM %v
			     WHERE info_id = NEW.info_id;
			    UPDATE master
			       SET total = total - 1
			     WHERE type      = '%[2]v'
			       AND changes() > 0;
			    UPDATE master
			       SET total = total + 1
			     WHERE type = '%[1]v';
			  END
		`), a...))
		triggers = append(triggers, fmt.Sprintf(cli.Dedent(`
			CREATE TRIGGER IF NOT EXISTS %v_delete
			  AFTER DELETE ON %[1]v
			  FOR EACH ROW
			  BEGIN
			    UPDATE master
			       SET done = done + 1
			     WHERE type = '%[1]v';
			  END
		`), a...))
	}
}

func open(name string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", name)
	if err != nil {
		return nil, err
	}

	list := make(sort.StringSlice, len(pragma))
	i := 0
	for k := range pragma {
		list[i] = k
		i++
	}
	list.Sort()
	for _, k := range list {
		v := pragma[k]
		if _, err := db.Exec(fmt.Sprintf("PRAGMA %v = %v", k, v)); err != nil {
			db.Close()
			return nil, fmt.Errorf("PRAGMA %v = '%v': %v", k, v, err)
		}
	}

	for k, v := range table {
		if _, err := db.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %v (\n  %v\n)", k, strings.Join(v, ",\n  "))); err != nil {
			db.Close()
			return nil, fmt.Errorf("CREATE TABLE %v: %v", k, err)
		}
	}

	for k, v := range index {
		for i, v := range v {
			idx := fmt.Sprintf("index_%v_%v", k, i+1)
			if _, err := db.Exec(fmt.Sprintf("CREATE INDEX IF NOT EXISTS %v ON %v (%v)", idx, k, strings.Join(v, ", "))); err != nil {
				db.Close()
				return nil, fmt.Errorf("CREATE INDEX %v: %v", idx, err)
			}
		}
	}

	for _, q := range triggers {
		if _, err := db.Exec(q); err != nil {
			db.Close()
			return nil, fmt.Errorf("CREATE TRIGGER: %v", err)
		}
	}

	tx, err := db.Begin()
	if err != nil {
		db.Close()
		return nil, err
	}
	defer tx.Rollback()

	for _, t := range []string{"file", "symlink"} {
		if _, err := tx.Exec("INSERT OR IGNORE INTO master (type) VALUES (?)", t); err != nil {
			db.Close()
			return nil, err
		}
	}

	return db, tx.Commit()
}
