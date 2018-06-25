
# Distributed Key-value store based on the raft algorithm

See https://raft.github.io

A network is made out of multple nodes, which each can have one or more of the following functions:

- **StorageHandler** - Responsable for storing and retrieving key-value pairs.
- **QueryHandler** - Responsable for sending incoming request to the correct storage.
- **UIHandler** - Provides a web-based user interface

# Usage

    .\keyvaluestore.exe ca ca.pem
    .\keyvaluestore.exe newcert mynode ca.pem mynode.pem
    .\keyvaluestore.exe server -cert mynode.pem
