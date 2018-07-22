// Copyright 2018 Hans van Leeuwen. All rights reserved.
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

// Get downloads files
func Get(config Config, urls []string, password []byte) error {

	for _, url := range urls {
		var h hash.Hash
		var r io.Reader
		var w io.Writer
		var err error

		r, err = download(url, config.ProgressBar)
		if err != nil {
			return err
		}

		if config.Encrypt {
			r, err = wrapReaderAES256(r, password)
			if err != nil {
				return err
			}
		}

		if config.Compress {
			r, err = gzip.NewReader(r)
			if err != nil {
				return err
			}
		}

		if config.Tar {
			return unpack(r, config.Dest)
		}

		if config.StdOut {
			w = os.Stdout
		} else {
			out := filepath.Join(config.Dest, path.Base(url))
			w, err = os.Create(out)
			if err != nil {
				return err
			}
		}

		// Create hash
		if config.Checksum {
			h = sha256.New()
			w = io.MultiWriter(w, h)
		}

		_, err = io.Copy(w, r)
		if err != nil {
			return err
		}

		if c, ok := r.(io.Closer); ok {
			err = c.Close()
			if err != nil {
				return err
			}
		}

		if c, ok := w.(io.Closer); ok {
			err = c.Close()
			if err != nil {
				return err
			}
		}

		// Create hash
		if config.Checksum {
			fmt.Printf("Checksum: %x\n", h.Sum(nil))
		}
	}

	return nil
}

func download(url string, progressbar bool) (io.ReadCloser, error) {

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

	if progressbar {
		prefix := path.Base(url)
		return wrapReaderProgressBar(res.Body, prefix, res.ContentLength), nil
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

func wrapReaderAES256(r io.Reader, password []byte) (io.Reader, error) {

	// First read the salt from the stream
	var header [16]byte
	_, err := io.ReadFull(r, header[:])
	if err != nil {
		return r, err
	}

	// See http://justsolve.archiveteam.org/wiki/OpenSSL_salted_format
	if string(header[:8]) != "Salted__" {
		return r, errors.New("Stream does not start with 'Salted__'")
	}

	// Create key by hashing the password
	key, iv := passwordToKey(password, header[8:])

	// Create reader
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return r, err
	}
	stream := cipher.NewOFB(block, iv)
	return cipher.StreamReader{S: stream, R: r}, nil
}
