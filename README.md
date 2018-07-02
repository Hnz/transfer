
# transfer.go

**transfer.go** is a commandline utility to upload files to https://transfer.sh.

The folks at transfer.sh advice you to use an [alias in your .bashrc file][1].

Main features are:

- Automatically creates a tar archive when multiple files are selected
- Can encrypt files using AES256
- Can compress files using gzip
- Uses streams for maximum efficiency
- Full Windows support

# Examples

Upload README.md and LICENSE.md, using default options.

    $ transfer put README.md LICENSE.md
    Enter password:
    https://transfer.sh/9mzIi/MYFILE

Retrieve an archive, decrypt it, and unpack it in the directory `tmp`.

    $ transfer get -d tmp https://transfer.sh/9mzIi/MYFILE
    Enter password:

[1]: https://gist.github.com/nl5887/a511f172d3fb3cd0e42d
