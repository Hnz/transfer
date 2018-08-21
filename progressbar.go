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
	Output   io.Writer
	Template *template.Template
}

type progressBarReader struct {
	progressBar
	r io.Reader
}

type progressBarWriter struct {
	progressBar
	w io.Writer
}

func (p progressBar) Draw() {
	p.Template.Execute(p.Output, p)
	//fmt.Fprintf(o, "\r%d/%d", c, l)
}

func (p progressBar) Finish() {
	fmt.Fprint(p.Output, "\n")
}

// Close closes the underlying Reader and returns its Close return value, if the Writer
// is also an io.Closer. Otherwise it returns nil.
func (p *progressBarReader) Close() error {
	p.Finish()
	if c, ok := p.r.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

// Close closes the underlying Writer and returns its Close return value, if the Writer
// is also an io.Closer. Otherwise it returns nil.
func (p *progressBarWriter) Close() error {
	p.Finish()
	if c, ok := p.w.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func (p *progressBarReader) Read(b []byte) (int, error) {
	p.Counter += int64(len(b))
	if p.Counter > p.Total {
		p.Counter = p.Total
	}
	p.Draw()
	return p.r.Read(b)
}

func (p *progressBarWriter) Write(b []byte) (int, error) {
	p.Counter += int64(len(b))
	if p.Counter > p.Total {
		p.Counter = p.Total
	}
	p.Draw()
	return p.w.Write(b)
}

func wrapWriterProgressBar(w io.Writer, prefix string, datalength int64) *progressBarWriter {
	tmpl := defaultTemplate()
	return &progressBarWriter{progressBar{0, datalength, prefix, os.Stdout, tmpl}, w}
}

func wrapReaderProgressBar(r io.Reader, prefix string, datalength int64) *progressBarReader {
	tmpl := defaultTemplate()
	return &progressBarReader{progressBar{0, datalength, prefix, os.Stdout, tmpl}, r}
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
