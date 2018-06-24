// Copyright 2018 Hans van Leeuwen. All rights reserved.
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

// Header represents the header of the archive
type Header uint16

const (
	// GZIP - compress using gzip
	GZIP Header = 1 << iota
	// AES256 - encrypt using AES256
	AES256
	// TAR - content is a tar archive
	TAR
)

// AddFlag adds a flag to the header
func (f *Header) AddFlag(flag Header) { *f |= flag }

// HasFlag checks if the header has flag defined
func (f Header) HasFlag(flag Header) bool { return f&flag != 0 }
