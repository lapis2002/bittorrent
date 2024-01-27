package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"unicode"
)

// The 'bencode' tag in PieceLength field is due to
// the fact that Go uses a capitalised version of the key by default,
// so we need to explicitly tell it to look for 'piece length'.
type MetaInfo struct {
	Announce string
	Info     struct {
		Length      int
		Name        string
		PieceLength int `bencode:"piece length"`
		Pieces      string
	}
}

type TorrentFile struct {
	metadata MetaInfo
	infoHash [20]byte
}

type TrackerResponse struct {
	Interval int
	Peers    string
}

const PEER_ID = "00112233445566778899"
const CLIENT_PORT = "6881"

// Example:
// - 5:hello -> hello
// - 10:hello12345 -> hello12345
func decodeBencode(bencodedString string) (interface{}, error) {
	if unicode.IsDigit(rune(bencodedString[0])) {
		decoded, _, err := decodeBencodeString(bencodedString)
		return decoded, err
	} else if bencodedString[0] == 'i' {
		decoded, _, err := decodeBencodeInt(bencodedString)
		return decoded, err
	} else if bencodedString[0] == 'l' {
		decoded, _, err := decodeBencodeList(bencodedString)
		return decoded, err
	} else if bencodedString[0] == 'd' {
		decoded, _, err := decodeBencodeDict(bencodedString)
		return decoded, err
	} else {
		return "", fmt.Errorf("only strings are supported at the moment")
	}
}

func decodeBencodeString(bencodedString string) (interface{}, int, error) {
	var firstColonIndex int
	for i := 0; i < len(bencodedString); i++ {
		if bencodedString[i] == ':' {
			firstColonIndex = i
			break
		}
	}

	lengthStr := bencodedString[:firstColonIndex]

	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return "", 0, err
	}

	decodedString := bencodedString[firstColonIndex+1 : firstColonIndex+1+length]
	return decodedString, firstColonIndex + 1 + length, nil
}
func decodeBencodeInt(bencodedString string) (interface{}, int, error) {
	var firstEIndex int // default 0
	for i := 0; i < len(bencodedString); i++ {
		if bencodedString[i] == 'e' {
			firstEIndex = i
			break
		}
	}

	decodeIntStr := bencodedString[1:firstEIndex]
	decodeInt, err := strconv.Atoi(decodeIntStr)
	if err != nil {
		return "", 0, err
	}
	return decodeInt, firstEIndex + 1, nil // include first 'i' and ending 'e'
}

func decodeBencodeList(bencodedString string) ([]interface{}, int, error) {
	decodedList := []interface{}{}

	toDecode := bencodedString[1:]
	totLen := 1
	for toDecode[0] != 'e' {
		var decoded interface{}
		var decodedLen int
		var err error
		switch toDecode[0] {
		case 'i':
			decoded, decodedLen, err = decodeBencodeInt(toDecode)
		case 'l':
			decoded, decodedLen, err = decodeBencodeList(toDecode)
		default:
			decoded, decodedLen, err = decodeBencodeString(toDecode)
		}

		if err != nil {
			return nil, 0, err
		}
		decodedList = append(decodedList, decoded)
		toDecode = toDecode[decodedLen:]
		totLen += decodedLen
	}
	if len(toDecode) == 0 || toDecode[0] != 'e' {
		return nil, 0, fmt.Errorf("list not properly terminated with 'e'")
	}

	return decodedList, totLen + 1, nil // include first 'i' and ending 'e'
}

