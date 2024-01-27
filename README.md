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

# Stage 1
[Bencode](https://en.wikipedia.org/wiki/Bencode) (pronounced Bee-encode) is a serialization format used in the [BitTorrent protocol](https://www.bittorrent.org/beps/bep_0003.html). It is used in torrent files and in communication between trackers and peers.

Bencode supports four data types:
- strings
- integers
- arrays
- dictionaries

In this stage, we'll focus on decoding strings.

Strings are encoded as <length>:<contents>. For example, the string "hello" is encoded as "5:hello".

You'll implement a decode command which takes a bencoded value as input and prints the decoded value as JSON.

Here’s how the tester will execute your program:

```
$ ./your_bittorrent.sh decode 5:hello
"hello"
```

# Stage 2
In this stage, you'll extend the decode command to support bencoded integers.

Integers are encoded as i<number>e. For example, 52 is encoded as i52e and -52 is encoded as i-52e.

Here's how the tester will execute your program:

```
$ ./your_bittorrent.sh decode i52e
52
```

If you'd prefer to use a library for this stage, [bencode-go](https://github.com/jackpal/bencode-go) is available for you to use.
