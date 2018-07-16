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
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
)

// Put uploads the files in files to https://transfer.sh
func Put(config Config, files []string, output io.Writer, key [32]byte, iv []byte) error {

	var b []byte
	url, err := url.Parse(config.BaseURL)
	if err != nil {
		return err
	}

	if len(files) == 1 && files[0] == "-" {
		if config.Tar {
			return errors.New("tar makes no sense when reading from stdin")
		}

		// Read from stdin
		r, w := io.Pipe()
		go writeFile(w, config.Compress, config.Encrypt, key, iv, os.Stdin)
		url.Path = path.Join(url.Path, "stdin")
		b, err := upload(r, url.String(), config.MaxDays, config.MaxDownloads)
		if err != nil {
			return err
		}
		fmt.Fprintln(output, string(b))
		return nil
	}

	// Create a tar archive before uploading
	if config.Tar {
		r, w := io.Pipe()
		go writeTar(w, config.Compress, config.Encrypt, key, iv, files)
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

		r, w := io.Pipe()
		go writeFile(w, config.Compress, config.Encrypt, key, iv, f)
		url.Path = path.Join(url.Path, filepath.Base(file))
		b, err = upload(r, url.String(), config.MaxDays, config.MaxDownloads)
		if err != nil {
			return err
		}
		fmt.Fprintln(output, string(b))
	}
	return nil
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
		err = fmt.Errorf("Invalid http status %d %s", res.StatusCode, http.StatusText(res.StatusCode))
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

func writeFile(w io.WriteCloser, compress, encrypt bool, key [32]byte, iv []byte, r io.Reader) error {
	defer w.Close()

	var err error

	if encrypt {
		w, err = wrapWriterAES256(w, key, iv)
		if err != nil {
			return err
		}
		defer w.Close()
	}

	if compress {
		w = wrapWriterGzip(w)
		defer w.Close()
	}

	_, err = io.Copy(w, r)
	return err
}

func writeTar(w io.WriteCloser, compress, encrypt bool, key [32]byte, iv []byte, filenames []string) error {
	defer w.Close()

	var err error

	if encrypt {
		w, err = wrapWriterAES256(w, key, iv)
		if err != nil {
			return err
		}
		defer w.Close()
	}

	if compress {
		w = wrapWriterGzip(w)
		defer w.Close()
	}

	// Create tar archive
	tw := tar.NewWriter(w)
	defer tw.Close()

	for _, f := range filenames {
		err = add(tw, f)
		if err != nil {
			return err
		}
	}

	return nil
}

func add(tw *tar.Writer, src string) error {
	// walk path
	return filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {

		print(fi.Name())

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

func wrapWriterAES256(w io.WriteCloser, key [32]byte, iv []byte) (io.WriteCloser, error) {

	// Create writer
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return w, err
	}
	stream := cipher.NewOFB(block, iv)
	return cipher.StreamWriter{S: stream, W: w}, nil
}
