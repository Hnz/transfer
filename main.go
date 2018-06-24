// Copyright 2018 Hans van Leeuwen. All rights reserved.
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

// Package gotransfer implements a Distributed Key-Value Store
package main

import (
	"archive/tar"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

const Version = "0.0.1"

// Config specifies configuration options
type Config struct {
	Cert     string `json:"cert"`
	DestDir  string `json:"destdir"`
	Host     string `json:"host"`
	Key      string `json:"key"`
	Port     int    `json:"port"`
	Compress bool   `json:"compress"`
	Encrypt  bool   `json:"encrypt"`
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
	case "server":
		cmdServer()
	default:
		printHelp()
	}
}

func printHelp() {
	u := `GoTransfer %s

Usage:
  gotransfer get [options]
  gotransfer put [options] <files...>
  gotransfer server [options]
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

	flag.BoolVar(&conf.Compress, "c", true, "compress")
	flag.BoolVar(&conf.Encrypt, "e", true, "Encrypt")
	flag.StringVar(&conf.DestDir, "d", ".", "Destination directory")

	parse(&conf)

	args := flag.Args()

	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "Error: Incorrect number of arguments.")
		flag.Usage()
	}

	r, err := http.Get(args[0])
	handleError(err)

	Get(r.Body, conf)
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

	var w io.WriteCloser
	r, w := io.Pipe()

	go Put(w, conf, args)

	fmt.Println(Upload("https://transfer.sh/MYFILE", r))
}

func cmdServer() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:\n  %s server [options]\n\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
	}

	conf := Config{}

	hostname, _ := os.Hostname()

	flag.StringVar(&conf.Cert, "cert", "", "path to PEM-encoded certificate file.")
	flag.StringVar(&conf.Key, "key", "", "path to PEM-encoded private key file.")
	flag.StringVar(&conf.Host, "host", hostname, "Host to listen on.")
	flag.IntVar(&conf.Port, "port", 1234, "Port to listen on.")

	parse(&conf)

	address := fmt.Sprintf("%s:%d", conf.Host, conf.Port)
	httpserver := &http.Server{
		Addr:    address,
		Handler: httphandler{},
	}

	if conf.Cert == "" {
		log.Println("Listening on http://" + address)
		log.Fatal(httpserver.ListenAndServe())
	}

	if conf.Key == "" {
		conf.Key = conf.Cert
	}

	httpserver.TLSConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	log.Println("Listening on https://" + address)
	log.Fatal(httpserver.ListenAndServeTLS(conf.Cert, conf.Key))
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
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}

		return nil
	})
}

// Ask for password and hash it to create the key
func getKey() [32]byte {
	fmt.Print("Enter password: ")
	password, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Print("\n")
	handleError(err)
	return sha256.Sum256(password)
}
