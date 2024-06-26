package protocol

import (
	"bytes"
	"encoding/gob"
	"io"
)

// LigoloEncoder is the structure containing the writer used when encoding Envelopes
type LigoloEncoder struct {
	writer io.Writer
}

// NewEncoder encode Ligolo-ng packets
func NewEncoder(writer io.Writer) LigoloEncoder {
	return LigoloEncoder{writer: writer}
}

// Encode an Envelope packet and write the result into the writer
func (e *LigoloEncoder) Encode(envelope Envelope) error {
	var payload bytes.Buffer
	encoder := gob.NewEncoder(&payload)
	if err := encoder.Encode(envelope); err != nil {
		return err
	}
	_, err := e.writer.Write(payload.Bytes())
	if err != nil {
		return err
	}
	return nil
}
