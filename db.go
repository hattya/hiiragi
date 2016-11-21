//
// hiiragi :: db.go
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
	db *sql.DB
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
	return &DB{db}, nil
}

func (db *DB) Close() error {
	return db.db.Close()
}

func (db *DB) NFile() (int64, int64, error) {
	return db.count(`file`)
}

func (db *DB) count(t string) (done, n int64, err error) {
	q := fmt.Sprintf(cli.Dedent(`
		SELECT count(done),
		       count(*)
		  FROM %v
	`), t)
	err = db.db.QueryRow(q).Scan(&done, &n)
	return
}

func (db *DB) NextFiles() ([]*File, error) {
	list, err := db.next(&File{})
	return list.([]*File), err
}

func (db *DB) next(t interface{}) (list interface{}, err error) {
	tt := reflect.Indirect(reflect.ValueOf(t)).Type()
	col := tt.Field(tt.NumField() - 1).Name
	lv := reflect.MakeSlice(reflect.SliceOf(reflect.PtrTo(tt)), 0, 0)
	list = lv.Interface() // make type assertion simple

	v := reflect.New(tt).Elem()
	ak := []reflect.Value{
		v.FieldByName("Dev"),
		v.FieldByName(col),
		v.FieldByName("Mtime"),
	}
	q := fmt.Sprintf(cli.Dedent(`
		SELECT dev,
		       %v,
		       i.mtime
		  FROM %v
		 INNER JOIN info AS i
		         ON info_id = i.id
		 WHERE done IS NULL
	`), strings.ToLower(col), strings.ToLower(tt.Name()))
	if err = db.db.QueryRow(q).Scan(ak[0].Addr().Interface(), ak[1].Addr().Interface(), ak[2].Addr().Interface()); err != nil {
		if err == sql.ErrNoRows {
			err = nil
		}
		return
	}

	a := []interface{}{
		ak[0].Interface(),
		ak[1].Interface(),
		ak[2].Interface(),
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, cli.Dedent(`
		SELECT i.path,
		       i.dev,
		       i.nlink,
		       i.mtime,
		       %v
		  FROM %v
		 INNER JOIN info AS i
		         ON info_id = i.id
		 WHERE dev     =  ?
		   AND %[1]v   =  ?
		   AND i.mtime =  ?
		   AND done    IS NULL
	`), strings.ToLower(col), strings.ToLower(tt.Name()))
	rows, err := db.db.Query(b.String(), a...)
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
	sortEntries(list)
	return
}

func (db *DB) Done(path string) (err error) {
	tx, err := db.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	for _, t := range []string{"file"} {
		q := fmt.Sprintf(cli.Dedent(`
			UPDATE %v
			   SET done = datetime('now')
			 WHERE EXISTS (
			         SELECT *
			           FROM info AS i
			          WHERE i.id   = info_id
			            AND i.path = ?
			       )
		`), t)
		if _, err = tx.Exec(q, path); err != nil {
			return
		}
	}

	return tx.Commit()
}

func (db *DB) SetHash(path, hash string) (err error) {
	tx, err := db.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	q := cli.Dedent(`
		UPDATE file
		   SET hash = ?,
		       done = datetime('now')
		 WHERE EXISTS (
		         SELECT *
		           FROM info AS i
		          WHERE i.id   = info_id
		            AND i.path = ?
		       )
	`)
	if _, err = tx.Exec(q, hash, path); err != nil {
		return
	}

	return tx.Commit()
}

func (db *DB) Update(fi FileInfoEx) (err error) {
	dev, err := fi.Dev()
	if err != nil {
		return
	}
	nlink, err := fi.Nlink()
	if err != nil {
		return
	}

	tx, err := db.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	i := cli.Dedent(`
		INSERT INTO info (dev, nlink, mtime, path)
		  VALUES (?, ?, ?, ?)
	`)
	u := cli.Dedent(`
		UPDATE info
		   SET dev   = ?,
		       nlink = ?,
		       mtime = ?
		 WHERE path = ?
	`)
	if err = db.upsert(tx, i, u, dev, nlink, fi.ModTime(), fi.Path()); err != nil {
		return
	}

	a := make([]interface{}, 2)
	a[1] = fi.Path()
	if fi.Mode()&os.ModeType == 0 {
		// file
		i = cli.Dedent(`
			INSERT INTO file (info_id, size)
			  SELECT id, ?
			    FROM info
			   WHERE path = ?
		`)
		u = cli.Dedent(`
			UPDATE file
			   SET size = ?
			 WHERE EXISTS (
			         SELECT *
			           FROM info AS i
			          WHERE i.id   = info_id
			            AND i.path = ?
			       )
		`)
		a[0] = fi.Size()
	}
	if err = db.upsert(tx, i, u, a...); err != nil {
		return
	}

	return tx.Commit()
}

