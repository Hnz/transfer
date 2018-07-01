package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

var files = []string{"README.md", "LICENSE.md"}

func TestAes256(t *testing.T) {

	var r io.Reader
	var w io.WriteCloser

	in := []byte("A long time ago in a galaxy far, far away...\n")

	password := []byte("ThePassword")
	key := sha256.Sum256(password)

	f, err := ioutil.TempFile("", "transfer_go")
	w = f
	handleError(err)

	// Update header
	var header Header
	header.AddFlag(AES256)

	// Make random IV and write it to the output buffer
	iv := make([]byte, aes.BlockSize)
	io.ReadFull(rand.Reader, iv)
	w.Write(iv)

	// Create writer
	block, err := aes.NewCipher(key[:])
	handleError(err)
	stream := cipher.NewOFB(block, iv[:])
	w = cipher.StreamWriter{S: stream, W: w}

	_, err = w.Write(in)
	handleError(err)

	f, err = os.Open(f.Name())
	r = f
	handleError(err)
	defer f.Close()

	// First read the IV from the stream
	ivx := make([]byte, aes.BlockSize)
	io.ReadFull(r, ivx)
	if string(iv) != string(ivx) {
		log.Fatalf("IV input is different from output.\nIn:  %s\nOut: %s\n", iv, ivx)
	}

	// Create reader
	block, err = aes.NewCipher(key[:])
	handleError(err)
	stream = cipher.NewOFB(block, iv[:])
	r = cipher.StreamReader{S: stream, R: r}

	out, err := ioutil.ReadAll(r)

	fmt.Println(string(out))

	if string(in) != string(out) {
		log.Fatalf("Input is different from output.\nIn:  %s\nOut: %s\n", in, out)
	}

}

func TestGzip(t *testing.T) {

	var r io.ReadCloser
	var w io.WriteCloser

	in := []byte("A long time ago in a galaxy far, far away...\n")

	f, err := ioutil.TempFile("", "transfer_go")
	w = f
	handleError(err)

	w = WrapWriterGzip(w)

	_, err = w.Write(in)
	handleError(err)

	err = w.Close()
	handleError(err)

	err = f.Close()
	handleError(err)

	f, err = os.Open(f.Name())
	r = f
	handleError(err)

	r, err = WrapReaderGzip(r)
	handleError(err)

	out, err := ioutil.ReadAll(r)

	fmt.Println(string(out))

	if string(in) != string(out) {
		log.Fatalf("Input is different from output.\nIn:  %s\nOut: %s\n", in, out)
	}

	if err := r.Close(); err != nil {
		log.Fatal(err)
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

func TestPutGet(t *testing.T) {

	dir, err := ioutil.TempDir("", "transfer_go")
	handleError(err)
	defer os.RemoveAll(dir)

	var conf = Config{
		Compress: true,
		Encrypt:  false,
		DestDir:  dir,
	}

	file := filepath.Join(dir, "archive")
	f, err := os.OpenFile(file, os.O_CREATE, 0600)
	handleError(err)
	Put(f, conf, files)
	f.Close()

	f, err = os.Open(file)
	handleError(err)
	Get(f, conf)
	f.Close()

	file1 := filepath.Join(dir, "LICENSE.md")
	file2 := "LICENSE.md"

	if !compareFiles(file1, file2) {
		t.Fatalf("File %s is different then file %s", file1, file2)
	}

	file1 = filepath.Join(dir, "README.md")
	file2 = "README.md"

	if !compareFiles(file1, file2) {
		t.Fatalf("File %s is different then file %s", file1, file2)
	}
}

func compareFiles(file1, file2 string) bool {
	file1stat, err := os.Stat(file1)
	handleError(err)
	file2stat, err := os.Stat(file2)
	handleError(err)

	return file1stat.Size() == file2stat.Size() && file1stat.Mode() == file2stat.Mode()
}

func TestMain(t *testing.T) {

	data, err := os.Open("LICENSE.md")
	if err != nil {
		//handle error
		log.Fatal(err)
	}

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPut, "https://transfer.sh/LICENSE.md", data)
	if err != nil {
		// handle error
		log.Fatal(err)
	}
	res, err := client.Do(req)
	if err != nil {
		// handle error
		log.Fatal(err)
	}

	defer res.Body.Close()
	contents, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
	fmt.Printf("%s\n", string(contents))
}

func assertEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Fatalf("%s != %s", a, b)
	}
}