func decodeBencodeDict(bencodedString string) (map[string]interface{}, int, error) {
	decodedDict := map[string]interface{}{}

	toDecode := bencodedString[1:]
	totLen := 1

	var decodedKey string
	var decodedValue interface{} = ""

	for toDecode[0] != 'e' {
		var decoded interface{}
		var decodedLen int
		var err error
		switch toDecode[0] {
		case 'i':
			decoded, decodedLen, err = decodeBencodeInt(toDecode)
		case 'l':
			decoded, decodedLen, err = decodeBencodeList(toDecode)
		case 'd':
			decoded, decodedLen, err = decodeBencodeDict(toDecode)
		default:
			decoded, decodedLen, err = decodeBencodeString(toDecode)
		}

		if err != nil {
			return nil, 0, err
		}

		if decodedKey == "" {
			decodedKey = decoded.(string)
		} else {
			decodedValue = decoded
		}
		if decodedKey != "" && decodedValue != "" {
			decodedDict[decodedKey] = decodedValue
			decodedKey = ""
			decodedValue = ""
		}
		toDecode = toDecode[decodedLen:]
		totLen += decodedLen
	}
	if len(toDecode) == 0 || toDecode[0] != 'e' {
		return nil, 0, fmt.Errorf("list not properly terminated with 'e'")
	}

	return decodedDict, totLen + 1, nil // include first 'i' and ending 'e'
}
func readTorrentFile(filename string) (TorrentFile, error) {
	var torrentFile TorrentFile
	buffer, err := os.ReadFile(filename)
	if err != nil {
		return torrentFile, err
	}
	s := string(buffer)
	decoded, err := decodeBencode(s)

	if err != nil {
		return torrentFile, err
	}
	decodedMap := decoded.(map[string]interface{})

	torrentFile.metadata.Announce = decodedMap["announce"].(string)
	infoMap := decodedMap["info"].(map[string]interface{})
	torrentFile.metadata.Info.Length = int(infoMap["length"].(int))
	torrentFile.metadata.Info.Name = infoMap["name"].(string)
	torrentFile.metadata.Info.PieceLength = int(infoMap["piece length"].(int))
	torrentFile.metadata.Info.Pieces = infoMap["pieces"].(string)

	infoMapEncoded, err := encodeBencodeDict(infoMap)
	if err != nil {
		return torrentFile, err
	}

	torrentFile.infoHash = sha1.Sum([]byte(infoMapEncoded))

	return torrentFile, nil
}

func encodeBencodeString(str string) string {
	return fmt.Sprintf("%d:%s", len(str), str)
}

func encodeBencodeInt(i int) string {
	return fmt.Sprintf("i%de", i)
}

func encodeBencodeDict(infoMap map[string]interface{}) (string, error) {
	var encodedString string
	keys := make([]string, 0, len(infoMap))
	for k := range infoMap {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		encodedString += encodeBencodeString(k)
		switch v := infoMap[k].(type) {
		case string:
			{
				encodedString += encodeBencodeString(v)
			}
		case int:
			{
				encodedString += encodeBencodeInt(v)
			}
		}
	}
	return fmt.Sprintf("d%se", encodedString), nil
}

func getPieceHashes(pieces []byte) []string {
	var pieceHashes []string
	for i := 0; i < len(pieces); i += 20 {
		hash := pieces[i : i+20]
		pieceHash := hex.EncodeToString(hash)
		pieceHashes = append(pieceHashes, pieceHash)
	}

	return pieceHashes
}

func discoverPeers(torrentFile TorrentFile) ([]string, error) {
	params := url.Values{}
	params.Add("info_hash", string(torrentFile.infoHash[:]))
	params.Add("peer_id", PEER_ID)
	params.Add("port", CLIENT_PORT)
	params.Add("uploaded", "0")
	params.Add("downloaded", "0")
	params.Add("left", strconv.Itoa(int(torrentFile.metadata.Info.Length)))
	params.Add("compact", "1")
	url := fmt.Sprint(torrentFile.metadata.Announce + "?" + params.Encode())

	response, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error in get request %v", err)
		return nil, err
	}
	defer response.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)
	respBytes := buf.String()
	respString := string(respBytes)

	decoded, err := decodeBencode(respString)
	if err != nil {
		return nil, err
	}

	decodedMap, ok := decoded.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected decoded value type")
	}

	interval, ok := decodedMap["interval"].(int)
	if !ok {
		return nil, fmt.Errorf("unexpected type in `interval`")
	}

	peers, ok := decodedMap["peers"].(string)
	if !ok {
		fmt.Println("Unexpected type in `peers`")
		return nil, fmt.Errorf("unexpected type in `peers`")
	}

	trackerResp := TrackerResponse{
		Interval: interval,
		Peers:    peers,
	}
	var peersList []string

	for i := 0; i <= len(trackerResp.Peers)-6; i += 6 {
		peer := []byte(trackerResp.Peers[i : i+4])
		port := binary.BigEndian.Uint16([]byte(trackerResp.Peers[i+4 : i+6]))
		fmt.Printf("%v.%v.%v.%v:%d\n", peer[0], peer[1], peer[2], peer[3], port)

		address := net.IPv4(peer[0], peer[1], peer[2], peer[3])
		peersList = append(peersList, fmt.Sprintf("%s:%d", address, port))
	}

	return peersList, nil
}

