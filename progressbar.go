// Copyright 2018 Hans van Leeuwen. All rights reserved.
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"os"
	"text/template"
)

type progressBar struct {
	Counter  int64
	Total    int64
	Prefix   string
	Output   io.WriteCloser
	Template *template.Template
}

type progressBarReadCloser struct {
	progressBar
	r io.ReadCloser
}

type progressBarWriteCloser struct {
	progressBar
	w io.WriteCloser
}

func draw(p progressBar) {
	p.Template.Execute(p.Output, p)
	//fmt.Fprintf(o, "\r%d/%d", c, l)
}

func (p *progressBar) Finish() {
	fmt.Fprint(p.Output, "\n")
}

func (p *progressBarReadCloser) Close() error {
	p.Finish()
	return p.r.Close()
}

func (p *progressBarWriteCloser) Close() error {
	p.Finish()
	return p.w.Close()
}

func (p *progressBarReadCloser) Read(b []byte) (int, error) {
	p.Counter += int64(len(b))
	draw(progressBar{p.Counter, p.Total, p.Prefix, p.Output, p.Template})
	return p.r.Read(b)
}

func (p *progressBarWriteCloser) Write(b []byte) (int, error) {
	p.Counter += int64(len(b))
	draw(progressBar{p.Counter, p.Total, p.Prefix, p.Output, p.Template})
	return p.w.Write(b)
}

func wrapWriterProgressBar(w io.WriteCloser, prefix string, datalength int64) *progressBarWriteCloser {
	tmpl, err := defaultTemplate()
	if err != nil {
		panic(err)
	}
	return &progressBarWriteCloser{progressBar{0, datalength, prefix, os.Stdout, tmpl}, w}
}

func wrapReaderProgressBar(r io.ReadCloser, prefix string, datalength int64) *progressBarReadCloser {
	tmpl, err := defaultTemplate()
	if err != nil {
		panic(err)
	}
	return &progressBarReadCloser{progressBar{0, datalength, prefix, os.Stdout, tmpl}, r}
}

func defaultTemplate() (*template.Template, error) {

	txt := "\r{{.Prefix}} {{.Counter}} / {{.Total}}  {{percentage .Counter .Total}}"

	fm := template.FuncMap{
		"divide": func(a, b int) int {
			return a / b
		},
		"percentage": func(a, b int) string {
			fmt.Println("Foo", a/b*100)
			return string(a / b * 100)
		}}
	return template.New("writer").Funcs(fm).Parse(txt)
}
