package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

type PeerMessageId byte

const (
	Choke      PeerMessageId = 0
	Unchoke    PeerMessageId = 1
	Interested PeerMessageId = 2
	// NotInterested PeerMessageId = 3
	// Have          PeerMessageId = 4
	Bitfield PeerMessageId = 5
	Request  PeerMessageId = 6
	Piece    PeerMessageId = 7
)

func (p *PeerMessageId) String() string {
	switch *p {
	case Unchoke:
		return "Unchoke"
	case Interested:
		return "Interested"
	case Bitfield:
		return "BitField"
	case Request:
		return "Request"
	case Piece:
		return "Piece"
	default:
		return "Unknown"
	}
}

type PeerMessage struct {
	length  uint32
	id      PeerMessageId
	payload []byte
}

func createPeerMessage(id PeerMessageId) PeerMessage {
	return PeerMessage{
		length:  1,
		id:      id,
		payload: nil,
	}
}

func (msg *PeerMessage) Bytes() []byte {
	bytes := make([]byte, 5)
	binary.BigEndian.PutUint32(bytes, msg.length)
	bytes[4] = byte(msg.id)
	bytes = append(bytes, msg.payload...)
	return bytes
}

func (msg *PeerMessage) setPayload(payload []byte) {
	msg.length = uint32(len(payload) + 1)
	msg.payload = payload
}

func readPeerMessage(conn net.Conn, expected PeerMessageId) (PeerMessage, error) {
	header := make([]byte, 5)
	_, err := io.ReadFull(conn, header)
	if err != nil {
		return PeerMessage{}, nil
	}

	length := binary.BigEndian.Uint32(header[:4]) - 1
	id := PeerMessageId(header[4])
	if id != expected {
		return PeerMessage{}, fmt.Errorf("unexpected peer message id (%s != %s)", id.String(), expected.String())
	}

	payload := make([]byte, length)
	_, err = io.ReadFull(conn, payload)
	if err != nil {
		return PeerMessage{}, nil
	}
	return PeerMessage{length: length, id: id, payload: payload}, nil
}

func createPayload(pieceId uint32, i uint32, length uint32) []byte {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], pieceId)
	binary.BigEndian.PutUint32(payload[4:8], i)
	binary.BigEndian.PutUint32(payload[8:12], length)
	return payload
}
