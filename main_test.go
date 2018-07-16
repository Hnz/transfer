// Copyright 2018 Hans van Leeuwen. All rights reserved.
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"crypto/aes"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

var baseURL string

type TestServerHandler struct {
	Basedir string
}

func (h TestServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPut {
		filename := filepath.Join(h.Basedir, path.Base(r.URL.Path))
		f, err := os.Create(filename)
		defer f.Close()
		defer r.Body.Close()
		if err != nil {
			panic(err)
		}
		io.Copy(f, r.Body)
		url := baseURL + r.URL.Path
		fmt.Fprintln(w, url)
	} else {
		fileserver := http.FileServer(http.Dir(h.Basedir))
		fileserver.ServeHTTP(w, r)
	}
}

func testServer(t *testing.T) (*httptest.Server, string) {
	dir, err := ioutil.TempDir("", "transfer")
	handleError(t, err)
	h := TestServerHandler{Basedir: dir}
	s := httptest.NewServer(h)
	baseURL = s.URL
	return s, dir
}

func TestUploadDownload(t *testing.T) {

	file := "LICENSE.md"

	// Create test server
	s, dir := testServer(t)
	defer s.Close()

	// Create test file
	f, err := os.Open(file)
	handleError(t, err)

	// Upload test file
	_, err = upload(f, s.URL+"/testfile", 1, 1)
	handleError(t, err)

	filename := filepath.Join(dir, "testfile")
	compareFiles(t, file, filename)

	// Download test file
	r, err := download(s.URL + "/testfile")
	handleError(t, err)
	w, err := ioutil.TempFile("", "transfer_go")
	handleError(t, err)

	_, err = io.Copy(w, r)
	handleError(t, err)

	// Check if download file is the same as uploaded file
	compareFiles(t, file, w.Name())
}

func TestWriteFile(t *testing.T) {
	var key [32]byte
	var iv [16]byte
	in := []byte("A long time ago in a galaxy far, far away...\n")

	// Create test file
	dir, err := ioutil.TempDir("", "transfer_go")
	handleError(t, err)
	filename := filepath.Join(dir, "in")

	ioutil.WriteFile(filename, in, 0600)

	r, err := os.Open(filename)
	handleError(t, err)

	w, err := os.Create(filename)
	handleError(t, err)

	err = writeFile(w, true, true, key, iv[:], r)
	handleError(t, err)
}

func TestAes256(t *testing.T) {

	var r io.Reader
	var w io.WriteCloser
	var key [32]byte
	var iv [16]byte

	in := []byte("A long time ago in a galaxy far, far away...\n")

	f, err := ioutil.TempFile("", "transfer_go")
	w = f
	handleError(t, err)
	defer os.Remove(f.Name())

	w, err = wrapWriterAES256(w, key, iv[:])
	handleError(t, err)

	_, err = w.Write(in)
	handleError(t, err)

	f, err = os.Open(f.Name())
	r = f
	handleError(t, err)
	defer f.Close()

	r, err = wrapReaderAES256(r, key, iv[:])
	handleError(t, err)

	out, err := ioutil.ReadAll(r)
	handleError(t, err)

	if string(in) != string(out) {
		log.Fatalf("Input is different from output.\nIn:  %s\nOut: %s\n", in, out)
	}

}

func TestGzip(t *testing.T) {

	var r io.Reader
	var w io.WriteCloser
	var out []byte
	in := []byte("A long time ago in a galaxy far, far away...\n")

	f, err := ioutil.TempFile("", "transfer_go")
	w = f
	handleError(t, err)
	defer os.Remove(f.Name())

	w = wrapWriterGzip(w)

	_, err = w.Write(in)
	handleError(t, err)

	err = w.Close()
	handleError(t, err)

	err = f.Close()
	handleError(t, err)

	f, err = os.Open(f.Name())
	r = f
	handleError(t, err)

	r, err = wrapReaderGzip(r)
	handleError(t, err)

	out, err = ioutil.ReadAll(r)
	handleError(t, err)

	if string(in) != string(out) {
		log.Fatalf("Input is different from output.\nIn:  %s\nOut: %s\n", in, out)
	}
}

func TestSingleFile(t *testing.T) {

	var key [32]byte
	var iv [16]byte
	var file = "LICENSE.md"
	var files = []string{file}
	var configs = []Config{
		{Compress: false, Encrypt: false, Tar: false},
		{Compress: false, Encrypt: false, Tar: true},
		{Compress: false, Encrypt: true, Tar: false},
		{Compress: false, Encrypt: true, Tar: true},
		{Compress: true, Encrypt: false, Tar: false},
		{Compress: true, Encrypt: false, Tar: true},
		{Compress: true, Encrypt: true, Tar: false},
		{Compress: true, Encrypt: true, Tar: true},
	}

	for _, config := range configs {
		var buf bytes.Buffer

		// Create test file
		outdir, err := ioutil.TempDir("", "transfer")
		defer os.RemoveAll(outdir)
		handleError(t, err)

		// Create test server
		s, dir := testServer(t)
		defer s.Close()
		defer os.RemoveAll(dir)

		config.BaseURL = s.URL
		config.Dest = outdir

		err = Put(config, files, &buf, key, iv[:])
		handleError(t, err)
		url := strings.TrimRight(buf.String(), "\n")
		err = Get(config, []string{url}, key, iv[:])
		handleError(t, err)

		// Check if download file is the same as uploaded file
		compareFiles(t, file, filepath.Join(outdir, file))
	}
}

func handleError(t *testing.T, err error) {
	if err != nil {
		panic(err)
	}
}

func compareFiles(t *testing.T, file1, file2 string) {
	file1stat, err := os.Stat(file1)
	handleError(t, err)
	file2stat, err := os.Stat(file2)
	handleError(t, err)

	b := file1stat.Size() == file2stat.Size()
	if !b {
		t.Fatalf("File %s differs from file %s", file1, file2)
	}
}

/*
Try to replicate the openssl encryption in go

$ openssl enc -aes-256-cbc -P -pass pass:test -S F6818CAE131872BD -md SHA256
salt=F6818CAE131872BD
key=109AE1C21965E57876731402D8DC5276A59B8782AEC354D7BF387A2DC77450F1
iv =0899F50C65F644985C9CEAD9773AEEA5
*/
func TestOpenSSL(t *testing.T) {
	pw := []byte("test")
	key, err := hex.DecodeString("109AE1C21965E57876731402D8DC5276A59B8782AEC354D7BF387A2DC77450F1")
	handleError(t, err)

	iv, err := hex.DecodeString("0899F50C65F644985C9CEAD9773AEEA5")
	handleError(t, err)

	outKey, outIV := passwordToKey(pw)

	if hex.EncodeToString(key) != hex.EncodeToString(outKey[:]) {
		t.Fatalf("%x does not equal %x", key, outKey)
	}

	if hex.EncodeToString(iv) != hex.EncodeToString(outIV[:aes.BlockSize]) {
		t.Fatalf("%x does not equal %x", iv, outIV[:aes.BlockSize])
	}
}
