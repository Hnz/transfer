
# Transfer.go [![GoDoc](https://godoc.org/github.com/Hnz/transfer?status.svg)](https://godoc.org/github.com/Hnz/transfer) [![Go Report Card](https://goreportcard.com/badge/github.com/Hnz/transfer)](https://goreportcard.com/report/github.com/Hnz/transfer)

**transfer.go** is a commandline utility to upload files to https://transfer.sh.

Main features are:

- Automatically creates a tar archive when multiple files are selected
- Can encrypt files using AES256
- Can compress files using gzip
- Uses streams for maximum efficiency
- Full Windows support

# Examples

## Upload LICENSE.md
    $ transfer LICENSE.md
    https://transfer.sh/9mzIi/LICENSE.md

## Download LICENSE.md in the current directory
    $ transfer -g https://transfer.sh/9mzIi/LICENSE.md

## Create a tar.gz archive
    $ transfer -t -z LICENSE.md README.md
    https://transfer.sh/Qznmo/tar

## Download and unpack the archive in `mydir`
    $ transfer.exe -g -t -z -d mydir https://transfer.sh/Qznmo/tar

## Read from stdin and encrypt using `passwordfile`
    $ echo "secret message" | transfer -e -p paswordfile -
    https://transfer.sh/OaJRF/stdin
