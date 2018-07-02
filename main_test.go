// Copyright 2018 Hans van Leeuwen. All rights reserved.
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
)

var files = []string{"README.md", "LICENSE.md"}
var key = []byte("Test Key 12345678901234567890123")

func TestAes256(t *testing.T) {

	var r io.Reader
	var w io.WriteCloser

	in := []byte("A long time ago in a galaxy far, far away...\n")

	f, err := ioutil.TempFile("", "transfer_go")
	w = f
	handleError(err)

	w = wrapWriterAES256(w, key)

	_, err = w.Write(in)
	handleError(err)

	f, err = os.Open(f.Name())
	r = f
	handleError(err)
	defer f.Close()

	r, err = wrapReaderAES256(r, key)
	handleError(err)

	out, err := ioutil.ReadAll(r)
	handleError(err)

	if string(in) != string(out) {
		log.Fatalf("Input is different from output.\nIn:  %s\nOut: %s\n", in, out)
	}

}

func TestGzip(t *testing.T) {

	var r io.Reader
	var w io.WriteCloser

	in := []byte("A long time ago in a galaxy far, far away...\n")

	f, err := ioutil.TempFile("", "transfer_go")
	w = f
	handleError(err)

	w = wrapWriterGzip(w)

	_, err = w.Write(in)
	handleError(err)

	err = w.Close()
	handleError(err)

	err = f.Close()
	handleError(err)

	f, err = os.Open(f.Name())
	r = f
	handleError(err)

	r, err = wrapReaderGzip(r)
	handleError(err)

	out, err := ioutil.ReadAll(r)

	if string(in) != string(out) {
		log.Fatalf("Input is different from output.\nIn:  %s\nOut: %s\n", in, out)
	}
}

func TestHeader(t *testing.T) {

	var b bytes.Buffer

	var headerIn, headerOut Header
	headerIn = GZIP | TAR | AES256
	err := binary.Write(&b, binary.LittleEndian, headerIn)
	handleError(err)

	b.Write([]byte("extra content"))

	err = binary.Read(&b, binary.LittleEndian, &headerOut)
	handleError(err)

	assertEqual(t, headerIn, headerOut)
	assertEqual(t, headerOut.HasFlag(GZIP), true)
}

func TestPutGetMultiple(t *testing.T) {

	dir, err := ioutil.TempDir("", "transfer_go")
	handleError(err)
	defer os.RemoveAll(dir)

	// Create header
	var header Header
	header.AddFlag(AES256)
	header.AddFlag(GZIP)
	header.AddFlag(TAR)

	file := filepath.Join(dir, "archive")
	f, err := os.OpenFile(file, os.O_CREATE, 0600)
	handleError(err)

	Put(f, conf, getTestKey, files)
	f.Close()

	f, err = os.Open(file)
	handleError(err)
	Get(f, dir, getTestKey)
	f.Close()

	for _, file := range files {
		file1 := filepath.Join(dir, file)

		if !compareFiles(file1, file) {
			t.Fatalf("File %s is different then file %s", file1, file)
		}
	}
}

func TestPutGetSingle(t *testing.T) {

	dir, err := ioutil.TempDir("", "transfer_go")
	handleError(err)
	defer os.RemoveAll(dir)

	var conf = Config{
		Compress: true,
		Encrypt:  true,
		DestDir:  dir,
	}

	file := filepath.Join(dir, "archive")
	f, err := os.Create(file)
	handleError(err)

	Put(f, conf, getTestKey, []string{"README.md"})
	f.Close()

	f, err = os.Open(file)
	handleError(err)

	dest := filepath.Join(dir, "README.md")
	Get(f, dest, getTestKey)
	f.Close()

	if !compareFiles("README.md", dest) {
		t.Fatalf("File README.md is different then file %s", dest)
	}
}

func compareFiles(file1, file2 string) bool {
	file1stat, err := os.Stat(file1)
	handleError(err)
	file2stat, err := os.Stat(file2)
	handleError(err)

	return file1stat.Size() == file2stat.Size() && file1stat.Mode() == file2stat.Mode()
}

func assertEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Fatalf("%s != %s", a, b)
	}
}

func getTestKey() []byte {
	return key
}
