package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"unicode"
)

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
	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