func connectToPeer(peerAddress string) (net.Conn, error) {
	conn, err := net.Dial("tcp", peerAddress)
	if err != nil {
		return nil, err
	}

	// defer used to schedule a function call to be executed when the surrounding function returns.
	// defer conn.Close() is used to defer the execution of the Close() method on the conn object
	// defer conn.Close()
	return conn, nil
}

func peerHandshake(conn net.Conn, infoHash []byte) (string, error) {
	pstrlen := byte(19)                   // length of the protocol string BitTorrent protocol
	pstr := []byte("BitTorrent protocol") // the string BitTorrent protocol (19 bytes)
	reserved := make([]byte, 8)           // Eight zeros
	handshake := append([]byte{pstrlen}, pstr...)
	handshake = append(handshake, reserved...)
	handshake = append(handshake, infoHash...)
	handshake = append(handshake, []byte(PEER_ID)...)
	// Send Handshake
	_, err := conn.Write(handshake)
	if err != nil {
		return "", err
	}

	buffer := make([]byte, len(handshake))
	_, err = conn.Read(buffer)
	if err != nil {
		return "", err
	}

	fmt.Printf("Peer ID: %x\n", buffer[len(buffer)-20:])
	return string(buffer[len(buffer)-20:]), nil
}

func exchangePerrMessages(torrentFile TorrentFile) (net.Conn, error) {
	peers, err := discoverPeers(torrentFile)
	if err != nil {
		return nil, err
	}

	if len(peers) == 0 {
		return nil, errors.New("no peers found")
	}

	conn, err := connectToPeer(peers[0])
	if err != nil {
		return conn, err
	}

	_, err = peerHandshake(conn, torrentFile.infoHash[:])
	if err != nil {
		return conn, err
	}

	msg, err := readPeerMessage(conn, Bitfield)
	if err != nil {
		return conn, err
	}

	msg = createPeerMessage(Interested)
	_, err = conn.Write(msg.Bytes())
	if err != nil {
		return conn, err
	}

	msg, err = readPeerMessage(conn, Unchoke)
	if err != nil {
		return conn, err
	}

	return conn, nil
}

func downloadPiece(conn net.Conn, torrentFile TorrentFile, pieceIdx int) ([]byte, error) {
	pieceHashes := getPieceHashes([]byte(torrentFile.metadata.Info.Pieces))
	pieceHash := pieceHashes[pieceIdx]
	fmt.Println("Requesting piece", pieceIdx, "with hash", pieceHash)

	maxlen := torrentFile.metadata.Info.Length - pieceIdx*torrentFile.metadata.Info.PieceLength
	blockSize := 1 << 14 // 2^14

	var piece []byte
	for i := 0; i < torrentFile.metadata.Info.PieceLength; i += blockSize {
		length := math.Min(float64(maxlen-i), float64(blockSize))
		payload := createPayload(uint32(pieceIdx), uint32(i), uint32(length))

		msg := createPeerMessage(Request)
		msg.setPayload(payload)

		_, err := conn.Write(msg.Bytes())
		if err != nil {
			return nil, err
		}
		msg, err = readPeerMessage(conn, Piece)
		if err != nil {
			return nil, err
		}
		if msg.length > 0 {
			piece = append(piece, msg.payload[8:msg.length]...)
		}
	}

	// Verify the piece hash
	h := sha1.New()
	h.Write(piece)
	hsum := hex.EncodeToString(h.Sum(nil))
	if hsum != pieceHash {
		return nil, fmt.Errorf("piece hash does not match (%x != %x)", hsum, pieceHash)
	}

	return piece, nil
}

