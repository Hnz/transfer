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

func Put(w io.WriteCloser, conf Config, files []string) {

	var header Header
	defer w.Close()

	if conf.Encrypt {
		// Update header
		header |= AES256

		// Make random IV and write it to the output buffer
		iv := make([]byte, aes.BlockSize)
		io.ReadFull(rand.Reader, iv)
		fmt.Println(iv)
		w.Write(iv)

		// Ask for password and hash it to create the key
		key := getKey()

		// Create writer
		block, err := aes.NewCipher(key[:])
		handleError(err)
		stream := cipher.NewOFB(block, iv[:])
		w = cipher.StreamWriter{S: stream, W: w}
		defer w.Close()
	}

	if conf.Compress {
		// Update header
		header |= GZIP

		w = gzip.NewWriter(w)
		defer w.Close()
	}

	// Write header
	header |= TAR
	fmt.Println("Header", header, header.HasFlag(TAR))
	binary.Write(w, binary.LittleEndian, header)

	tw := tar.NewWriter(w)
	defer tw.Close()

	for _, f := range files {
		err := add(tw, f)
		handleError(err)
	}
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
