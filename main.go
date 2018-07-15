// Copyright 2018 Hans van Leeuwen. All rights reserved.
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

// Package transfer is a command line utility for uploading files to transfer.sh
package main

import (
	"flag"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

// Version is the version of the application
const Version = "0.3.0"
const useragent = "Transfer.go/" + Version

var verbose bool

// Config specifies configuration options
type Config struct {
	BaseURL      string
	Compress     bool
	Dest         string
	Encrypt      bool
	PasswordFile string
	MaxDownloads int
	MaxDays      int
	Tar          bool
	Verbose      bool
}

func main() {
	var config Config
	var err error

	flag.StringVar(&config.BaseURL, "b", "https://transfer.sh", "Base url.")
	flag.BoolVar(&config.Compress, "z", false, "Compress the content using gzip.")
	flag.StringVar(&config.Dest, "d", "", "Directory in which to place the downloaded file.")
	flag.BoolVar(&config.Encrypt, "e", false, "Encrypt the content using AES256.")
	flag.StringVar(&config.PasswordFile, "p", "", "File from which to load the encryption password.")
	flag.IntVar(&config.MaxDays, "y", 0, "Remove the uploaded content after X days.")
	flag.IntVar(&config.MaxDownloads, "m", 0, "Max amount of downloads to allow. Use 0 for unlimited.")
	flag.BoolVar(&config.Tar, "t", false, "Create a tar archive.")
	flag.BoolVar(&config.Verbose, "v", false, "Output log.")

	get := flag.Bool("g", false, "Get")

	flag.Usage = printHelp
	flag.Parse()
	args := flag.Args()

	verbose = config.Verbose

	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: Incorrect number of arguments.")
		flag.Usage()
	}

	// Get the password if needed
	key, err := getKey(config, args)

	if *get {
		err = Get(config, args, key)
	} else {
		err = Put(config, args, os.Stdout, key)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
	}
}

func printHelp() {
	u := `Usage:
%s [options] <files...>

Options:
`
	fmt.Fprintf(os.Stderr, u, os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, `
Examples:
  $ transfer LICENSE.md
  https://transfer.sh/9mzIi/LICENSE.md

  $ transfer -t -z LICENSE.md README.md
  https://transfer.sh/Qznmo/tar

  # Read from stdin and encrypt
  $ echo "secret message" | transfer -e -
  Enter password:
  https://transfer.sh/OaJRF/LICENSE.md
  https://transfer.sh/G0UXd/README.md
`)
	os.Exit(2)
}

func print(s string) {
	if verbose {
		fmt.Println(s)
	}
}

func getPassword() ([]byte, error) {
	fmt.Print("Enter password: ")
	password, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println("")
	return password, err
}
