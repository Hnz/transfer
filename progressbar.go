// Copyright 2018 Hans van Leeuwen. All rights reserved.
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"os"
)

type progressBarReadCloser struct {
	c      int64
	l      int64
	prefix string
	o      io.WriteCloser
	r      io.ReadCloser
}

type progressBarWriteCloser struct {
	c      int64
	l      int64
	prefix string
	o      io.WriteCloser
	w      io.WriteCloser
}

func draw(c, l int64, o io.WriteCloser) {
	//t := template.New("progress")
	//t.Parse("{{.Prefix}}")
	//t.Execute(o, )
	fmt.Fprintf(o, "\r%d/%d", c, l)
}

func (p *progressBarReadCloser) Add(i int) {
	p.c += int64(i)
	draw(p.c, p.l, p.o)
}

func (p *progressBarWriteCloser) Add(i int) {
	p.c += int64(i)
	draw(p.c, p.l, p.o)
}

func (p *progressBarReadCloser) Close() error {
	p.Finish()
	return p.r.Close()
}

func (p *progressBarWriteCloser) Close() error {
	p.Finish()
	return p.w.Close()
}

func (p *progressBarReadCloser) Finish() {
	fmt.Fprintf(p.o, "\r%d/%d Finised\n", p.l, p.l)
}

func (p *progressBarWriteCloser) Finish() {
	fmt.Fprintf(p.o, "\r%d/%d Finised\n", p.l, p.l)
}

func (p *progressBarReadCloser) Read(b []byte) (int, error) {
	p.Add(len(b))
	return p.r.Read(b)
}

func (p *progressBarWriteCloser) Write(b []byte) (int, error) {
	p.Add(len(b))
	return p.w.Write(b)
}

func wrapWriterProgressBar(w io.WriteCloser, prefix string, datalength int64) *progressBarWriteCloser {

	return &progressBarWriteCloser{c: 0, l: datalength, prefix: prefix, o: os.Stdout, w: w}
}

func wrapReaderProgressBar(r io.ReadCloser, prefix string, datalength int64) *progressBarReadCloser {

	return &progressBarReadCloser{c: 0, l: datalength, prefix: prefix, o: os.Stdout, r: r}
}
