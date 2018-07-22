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

func (p *progressBar) Draw() {
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
	if p.Counter > p.Total {
		p.Counter = p.Total
	}

	p.Draw()
	return p.r.Read(b)
}

func (p *progressBarWriteCloser) Write(b []byte) (int, error) {
	p.Counter += int64(len(b))
	p.Draw()
	return p.w.Write(b)
}

/*
func wrapWriterProgressBar(w io.WriteCloser, prefix string, datalength int64) io.WriteCloser {
	tmpl := defaultTemplate()
	return &progressBarWriteCloser{progressBar{0, datalength, prefix, os.Stdout, tmpl}, w}
}
*/
func wrapReaderProgressBar(r io.ReadCloser, prefix string, datalength int64) io.ReadCloser {
	tmpl := defaultTemplate()
	return &progressBarReadCloser{progressBar{0, datalength, prefix, os.Stdout, tmpl}, r}
}

func defaultTemplate() *template.Template {

	txt := "\r{{.Prefix}} {{.Counter}} / {{.Total}}  {{percentage .Counter .Total}}"

	fm := template.FuncMap{
		"bar": func(a, b int) int {
			return a / b
		},
		"percentage": func(a, b int64) string {
			return fmt.Sprintf("%6.2f%%", float64(a)/float64(b)*100)
		}}

	return template.Must(template.New("defaulttemplate").Funcs(fm).Parse(txt))
}
