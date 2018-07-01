// Copyright 2018 Hans van Leeuwen. All rights reserved.
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

// Package gotransfer is a command line utility for uploading files to transfer.sh
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

// Version is the version of the application
const Version = "0.1.0"

// Config specifies configuration options
type Config struct {
	Cert         string   `json:"cert"`
	Compress     bool     `json:"compress"`
	DestDir      string   `json:"destdir"`
	Encrypt      bool     `json:"encrypt"`
	Host         string   `json:"host"`
	Key          [32]byte `json:"key"`
	MaxDownloads int      `json:"maxdownloads"`
	MaxDays      int      `json:"maxdays"`
	Port         int      `json:"port"`
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
	u := `GoTransfer %s

Usage:
  gotransfer get [options] <url>
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
  %s get [options] <url>

Options:
`
		fmt.Fprintf(os.Stderr, u, os.Args[0])
		flag.PrintDefaults()
		os.Exit(2)
	}

	conf := Config{}

	flag.BoolVar(&conf.Compress, "z", true, "Decompress the content using gzip.")
	flag.BoolVar(&conf.Encrypt, "e", true, "Decrypt the content using AES256.")
	flag.StringVar(&conf.DestDir, "d", ".", "Destination directory")

	parse(&conf)

	args := flag.Args()

	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "Error: Incorrect number of arguments.")
		flag.Usage()
	}

	conf.Key = getKey()

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

	flag.BoolVar(&conf.Compress, "z", true, "Compress the content using gzip.")
	flag.BoolVar(&conf.Encrypt, "e", true, "Encrypt the content using AES256.")
	flag.IntVar(&conf.MaxDays, "y", 14, "Remove the uploaded content after X days. Cannot be more than 14.")
	flag.IntVar(&conf.MaxDownloads, "w", 0, "Max amount of downloads to allow. Use 0 for unlimited.")

	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: Incorrect number of arguments.")
		flag.Usage()
	}

	conf.Key = getKey()

	var w io.WriteCloser
	r, w := io.Pipe()

	go Put(w, conf, args)

	// Make http request
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPut, "https://transfer.sh/MYFILE", r)
	handleError(err)

	// Set headers
	req.Header.Set("Max-Days", strconv.Itoa(conf.MaxDays))
	if conf.MaxDownloads != 0 {
		req.Header.Set("Max-Downloads", strconv.Itoa(conf.MaxDownloads))
	}

	// Get http response
	res, err := client.Do(req)
	handleError(err)
	defer res.Body.Close()

	// Output response body
	body, err := ioutil.ReadAll(res.Body)
	handleError(err)
	fmt.Println(string(body))
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

// Ask for password and hash it to create the key
func getKey() [32]byte {
	fmt.Print("Enter password: ")
	password, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Print("\n")
	handleError(err)
	return sha256.Sum256(password)
}
