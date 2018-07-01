// Copyright 2018 Hans van Leeuwen. All rights reserved.
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func Put(w io.WriteCloser, conf Config, files []string) error {

	var header Header
	o := w
	defer w.Close()

	if conf.Encrypt {
		header.AddFlag(AES256)
	}

	if conf.Compress {
		header.AddFlag(GZIP)
	}

	// Write header
	header.AddFlag(TAR)
	binary.Write(o, binary.LittleEndian, header)

	if header.HasFlag(AES256) {
		w = WrapWriterAes256(w, conf.Key)
		defer w.Close()
	}

	if header.HasFlag(GZIP) {
		w = WrapWriterGzip(w)
		defer w.Close()
	}

	tw := tar.NewWriter(w)
	defer tw.Close()

	for _, f := range files {
		err := add(tw, f)
		if err != nil {
			return err
		}
	}

	return nil
}

// Tar takes a source and variable writers and walks 'source' writing each file
// found to the tar writer
func Tar(files []string, writer io.Writer) error {

	tw := tar.NewWriter(writer)
	defer tw.Close()

	for _, f := range files {
		err := add(tw, f)
		if err != nil {
			return err
		}
	}

	return nil
}

func add(tw *tar.Writer, src string) error {
	// walk path
	return filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {

		handleError(err)

		// create a new dir/file header
		header, err := tar.FileInfoHeader(fi, fi.Name())
		handleError(err)
		fmt.Println("<", header.Name)

		// write the header
		err = tw.WriteHeader(header)
		if err != nil {
			return err
		}

		// return on non-regular files (thanks to [kumo](https://medium.com/@komuw/just-like-you-did-fbdd7df829d3) for this suggested update)
		if !fi.Mode().IsRegular() {
			return nil
		}

		// open files for taring
		f, err := os.Open(file)
		defer f.Close()
		if err != nil {
			return err
		}

		// copy file data into tar writer
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}

		return nil
	})
}

func WrapWriterGzip(w io.WriteCloser) io.WriteCloser {
	return gzip.NewWriter(w)
}

func WrapWriterAes256(w io.WriteCloser, key [32]byte) io.WriteCloser {

	// Make random IV and write it to the output buffer
	iv := make([]byte, aes.BlockSize)
	io.ReadFull(rand.Reader, iv)
	w.Write(iv)

	// Create writer
	block, err := aes.NewCipher(key[:])
	handleError(err)
	stream := cipher.NewOFB(block, iv[:])
	return cipher.StreamWriter{S: stream, W: w}
}
