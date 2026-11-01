package ipc

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"

	"example.edu/6cc545/cw1/internal/config"
)

const MaxMessage = 4096
const Version = 1

var ErrUnimplemented = errors.New("TODO Week 4: implement IPC")

type Message struct {
	Version   int    `json:"version"`
	RequestID string `json:"request_id"`
	NodeID    string `json:"node_id"`
	Text      string `json:"text"`
}
type Ack struct {
	Version   int    `json:"version"`
	RequestID string `json:"request_id"`
	Receiver  string `json:"receiver"`
	Accepted  bool   `json:"accepted"`
}

func (m Message) Validate() error {
	// Accept only the protocol version supported by this implementation.
	if m.Version != Version {
		return errors.New("unsupported message version")
	}

	// Ensure the request ID follows the application's ID rules.
	if !config.ValidID(m.RequestID) {
		return errors.New("invalid request ID")
	}

	// Validate the sender's node ID using the same ID rules.
	if !config.ValidID(m.NodeID) {
		return errors.New("invalid node ID")
	}

	// Reject text that is not valid UTF-8.
	if !utf8.ValidString(m.Text) {
		return errors.New("text is not valid UTF-8")
	}

	// Enforce the permitted text length in bytes, not characters.
	if len(m.Text) < 1 || len(m.Text) > 512 {
		return errors.New("text must be between 1 and 512 bytes")
	}

	// All protocol validation checks have passed.
	return nil
}
func Encode(v any) ([]byte, error) {
	b, e := json.Marshal(v)
	if e != nil {
		return nil, e
	}
	if len(b) == 0 || len(b) > MaxMessage {
		return nil, errors.New("message exceeds limit")
	}
	return b, nil
}
func Decode(data []byte, v any) error {
	// Reject empty input and JSON payloads exceeding the protocol limit.
	if len(data) == 0 || len(data) > MaxMessage {
		return errors.New("invalid message size")
	}

	// Create a decoder over the supplied bytes and reject unknown fields.
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	// Decode the first JSON value into the destination.
	if err := dec.Decode(v); err != nil {
		return err
	}

	// Require the input to contain exactly one JSON value.
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}

	// The input contains one valid JSON value with no trailing data.
	return nil
}
func WriteFrame(w io.Writer, data []byte) error {
	// Reject empty or oversized frames.
	if len(data) == 0 || len(data) > MaxMessage {
		return errors.New("invalid frame length")
	}

	// Create the four-byte big-endian length header.
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(data)))

	// Write the header and payload, handling short writes.
	for _, part := range [][]byte{header, data} {
		for len(part) > 0 {
			n, err := w.Write(part)
			if err != nil {
				return err
			}
			if n == 0 {
				return io.ErrShortWrite
			}
			part = part[n:]
		}
	}

	return nil
}
func ReadFrame(r io.Reader) ([]byte, error) {
	// Read the complete four-byte header.
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}

	// Validate the length before allocating the payload.
	length := binary.BigEndian.Uint32(header[:])
	if length == 0 || length > MaxMessage {
		return nil, errors.New("invalid frame length")
	}

	// Read the complete payload, even when reads are fragmented.
	data := make([]byte, int(length))
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}

	return data, nil
}

var _ = binary.BigEndian
var _ = bytes.NewReader
var _ = utf8.ValidString
var _ = config.ValidID
