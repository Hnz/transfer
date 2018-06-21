// Copyright 2018 Hans van Leeuwen. All rights reserved.
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

// Package gotransfer implements a Distributed Key-Value Store
package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const Version = "0.0.1"

// Config specifies configuration options
type Config struct {
	Compress bool `json:"compress"`
	Encrypt  bool `json:"encrypt"`
}

var configfile string

func main() {

	// Set flags that apply to all commands
	flag.StringVar(&configfile, "configfile", "", "Path to a JSON-formatted config file. Options read "+
		"from the config file will \n        overwrite options set on the commandline.")

	if len(os.Args) < 2 {
		printHelp()
	}

	command := os.Args[1]

	// Remove command from os.Args
	os.Args = append(os.Args[:1], os.Args[2:]...)

	switch command {
	case "get":
		cmdGet()
	case "put":
		cmdPut()
	default:
		printHelp()
	}
}

func printHelp() {
	u := `Key Value Store %s

Usage:
  gotransfer get [options]
  gotransfer put [options] <files...>
  gotransfer -h | --help

Options:
`
	fmt.Fprintf(os.Stderr, u, Version)
	flag.PrintDefaults()
	os.Exit(2)
}

func cmdGet() {
	flag.Usage = func() {
		u := `Usage:
  %s get [options]

Options:
`
		fmt.Fprintf(os.Stderr, u, os.Args[0])
		flag.PrintDefaults()
		os.Exit(2)
	}

	conf := Config{}

	parse(&conf)

	args := flag.Args()

	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "Error: Incorrect number of arguments.")
		flag.Usage()
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fmt.Println("> " + scanner.Text())
	}
}

func cmdPut() {
	flag.Usage = func() {
		u := `Usage:
  %s put [options] <files...>

Options:
`
		fmt.Fprintf(os.Stderr, u, os.Args[0])
		flag.PrintDefaults()
		os.Exit(2)
	}

	conf := Config{}

	flag.BoolVar(&conf.Compress, "c", true, "compress")
	flag.BoolVar(&conf.Encrypt, "e", true, "Encrypt")

	parse(&conf)

	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: Incorrect number of arguments.")
		flag.Usage()
	}

	var out io.WriteCloser
	out = os.Stdout

	if conf.Encrypt {
		// Ask for password
		var password string
		fmt.Print("Enter password: ")
		fmt.Scanln(&password)
		key := sha256.Sum256([]byte(password))
		fmt.Println(key)

		// TODO: Make iv random
		iv := make([]byte, aes.BlockSize)
		io.ReadFull(rand.Reader, iv)

		block, err := aes.NewCipher(key[:])
		handleError(err)
		stream := cipher.NewOFB(block, iv[:])
		out = cipher.StreamWriter{S: stream, W: out}
		defer out.Close()
	}

	if conf.Compress {
		out = gzip.NewWriter(out)
		defer out.Close()
	}

	tw := tar.NewWriter(out)
	defer tw.Close()

	for _, f := range args {
		err := add(tw, f)
		if err != nil {
			handleError(err)
		}
	}
}

func parse(conf *Config) {

	flag.Parse()

	flag.Visit(func(f *flag.Flag) {
		fmt.Println("FLAG", f.Name, f.Value)
	})

	// Parse config file
	if configfile != "" {
		f, err := os.Open(configfile)
		handleError(err)

		err = json.NewDecoder(f).Decode(&conf)
		handleError(err)
	}
}

func handleError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func add(tw *tar.Writer, src string) error {
	// walk path
	return filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {

		handleError(err)

		// create a new dir/file header
		header, err := tar.FileInfoHeader(fi, fi.Name())
		handleError(err)

		// update the name to correctly reflect the desired destination when untaring
		header.Name = strings.TrimPrefix(strings.Replace(file, src, "", -1), string(filepath.Separator))

		// write the header
		err = tw.WriteHeader(header)
		handleError(err)

		// return on non-regular files (thanks to [kumo](https://medium.com/@komuw/just-like-you-did-fbdd7df829d3) for this suggested update)
		if !fi.Mode().IsRegular() {
			return nil
		}

		// open files for taring
		f, err := os.Open(file)
		defer f.Close()
		handleError(err)

		// copy file data into tar writer
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}

		return nil
	})
}
