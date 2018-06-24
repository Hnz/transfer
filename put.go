package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

func Put(w io.WriteCloser, conf Config, files []string) {

	defer w.Close()

	if conf.Encrypt {
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
		w = gzip.NewWriter(w)
		defer w.Close()
	}

	tw := tar.NewWriter(w)
	defer tw.Close()

	for _, f := range files {
		err := add(tw, f)
		handleError(err)
	}
}

func Upload(url string, r io.ReadCloser) string {

	// Make http request
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPut, url, r)
	handleError(err)

	//req.Header.Set("Content-Encoding", "identity")

	// Get http response
	res, err := client.Do(req)
	handleError(err)
	defer res.Body.Close()

	// Output response body
	body, err := ioutil.ReadAll(res.Body)
	handleError(err)
	return string(body)
}
