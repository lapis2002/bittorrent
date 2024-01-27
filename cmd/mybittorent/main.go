package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"unicode"
)

// The 'bencode' tag in PieceLength field is due to
// the fact that Go uses a capitalised version of the key by default,
// so we need to explicitly tell it to look for 'piece length'.
type TorrentFile struct {
	Announce string
	Info     struct {
		Length      int
		Name        string
		PieceLength int `bencode:"piece length"`
		Pieces      string
	}
}

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
func readTorrentFile(filename string) (TorrentFile, [20]byte, error) {
	var metadata TorrentFile
	buffer, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println(err)
		return metadata, [20]byte{}, err
	}
	s := string(buffer)
	decoded, err := decodeBencode(s)

	if err != nil {
		fmt.Println(err)
		return metadata, [20]byte{}, err
	}
	decodedMap := decoded.(map[string]interface{})

	metadata.Announce = decodedMap["announce"].(string)
	infoMap := decodedMap["info"].(map[string]interface{})
	metadata.Info.Length = int(infoMap["length"].(int))
	metadata.Info.Name = infoMap["name"].(string)
	metadata.Info.PieceLength = int(infoMap["piece length"].(int))
	metadata.Info.Pieces = infoMap["pieces"].(string)

	infoMapEncoded, err := encodeBencodeDict(infoMap)
	if err != nil {
		fmt.Println(err)
		return metadata, [20]byte{}, err
	}
	infoHash := sha1.Sum([]byte(infoMapEncoded))

	return metadata, infoHash, nil
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
		metadata, infoHash, err := readTorrentFile(os.Args[2])
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Tracker URL:", metadata.Announce)
		fmt.Println("Length:", metadata.Info.Length)
		fmt.Printf("Info Hash: %x\n", infoHash)
		fmt.Println("Piece Length:", metadata.Info.PieceLength)
		fmt.Println("Piece Hashes:")
		printPieceHashes([]byte(metadata.Info.Pieces))

	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
