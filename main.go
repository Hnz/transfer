// Copyright 2018 Hans van Leeuwen. All rights reserved.
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

// Package transfer is a command line utility for uploading files to transfer.sh
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
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

// Version is the version of the application
const Version = "0.1.0"

// Salt is added to the password when hashing it
const Salt = "WGL4xaNR5mOZCmznamuxLIYNXja4uF7N"

// Config specifies configuration options
type Config struct {
	Compress bool   `json:"compress"`
	DestDir  string `json:"destdir"`
	Encrypt  bool   `json:"encrypt"`
	//Key          [32]byte `json:"key"`
	MaxDownloads int    `json:"maxdownloads"`
	MaxDays      int    `json:"maxdays"`
	Password     string `json:"password"`
}

var config Config
var configfile string

func main() {

	usr, _ := user.Current()
	configfile = filepath.Join(usr.HomeDir, ".transfer.conf")

	if len(os.Args) < 2 {
		printHelp()
	}

	// Read config file. Open file errors are ignored.
	f, err := os.Open(configfile)
	if err == nil {
		err = json.NewDecoder(f).Decode(&config)
		handleError(err)
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
	u := `Transfer %s

Usage:
  %s get [options] <url>
  %s put [options] <files...>
  %s -h | --help
  %s --version

  Config is read from %s
`
	fmt.Fprintf(os.Stderr, u, Version, os.Args[0], os.Args[0], os.Args[0], configfile)
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

	flag.BoolVar(&config.Compress, "z", true, "Decompress the content using gzip.")
	flag.BoolVar(&config.Encrypt, "e", true, "Decrypt the content using AES256.")
	flag.StringVar(&config.DestDir, "d", ".", "Destination directory.")

	args := parseArgs()

	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "Error: Incorrect number of arguments.")
		flag.Usage()
	}

	r, err := http.Get(args[0])
	handleError(err)

	Get(r.Body, config, getPassword)
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

	flag.BoolVar(&config.Compress, "z", true, "Compress the content using gzip.")
	flag.BoolVar(&config.Encrypt, "e", true, "Encrypt the content using AES256.")
	flag.IntVar(&config.MaxDays, "y", 14, "Remove the uploaded content after X days. Cannot be more than 14.")
	flag.IntVar(&config.MaxDownloads, "w", 0, "Max amount of downloads to allow. Use 0 for unlimited.")

	args := parseArgs()

	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: Incorrect number of arguments.")
		flag.Usage()
	}

	var w io.WriteCloser
	r, w := io.Pipe()

	go Put(w, config, getPassword, args)

	// Make http request
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPut, "https://transfer.sh/MYFILE", r)
	handleError(err)

	// Set headers
	req.Header.Set("Max-Days", strconv.Itoa(config.MaxDays))
	if config.MaxDownloads != 0 {
		req.Header.Set("Max-Downloads", strconv.Itoa(config.MaxDownloads))
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

func parseArgs() []string {

	v := flag.Bool("version", false, "Show version and exit.")
	flag.Parse()

	if *v {
		fmt.Println(Version)
		os.Exit(0)
	}

	return flag.Args()
}

func handleError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// Ask for password and hash it to create the key
func getPassword() []byte {
	fmt.Print("Enter password: ")
	password, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Print("\n")
	handleError(err)
	return password
}

func passwordToKey(password []byte) []byte {
	h := sha256.New()
	h.Write(password)
	h.Write([]byte(Salt))
	return h.Sum(nil)
}
