// Copyright 2018 Hans van Leeuwen. All rights reserved.
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

// Get downloads files
func Get(config Config, urls []string, key [32]byte, iv []byte) error {

	for _, url := range urls {
		r, err := download(url)
		if err != nil {
			return err
		}

		if config.Encrypt {
			r, err = wrapReaderAES256(r, key, iv)
			if err != nil {
				return err
			}
		}

		if config.Compress {
			r, err = wrapReaderGzip(r)
			if err != nil {
				return err
			}
		}

		if config.Tar {
			return unpack(r, config.Dest)
		}

		if config.StdOut {
			_, err = io.Copy(os.Stdout, r)
		} else {
			out := filepath.Join(config.Dest, path.Base(url))
			f, err := os.Create(out)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(f, r)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func download(url string) (io.Reader, error) {

	// Make http request
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("User-Agent", useragent)

	res, err := http.DefaultClient.Do(req)
	if err == nil && (res.StatusCode < 200 || res.StatusCode > 299) {
		return nil, fmt.Errorf("Invalid http status %d %s", res.StatusCode, http.StatusText(res.StatusCode))
	}

	if err != nil {
		return nil, err
	}
	return res.Body, nil
}

func unpack(r io.Reader, destdir string) error {

	tr := tar.NewReader(r)

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
		target := filepath.Join(destdir, header.Name)
		print(target)

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

func wrapReaderGzip(r io.Reader) (io.Reader, error) {
	return gzip.NewReader(r)
}

func wrapReaderAES256(r io.Reader, key [32]byte, iv []byte) (io.Reader, error) {
	// Create reader
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return r, err
	}
	stream := cipher.NewOFB(block, iv)
	return cipher.StreamReader{S: stream, R: r}, nil
}
