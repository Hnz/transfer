// Copyright 2018 Hans van Leeuwen. All rights reserved.
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh/terminal"
)

type progressBar struct {
	BarEmpty string    // Character to print for the empty part of the progress bar.
	BarFull  string    // Character to print for the full part of the progress bar.
	Counter  int64     // Tracks the progress made. Should not be greater than Total.
	Total    int64     // Total number of ticks.
	Prefix   string    // Text to put before the progress bar.
	Output   io.Writer // Write output here. Should probably be os.Stdout.
}

type progressBarReader struct {
	progressBar
	r io.Reader
}

type progressBarWriter struct {
	progressBar
	w io.Writer
}

// Draw outputs
func (p progressBar) Draw() {

	// Get terminal width
	fd := int(os.Stdout.Fd())
	width, _, err := terminal.GetSize(fd)
	if err != nil {
		// There seems to be no terminal. Don't draw anything.
		return
	}

	var percentage float64
	if p.Counter == 0 {
		percentage = 0
	} else {
		percentage = float64(p.Counter) / float64(p.Total)
	}

	prefixLength := len(p.Prefix)
	totalLength := len(string(p.Total))
	barLength := width - prefixLength - totalLength*2 - 10
	barFullCount := int(float64(barLength) * percentage)
	barEmptyCount := barLength - barFullCount
	barFullString := strings.Repeat(p.BarFull, barFullCount)
	barEmptyString := strings.Repeat(p.BarEmpty, barEmptyCount)

	//fmt.Println(percentage, prefixLength, totalLength, barLength, barFullCount)

	txt := "\r" +
		p.Prefix +
		" [" +
		barFullString +
		barEmptyString +
		"] " +
		strconv.FormatInt(p.Counter, 10) +
		"/" +
		strconv.FormatInt(p.Total, 10)

	fmt.Fprint(p.Output, txt)
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
	return &progressBarWriter{progressBar{" ", "=", 0, datalength, prefix, os.Stdout}, w}
}

func wrapReaderProgressBar(r io.Reader, prefix string, datalength int64) *progressBarReader {
	return &progressBarReader{progressBar{" ", "=", 0, datalength, prefix, os.Stdout}, r}
}
