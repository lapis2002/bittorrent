[![progress-banner](https://backend.codecrafters.io/progress/bittorrent/cc170dc5-6c1b-49f2-9b7e-2d22e3750ae8)](https://app.codecrafters.io/users/codecrafters-bot?r=2qF)

This is a starting point for Go solutions to the
["Build Your Own BitTorrent" Challenge](https://app.codecrafters.io/courses/bittorrent/overview).

In this challenge, you’ll build a BitTorrent client that's capable of parsing a
.torrent file and downloading a file from a peer. Along the way, we’ll learn
about how torrent files are structured, HTTP trackers, BitTorrent’s Peer
Protocol, pipelining and more.

**Note**: If you're viewing this repo on GitHub, head over to
[codecrafters.io](https://codecrafters.io) to try the challenge.

# Passing the first stage

The entry point for your BitTorrent implementation is in
`cmd/mybittorrent/main.go`. Study and uncomment the relevant code, and push your
changes to pass the first stage:

```sh
git add .
git commit -m "pass 1st stage" # any msg
git push origin master
```
## Testing

1. Ensure you have `go (1.19)` installed locally
2. Run `./your_bittorrent.sh` to run your program, which is implemented in
   `cmd/mybittorrent/main.go`.
3. Commit your changes and run `git push origin master` to submit your solution
   to CodeCrafters. Test output will be streamed to your terminal.

# Stage 1: Decode bencoded strings
[Bencode](https://en.wikipedia.org/wiki/Bencode) (pronounced Bee-encode) is a serialization format used in the [BitTorrent protocol](https://www.bittorrent.org/beps/bep_0003.html). It is used in torrent files and in communication between trackers and peers.

Bencode supports four data types:
- strings
- integers
- arrays
- dictionaries

In this stage, we'll focus on decoding strings.

Strings are encoded as \<length>:\<contents>. For example, the string `"hello"` is encoded as `"5:hello"`.

You'll implement a decode command which takes a bencoded value as input and prints the decoded value as JSON.

---
Here’s how the tester will execute your program:

```
$ ./your_bittorrent.sh decode 5:hello
"hello"
```

# Stage 2: Decode bencoded integers
In this stage, you'll extend the decode command to support bencoded integers.

Integers are encoded as i\<number>e. For example, `52` is encoded as `i52e` and `-52` is encoded as `i-52e`.

Here's how the tester will execute your program:

```
$ ./your_bittorrent.sh decode i52e
52
```

If you'd prefer to use a library for this stage, [bencode-go](https://github.com/jackpal/bencode-go) is available for you to use.

# Stage 3: Decoding bencoded lists


In this stage, you'll extend the decode command to support bencoded lists.

Lists are encoded as l<bencoded_elements>e.

For example, `["hello", 52]` would be encoded as `l5:helloi52ee`. Note that there are no separators between the elements.

---
Here’s how the tester will execute your program:

```
$ ./your_bittorrent.sh decode l5:helloi52ee
[“hello”,52]
```

If you'd prefer to use a library for this stage, [bencode-go](https://github.com/jackpal/bencode-go) is available for you to use.

# Stage 4: Decode bencoded dictionaries
In this stage, you'll extend the decode command to support bencoded dictionaries.

A dictionary is encoded as d\<key1>\<value1>...\<keyN>\<valueN>e. \<key1>, \<value1> etc. correspond to the bencoded keys & values. The keys are sorted in lexicographical order and must be strings.

For example, {"hello": 52, "foo":"bar"} would be encoded as: d3:foo3:bar5:helloi52ee (note that the keys were reordered).

---
Here’s how the tester will execute your program:

```
$ ./your_bittorrent.sh decode d3:foo3:bar5:helloi52ee
{"foo":"bar","hello":52}
```
If you'd prefer to use a library for this stage, [bencode-go](https://github.com/jackpal/bencode-go) is available for you to use.

# Stage 5: Parse torrent file
In this stage, you'll parse a torrent file and print information about the torrent.

A torrent file (also known as a metainfo file) contains a bencoded dictionary with the following keys and values:
- `announce`:
    - URL to a "tracker", which is a central server that keeps track of peers participating in the sharing of a torrent.
- `info`:
    - A dictionary with keys:
        - `length`: size of the file in bytes, for single-file torrents
        - `name`: suggested name to save the file / directory as
        - `piece length`: number of bytes in each piece
        - `pieces`: concatenated SHA-1 hashes of each piece

Note: The info dictionary looks slightly different for multi-file torrents. For this challenge, we'll only implement support for single-file torrents.

In this stage, we'll focus on extracting the tracker URL and the length of the file (in bytes).

---
Here’s how the tester will execute your program:

```
$ ./your_bittorrent.sh info sample.torrent
```
and here’s the output it expects:

```
Tracker URL: http://bittorrent-test-tracker.codecrafters.io/announce
Length: 92063
```

# Stage 6: Calculate info hash
Info hash is a unique identifier for a torrent file. It's used when talking to trackers or peers.

In this stage, you'll calculate the info hash for a torrent file and print it in hexadecimal format.

To calculate the info hash, you'll need to:
- Extract the info dictionary from the torrent file after parsing
- Bencode the contents of the info dictionary
- Calculate the SHA-1 hash of this bencoded dictionary

---
Here’s how the tester will execute your program:

```
$ ./your_bittorrent.sh info sample.torrent
```

and here’s the output it expects:

```
Tracker URL: http://bittorrent-test-tracker.codecrafters.io/announce
Length: 92063
Info Hash: d69f91e6b2ae4c542468d1073a71d4ea13879a7f
```
# Stage 7: Piece Hashes
In a torrent, a file is split into equally-sized parts called pieces. A piece is usually 256 KB or 1 MB in size.

Each piece is assigned a SHA-1 hash value. On public networks, there may be malicious peers that send fake data. These hash values allow us to verify the integrity of each piece that we'll download.

Piece length and piece hashes are specified in the info dictionary of the torrent file under the following keys:
- `piece length`: number of bytes in each piece, an integer
- `pieces`: concatenated SHA-1 hashes of each piece (20 bytes each), a string

The [BitTorrent Protocol Specification](https://www.bittorrent.org/beps/bep_0003.html#info-dictionary) has more information about these keys.

In this stage, the tester will expect your program to print piece length and a list of piece hashes in hexadecimal format.

---
Here's how the tester will execute your program:
```
$ ./your_bittorrent.sh info sample.torrent
```
and here's the output it expects:
```
Tracker URL: http://bittorrent-test-tracker.codecrafters.io/announce
Length: 92063
Info Hash: d69f91e6b2ae4c542468d1073a71d4ea13879a7f
Piece Length: 32768
Piece Hashes:
e876f67a2a8886e8f36b136726c30fa29703022d
6e2275e604a0766656736e81ff10b55204ad8d35
f00d937a0213df1982bc8d097227ad9e909acc17
```

# Stage 8: Discover peers


Trackers are central servers that maintain information about peers participating in the sharing and downloading of a torrent.

In this stage, you'll make a GET request to a HTTP tracker to discover peers to download the file from.
### Tracker GET request

You'll need to make a request to the tracker URL you extracted in the previous stage, and include these query params:
- `info_hash`: the info hash of the torrent
    - 20 bytes long, will need to be URL encoded
    - **Note**: this is **NOT** the hexadecimal representation, which is 40 bytes long
- `peer_id`: a unique identifier for your client
    - A string of length 20 that you get to pick. You can use something like 00112233445566778899.
- `port`: the port your client is listening on
    - You can set this to 6881, you will not have to support this functionality during this challenge.
- `uploaded`: the total amount uploaded so far
    - Since your client hasn't uploaded anything yet, you can set this to 0.
- `downloaded`: the total amount downloaded so far
    - Since your client hasn't downloaded anything yet, you can set this to 0.
- `left`: the number of bytes left to download
    - Since you client hasn't downloaded anything yet, this'll be the total length of the file (you've extracted this value from the torrent file in previous stages)
- `compact`: whether the peer list should use the [compact representation](https://www.bittorrent.org/beps/bep_0023.html)
    - For the purposes of this challenge, set this to 1.
    - The compact representation is more commonly used in the wild, the non-compact representation is mostly supported for backward-compatibility.

Read the [BitTorrent Protocol Specification](https://www.bittorrent.org/beps/bep_0003.html#trackers) for more information about these query parameters.

### Tracker response

The tracker's response will be a bencoded dictionary with two keys:
- `interval`:
   - An integer, indicating how often your client should make a request to the tracker.
   - You can ignore this value for the purposes of this challenge.
- `peers`:
   - A string, which contains list of peers that your client can connect to.
   - Each peer is represented using 6 bytes. The first 4 bytes are the peer's IP address and the last 2 bytes are the peer's port number.

---
Here’s how the tester will execute your program:
```
$ ./your_bittorrent.sh peers sample.torrent
```
and here’s the output it expects:
```
178.62.82.89:51470
165.232.33.77:51467
178.62.85.20:51489
```

# Stage 9: Peer handshake


In this stage, you’ll establish a TCP connection with a peer and complete a handshake.

The handshake is a message consisting of the following parts as described in the [peer protocol](https://www.bittorrent.org/beps/bep_0003.html#peer-protocol):
- length of the protocol string (BitTorrent protocol) which is `19` (1 byte)
- the string `BitTorrent protocol` (19 bytes)
- eight reserved bytes, which are all set to zero (8 bytes)
- sha1 infohash (20 bytes) (**NOT** the hexadecimal representation, which is 40 bytes long)
- peer id (20 bytes) (you can use `00112233445566778899` for this challenge)

After we send a handshake to our peer, we should receive a handshake back in the same format.

Your program should print the hexadecimal representation of the peer id you've received during the handshake.

---
Here’s how the tester will execute your program:
```
$ ./your_bittorrent.sh handshake sample.torrent <peer_ip>:<peer_port>
```
and here’s the output it expects:
```
Peer ID: 0102030405060708090a0b0c0d0e0f1011121314
```
(Exact value will be different as it is randomly generated.)

**Note**: To get a peer IP & port to test this locally, run `./your_bittorrent.sh peers sample.torrent` and pick any peer from the list.


