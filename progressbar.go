// Copyright 2018 Hans van Leeuwen. All rights reserved.
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"io"

	pb "gopkg.in/cheggaaa/pb.v1"
)

type progressBar struct {
	pb *pb.ProgressBar
	w  io.WriteCloser
}

func (p *progressBar) Write(b []byte) (n int, err error) {
	p.pb.Write(b)
	return p.w.Write(b)
}

func (p *progressBar) Close() error {
	p.pb.Finish()
	return nil
}

func wrapWriterProgressBar(w io.WriteCloser, prefix string, datalength int64) io.WriteCloser {

	// create and start bar
	bar := pb.New64(datalength).SetUnits(pb.U_BYTES).Prefix(prefix)
	bar.Start()

	return &progressBar{pb: bar, w: w}
}

func wrapReaderProgressBar(r io.Reader, prefix string, datalength int64) *pb.Reader {
	// create and start bar
	bar := pb.New(int(datalength)).SetUnits(pb.U_BYTES).Prefix(prefix)
	bar.Start()

	// return the proxy reader
	return bar.NewProxyReader(r)
}
