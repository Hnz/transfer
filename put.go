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
	"io"
	"os"
	"path/filepath"
)

// If files is empty, stdin will be used
func Put(w io.WriteCloser, conf Config, keyFunc KeyFunc, files []string) error {

	defer w.Close()

	// Create header
	var header Header

	if len(files) > 1 {
		header.AddFlag(TAR)
	}

	if conf.Encrypt {
		header.AddFlag(AES256)
	}

	if conf.Compress {
		header.AddFlag(GZIP)
	}

	// Write header
	binary.Write(w, binary.LittleEndian, header)

	if header.HasFlag(AES256) {
		w = wrapWriterAES256(w, keyFunc())
		defer w.Close()
	}

	if header.HasFlag(GZIP) {
		w = wrapWriterGzip(w)
		defer w.Close()
	}

	if len(files) > 1 {
		// Create tar archive if more than files are given
		tw := tar.NewWriter(w)
		defer tw.Close()

		for _, f := range files {
			err := add(tw, f)
			if err != nil {
				return err
			}
		}
	} else if len(files) == 1 {
		// Else read from single file
		f, err := os.Open(files[0])
		if err != nil {
			return err
		}

		_, err = io.Copy(w, f)
		return err
	} else {
		// Else read from stdin
		_, err := io.Copy(w, os.Stdin)
		return err
	}

	return nil
}

func add(tw *tar.Writer, src string) error {
	// walk path
	return filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {

		handleError(err)

		info(fi.Name())

		// create a new dir/file header
		header, err := tar.FileInfoHeader(fi, fi.Name())
		handleError(err)

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
		_, err = io.Copy(tw, f)

		return err
	})
}

func wrapWriterGzip(w io.WriteCloser) io.WriteCloser {
	return gzip.NewWriter(w)
}

func wrapWriterAES256(w io.WriteCloser, key []byte) io.WriteCloser {

	// Make random IV and write it to the output buffer
	iv := make([]byte, aes.BlockSize)
	io.ReadFull(rand.Reader, iv)
	w.Write(iv)

	// Create writer
	block, err := aes.NewCipher(key)
	handleError(err)
	stream := cipher.NewOFB(block, iv[:])
	return cipher.StreamWriter{S: stream, W: w}
}
