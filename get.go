// Copyright 2018 Hans van Leeuwen. All rights reserved.
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func Get(r io.Reader, conf Config) {

	// Read header
	var header Header
	binary.Read(r, binary.LittleEndian, &header)

	fmt.Println("HEADER", header)

	if conf.Encrypt {
		// First read the IV from the stream
		iv := make([]byte, aes.BlockSize)
		io.ReadFull(r, iv)
		//r.Read(iv)
		fmt.Println(iv)
		key := getKey()

		// Create reader
		block, err := aes.NewCipher(key[:])
		handleError(err)
		stream := cipher.NewOFB(block, iv[:])
		r = cipher.StreamReader{S: stream, R: r}
	}

	if conf.Compress {
		r, err := gzip.NewReader(r)
		handleError(err)
		defer r.Close()
	}

	tr := tar.NewReader(r)

	err := Unpack(tr, conf.DestDir)
	handleError(err)
}

func Unpack(tr *tar.Reader, dest string) error {

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		case header == nil:
			return errors.New("Unable to read header")
		}

		// the target location where the dir/file should be created
		target := filepath.Join(dest, header.Name)
		fmt.Println("<", target)
		// the following switch could also be done using fi.Mode(), not sure if there
		// a benefit of using one vs. the other.
		// fi := header.FileInfo()

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			defer f.Close()

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}
		}
	}
}
