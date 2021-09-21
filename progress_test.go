//
// hiiragi :: progress_test.go
//
//   Copyright (c) 2016-2021 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
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
