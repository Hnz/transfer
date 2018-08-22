// Copyright 2018 Hans van Leeuwen. All rights reserved.
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"crypto/rand"
	"io"
	"io/ioutil"
	"testing"
	"time"
)

func TestProgressBarReader(t *testing.T) {

	var datalength int64
	datalength = 50000
	x := make([]byte, datalength)
	b := bytes.NewBuffer(x)
	r := wrapReaderProgressBar(b, "Prefix", datalength)
	io.Copy(ioutil.Discard, r)
}

func TestProgressBarWriter(t *testing.T) {

	datalength := 10000
	var b bytes.Buffer
	w := wrapWriterProgressBar(&b, "Prefix", int64(datalength))

	iterations := 10
	for i := 0; i < iterations; i++ {
		time.Sleep(1000 * time.Millisecond)
		x := make([]byte, datalength/iterations)
		io.ReadFull(rand.Reader, x)
		w.Write(x)
	}
}