func (db *DB) upsert(tx *sql.Tx, insert, update string, a ...interface{}) (err error) {
	res, err := tx.Exec(update, a...)
	if err != nil {
		return
	}
	n, err := res.RowsAffected()
	if err == nil && n < 1 {
		res, err = tx.Exec(insert, a...)
	}
	return
}

type File struct {
	Path  string
	Dev   uint64
	Nlink uint64
	Mtime time.Time
	Size  int64
}

func sortEntries(list interface{}) interface{} {
	lv := reflect.ValueOf(list)
	n := lv.Len()
	s := make(entrySlice, n)
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
	sort.Sort(s)
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

type entrySlice []*entry

func (p entrySlice) Len() int      { return len(p) }
func (p entrySlice) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func (p entrySlice) Less(i, j int) bool {
	a := p[i]
	b := p[j]
	if !a.Mtime.Equal(b.Mtime) {
		return a.Mtime.Before(b.Mtime)
	}
	if a.Nlink != b.Nlink {
		return a.Nlink > b.Nlink
	}
	if p.less(a.Vol, b.Vol) {
		// local disk < shared folder
		return true
	}
	return p.less(a.Path, b.Path)
}

func (p entrySlice) less(a, b []string) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	} else {
		for i := range a {
			if a[i] != b[i] {
				return a[i] < b[i]
			}
		}
		return false
	}
}

var (
	pragma   map[string]string
	table    map[string][]string
	triggers []string
)

func init() {
	pragma = map[string]string{
		"auto_vacuum":   "FULL",
		"foreign_keys":  "ON",
		"journal_mode":  "WAL",
		"secure_delete": "ON",
	}

	table = make(map[string][]string)
	table["info"] = []string{
		"id         INTEGER   NOT NULL PRIMARY KEY",
		"path       TEXT      NOT NULL UNIQUE",
		"dev        INTEGER   NOT NULL",
		"nlink      INTEGER   NOT NULL DEFAULT 1 CHECK(0 < nlink)",
		"mtime      TIMESTAMP NOT NULL",
		"updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP",
	}
	table["file"] = []string{
		"id         INTEGER   NOT NULL PRIMARY KEY",
		"info_id    INTEGER   NOT NULL REFERENCES info (id) ON DELETE CASCADE UNIQUE",
		"size       INTEGER   NOT NULL",
		"hash       TEXT",
		"done       TIMESTAMP",
	}

	// info
	triggers = append(triggers, cli.Dedent(`
		CREATE TRIGGER IF NOT EXISTS info_update
		  AFTER UPDATE OF path, dev, nlink, mtime ON info
		  FOR EACH ROW
		  BEGIN
		    UPDATE info
		       SET updated_at = datetime('now')
		     WHERE id = NEW.id;
		    UPDATE file
		       SET hash = NULL,
		           done = NULL
		     WHERE EXISTS (
		             SELECT *
		               FROM info AS i
		              INNER JOIN file AS f
		                      ON i.id = f.info_id
		              WHERE i.id = NEW.id
		           );
		  END
	`))
	// file
	triggers = append(triggers, cli.Dedent(`
		CREATE TRIGGER IF NOT EXISTS file_update
		  AFTER UPDATE OF info_id, size ON file
		  FOR EACH ROW
		  BEGIN
		    UPDATE file
		       SET hash = NULL,
		           done = NULL
		     WHERE id = NEW.id
		       AND EXISTS (
		             SELECT *
		               FROM info AS i
		              INNER JOIN file AS f
		                      ON i.id = f.info_id
		              WHERE i.id         = NEW.info_id
		                AND i.updated_at > NEW.done
		           );
		  END
	`))
}

func open(name string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", name)
	if err != nil {
		return nil, err
	}

	for k, v := range pragma {
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

	for _, q := range triggers {
		if _, err := db.Exec(q); err != nil {
			db.Close()
			return nil, fmt.Errorf("CREATE TRIGGER: %v", err)
		}
	}

	return db, err
}
