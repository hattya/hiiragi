//
// hiiragi :: progress_test.go
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

package hiiragi_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hattya/go.cli"
	"github.com/hattya/hiiragi"
)

func TestCounter(t *testing.T) {
	var b bytes.Buffer
	ui := cli.NewCLI()
	ui.Stdout = &b
	ui.Stderr = &b

	b.Reset()
	c := hiiragi.NewCounter(ui, "test")
	c.N = 8
	for i := int64(0); i < c.N; i++ {
		c.Update(1)
	}
	c.Close()
	if !strings.Contains(b.String(), "\x1b") {
		t.Errorf("expected ANSI escape sequence to be found")
	}

	b.Reset()
	c = hiiragi.NewCounter(ui, "test")
	c.N = 8
	c.Show = false
	for i := int64(0); i < c.N; i++ {
		c.Update(1)
	}
	c.Close()
	if g, e := b.String(), "test: 8 / 8\n"; g != e {
		t.Errorf("expected %q, got %q", e, g)
	}

	b.Reset()
	c = hiiragi.NewCounter(ui, "test")
	c.Show = false
	for i := 0; i < 8; i++ {
		c.Update(1)
	}
	c.Close()
	if g, e := b.String(), "test: 8\n"; g != e {
		t.Errorf("expected %q, got %q", e, g)
	}
}