func writePieceToFile(piece []byte, outputPath string) error {
	// Write the piece to the output file
	err := os.WriteFile(outputPath, piece, 0644)
	if err != nil {
		return err
	}
	return nil
}

func downloadFile(filename string, outputPath string) error {
	torrentFile, err := readTorrentFile(filename)
	if err != nil {
		return err
	}

	conn, err := exchangePerrMessages(torrentFile)
	if err != nil {
		return err
	}

	pieceHashes := getPieceHashes([]byte(torrentFile.metadata.Info.Pieces))

	filePieces := make([]byte, 0)
	for i := range pieceHashes {
		piece, err := downloadPiece(conn, torrentFile, i)
		if err != nil {
			return err
		}
		filePieces = append(filePieces, piece...)
	}

	// write to file
	err = writePieceToFile(filePieces, outputPath)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	// fmt.Println("Logs from your program will appear here!")

	command := os.Args[1]
	switch command {
	case "decode":
		bencodedValue := os.Args[2]
		decoded, err := decodeBencode(bencodedValue)
		if err != nil {
			log.Fatal(err)
		}

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))

	case "info":
		torrentFile, err := readTorrentFile(os.Args[2])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Tracker URL:", torrentFile.metadata.Announce)
		fmt.Println("Length:", torrentFile.metadata.Info.Length)
		fmt.Printf("Info Hash: %x\n", torrentFile.infoHash)
		fmt.Println("Piece Length:", torrentFile.metadata.Info.PieceLength)
		fmt.Println("Piece Hashes:")
		pieceHashes := getPieceHashes([]byte(torrentFile.metadata.Info.Pieces))
		for _, pieceHash := range pieceHashes {
			fmt.Println(pieceHash)
		}

	case "peers":
		torrentFile, err := readTorrentFile(os.Args[2])
		if err != nil {
			log.Fatal(err)
		}
		_, err = discoverPeers(torrentFile)
		if err != nil {
			log.Fatal(err)
		}

	case "handshake":
		torrentFile, err := readTorrentFile(os.Args[2])

		if err != nil {
			log.Fatal(err)
			return

		}
		conn, err := connectToPeer(os.Args[3])
		if err != nil {
			log.Fatal(err)
		}

		peerHandshake(conn, torrentFile.infoHash[:])

	case "download_piece":
		if len(os.Args) < 5 {
			log.Fatalf("usage: %s %s -o piece_filename torrent_file piece_num\n", os.Args[0], command)
		}

		outputPath := os.Args[3]
		torrentPath := os.Args[4]
		pieceStr := os.Args[5]
		pieceIdx, err := strconv.Atoi(pieceStr)
		if err != nil {
			log.Fatal(err)
		}

		if err != nil {
			log.Fatal(err)
		}
		torrentFile, err := readTorrentFile(torrentPath)
		if err != nil {
			log.Fatal(err)
		}

		conn, err := exchangePerrMessages(torrentFile)
		if err != nil {
			log.Fatal(err)
		}
		piece, err := downloadPiece(conn, torrentFile, pieceIdx)
		if err != nil {
			log.Fatal(err)
		}

		err = writePieceToFile(piece, outputPath)
		if err != nil {
			log.Fatal(err)
		}

	case "download":
		if len(os.Args) < 4 {
			log.Fatalf("usage: %s %s -o output_path torrent_file\n", os.Args[0], command)
		}
		outputPath := os.Args[3]
		torrentPath := os.Args[4]

		err := downloadFile(torrentPath, outputPath)
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("Unknown command: " + command)
	}
}
