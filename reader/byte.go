package reader

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

func ReadBytes(r io.Reader, size int) ([]byte, error) {
	buf := make([]byte, size)

	_, err := io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func ReadUint8(r io.Reader) (uint8, error) {
	b, err := ReadBytes(r, 1)
	if err != nil {
		return 0, err
	}

	return b[0], nil
}

func ReadUint16(r io.Reader, byteOrder binary.ByteOrder) (uint16, error) {
	b, err := ReadBytes(r, 2)
	if err != nil {
		return 0, err
	}

	return byteOrder.Uint16(b), nil
}

func ReadUint32(r io.Reader, byteOrder binary.ByteOrder) (uint32, error) {
	b, err := ReadBytes(r, 4)
	if err != nil {
		return 0, err
	}

	return byteOrder.Uint32(b), nil
}

func ReadUint64(r io.Reader, byteOrder binary.ByteOrder) (uint64, error) {
	b, err := ReadBytes(r, 8)
	if err != nil {
		return 0, err
	}

	return byteOrder.Uint64(b), nil
}

func ReadNullTerminatedString(r io.Reader) (string, error) {
	var buf bytes.Buffer
	b := make([]byte, 1)
	for {
		n, err := r.Read(b)
		if err != nil {
			if err == io.EOF && buf.Len() > 0 {
				return "", errors.New("null byte not found before EOF")
			}
			return "", err
		}

		if n == 0 {
			continue
		}

		if b[0] == 0x00 {
			break
		}

		buf.WriteByte(b[0])
	}

	return buf.String(), nil
}
