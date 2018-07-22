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
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
)

// Put uploads the files in files to https://transfer.sh
func Put(config Config, files []string, output io.Writer, password []byte) error {

	url, err := url.Parse(config.BaseURL)
	if err != nil {
		return err
	}

	if len(files) == 1 && files[0] == "-" {
		if config.Tar {
			return errors.New("tar makes no sense when reading from stdin")
		}

		// Read from stdin
		return copy(os.Stdin, url, config, "stdin", password, output)
	}

	// Create a tar archive before uploading
	if config.Tar {
		r, w := io.Pipe()
		go writeTar(w, config.Compress, config.Encrypt, config.ProgressBar, password, files)
		url.Path = path.Join(url.Path, "tar")
		b, err := upload(r, url.String(), config.MaxDays, config.MaxDownloads)
		if err != nil {
			return err
		}
		fmt.Fprintln(output, string(b))
		return nil
	}

	// Upload all files in files
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			return err
		}

		err = copy(f, url, config, filepath.Base(file), password, output)
		if err != nil {
			return err
		}
	}
	return nil
}

func copy(f io.ReadCloser, url *url.URL, config Config, name string, password []byte, output io.Writer) error {
	r, w := io.Pipe()
	go writeFile(w, config.Compress, config.Encrypt, config.Checksum, password, f, name, 0)
	url.Path = path.Join(url.Path, name)
	b, err := upload(r, url.String(), config.MaxDays, config.MaxDownloads)
	fmt.Fprintln(output, string(b))
	return err
}

func upload(r io.Reader, url string, maxdays, maxdownloads int) ([]byte, error) {

	// Create the request
	req, err := http.NewRequest(http.MethodPut, url, r)
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("User-Agent", useragent)
	if maxdays != 0 {
		req.Header.Set("Max-Days", strconv.Itoa(maxdays))
	}
	if maxdownloads != 0 {
		req.Header.Set("Max-Downloads", strconv.Itoa(maxdownloads))
	}

	// Do request
	res, err := http.DefaultClient.Do(req)
	if err == nil && (res.StatusCode < 200 || res.StatusCode > 299) {
		return nil, fmt.Errorf("Invalid http status %d %s", res.StatusCode, http.StatusText(res.StatusCode))
	}
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// Read body
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func wrapWriter(w io.Writer, compress, encrypt bool, password []byte) (io.Writer, error) {
	var err error

	if encrypt {
		w, err = wrapWriterAES256(w, password)
	}

	if compress {
		w = gzip.NewWriter(w)
	}

	return w, err
}

func writeFile(w io.Writer, compress, encrypt, checksum bool, password []byte, r io.ReadCloser, prefix string, datalength int64) error {
	defer r.Close()

	// Make sure we close the w if it is a io.Closer
	if c, ok := w.(io.Closer); ok {
		defer c.Close()
	}

	var err error
	var h hash.Hash

	if checksum {
		h = sha256.New()
		w = io.MultiWriter(w, h)
	}

	if datalength > 0 {
		r = wrapReaderProgressBar(r, prefix, datalength)
		defer r.Close()
	}

	w, err = wrapWriter(w, compress, encrypt, password)
	if c, ok := w.(io.Closer); ok {
		defer c.Close()
	}
	if err != nil {
		return err
	}

	_, err = io.Copy(w, r)

	// Print checksum
	if checksum {
		fmt.Printf("Checksum: %x\n", h.Sum(nil))
	}

	return err
}

func writeTar(w io.Writer, compress, encrypt, progressbar bool, password []byte, filenames []string) error {

	var err error

	if c, ok := w.(io.Closer); ok {
		defer c.Close()
	}

	w, err = wrapWriter(w, compress, encrypt, password)
	if c, ok := w.(io.Closer); ok {
		defer c.Close()
	}

	// Create tar archive
	tw := tar.NewWriter(w)
	defer tw.Close()

	for _, f := range filenames {
		err = add(tw, f, progressbar)
		if err != nil {
			return err
		}
	}

	return nil
}

func add(tw *tar.Writer, src string, progressbar bool) error {
	// walk path
	return filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {

		var r io.ReadCloser

		// create a new dir/file header
		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

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
		r, err = os.Open(file)
		defer r.Close()
		if err != nil {
			return err
		}

		if progressbar {
			r = wrapReaderProgressBar(r, fi.Name(), fi.Size())
			defer r.Close()
		}

		// copy file data into tar writer
		_, err = io.Copy(tw, r)

		return err
	})
}

func wrapWriterAES256(w io.Writer, password []byte) (io.WriteCloser, error) {

	header := []byte("Salted__")
	w.Write(header)

	// Create random salt
	salt := make([]byte, 8)
	io.ReadFull(rand.Reader, salt)
	w.Write(salt[:])

	// Create key by hashing the password
	key, iv := passwordToKey(password, salt)

	// Create writer
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	stream := cipher.NewOFB(block, iv)
	return cipher.StreamWriter{S: stream, W: w}, nil
}

type hashWriter struct {
	h hash.Hash
	w io.Writer
}

// Close closes the underlying Writer and returns its Close return value, if the Writer
// is also an io.Closer. Otherwise it returns nil.
func (h hashWriter) Close() error {
	checksum := h.h.Sum(nil)

	fmt.Printf("Checksum: %x\n", checksum)

	if c, ok := h.w.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func (h hashWriter) Write(b []byte) (int, error) {
	// Write to hash
	n, err := h.h.Write(b)
	if err != nil {
		return n, err
	}

	// Write to writer
	return h.w.Write(b)
}

func wrapWriterSHA256(w io.Writer) io.WriteCloser {

	return hashWriter{h: sha256.New(), w: w}
}
