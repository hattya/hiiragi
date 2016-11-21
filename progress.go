//
// hiiragi :: progress.go
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
	"sync"

	"github.com/hattya/go.cli"
)

type counter struct {
	N int64

	ui    *cli.CLI
	label string

	mu  sync.Mutex
	pos int64
	bol bool
}

func newCounter(ui *cli.CLI, label string) *counter {
	return &counter{
		ui:    ui,
		label: label,
	}
}

func (c *counter) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ui.Printf("\x1b[?25h")
	if !c.bol {
		c.ui.Printf("\n")
		c.bol = true
	}
}

func (c *counter) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ui.Printf("\x1b[1K\r")
	c.bol = true
}

func (c *counter) Set(i int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.pos = i
}

func (c *counter) Update(i int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.pos += i
	c.ui.Printf("\r\x1b[?25l")
	if 0 < c.N {
		c.ui.Printf("%s: %v / %v", c.label, c.pos, c.N)
	} else {
		c.ui.Printf("%s: %v", c.label, c.pos)
	}
	c.bol = false
}
