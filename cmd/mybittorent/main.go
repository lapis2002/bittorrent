package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

var PEER_ID = "00112233445566778899"
var CLIENT_PORT = "6881"

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
		fmt.Println(err)
		return torrentFile, err
	}
	s := string(buffer)
	decoded, err := decodeBencode(s)

	if err != nil {
		fmt.Println(err)
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
		fmt.Println(err)
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

func printPieceHashes(pieces []byte) {
	for i := 0; i < len(pieces); i += 20 {
		hash := pieces[i : i+20]
		fmt.Println(hex.EncodeToString(hash))
	}
}

func discoverPeers(torrentFile TorrentFile) {
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
	}
	defer response.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)
	respBytes := buf.String()
	respString := string(respBytes)

	decoded, err := decodeBencode(respString)
	if err != nil {
		fmt.Println(err)
		return
	}

	decodedMap, ok := decoded.(map[string]interface{})
	if !ok {
		fmt.Println("Unexpected decoded value type.")
	}

	interval, ok := decodedMap["interval"].(int)
	if !ok {
		fmt.Println("Unexpected type in `interval`")
	}

	peers, ok := decodedMap["peers"].(string)
	if !ok {
		fmt.Println("Unexpected type in `peers`")
	}

	trackerResp := TrackerResponse{
		Interval: interval,
		Peers:    peers,
	}

	for i := 0; i <= len(trackerResp.Peers)-6; i += 6 {
		peer := []byte(trackerResp.Peers[i : i+4])
		port := binary.BigEndian.Uint16([]byte(trackerResp.Peers[i+4 : i+6]))
		fmt.Printf("%v.%v.%v.%v:%d\n", peer[0], peer[1], peer[2], peer[3], port)
	}
}

func main() {
	command := os.Args[1]

	if command == "decode" {
		bencodedValue := os.Args[2]

		decoded, err := decodeBencode(bencodedValue)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))
	} else if command == "info" {
		torrentFile, err := readTorrentFile(os.Args[2])
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Tracker URL:", torrentFile.metadata.Announce)
		fmt.Println("Length:", torrentFile.metadata.Info.Length)
		fmt.Printf("Info Hash: %x\n", torrentFile.infoHash)
		fmt.Println("Piece Length:", torrentFile.metadata.Info.PieceLength)
		fmt.Println("Piece Hashes:")
		printPieceHashes([]byte(torrentFile.metadata.Info.Pieces))

	} else if command == "peers" {
		torrentFile, err := readTorrentFile(os.Args[2])

		if err != nil {
			fmt.Println(err)
			return

		}
		discoverPeers(torrentFile)
	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
