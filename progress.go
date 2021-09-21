//
// hiiragi :: progress.go
//
//   Copyright (c) 2016-2021 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package hiiragi

import (
	"sync"

	"github.com/hattya/go.cli"
)

type counter struct {
	N    int64
	Show bool

	ui    *cli.CLI
	label string

	mu  sync.Mutex
	pos int64
	bol bool
}

func newCounter(ui *cli.CLI, label string) *counter {
	return &counter{
		Show:  true,
		ui:    ui,
		label: label,
		bol:   true,
	}
}

func (c *counter) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Show {
		c.ui.Printf("\x1b[?25h")
	} else {
		c.render()
	}
	if !c.bol {
		c.ui.Printf("\n")
		c.bol = true
	}
}

func (c *counter) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Show {
		c.ui.Printf("\x1b[1K\r")
	}
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
	if c.Show {
		c.ui.Printf("\r\x1b[?25l")
		c.render()
	}
}

func (c *counter) render() {
	if c.N > 0 {
		c.ui.Printf("%s: %v / %v", c.label, c.pos, c.N)
	} else {
		c.ui.Printf("%s: %v", c.label, c.pos)
	}
	c.bol = false
}
