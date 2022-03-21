package database

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/eso-tools/eso-tools/reader"
	"io"
)

var indexSignature = []byte{0xfb, 0xfb, 0xec, 0xec}

type Index struct {
	Signature []byte
	Field2    uint32
	Field3    uint16
	Field4    uint32
	Field5    uint32
	Field6    uint32
	Field7    uint32
	Count     uint32
	Offsets   map[uint32]uint32
}

func ParseIndex(r io.Reader) (*Index, error) {
	var data []byte
	var err error

	index := &Index{
		Offsets: map[uint32]uint32{},
	}

	data, err = reader.ReadBytes(r, len(indexSignature))
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(data, indexSignature) {
		return nil, errors.New("wrong signature")
	}
	index.Signature = data

	field2, err := reader.ReadUint32(r, binary.BigEndian)
	if err != nil {
		return nil, err
	}
	index.Field2 = field2

	field3, err := reader.ReadUint16(r, binary.BigEndian)
	if err != nil {
		return nil, err
	}
	index.Field3 = field3

	field4, err := reader.ReadUint32(r, binary.BigEndian)
	if err != nil {
		return nil, err
	}
	index.Field4 = field4

	field5, err := reader.ReadUint32(r, binary.BigEndian)
	if err != nil {
		return nil, err
	}
	index.Field5 = field5

	field6, err := reader.ReadUint32(r, binary.BigEndian)
	if err != nil {
		return nil, err
	}
	index.Field6 = field6

	field7, err := reader.ReadUint32(r, binary.BigEndian)
	if err != nil {
		return nil, err
	}
	index.Field7 = field7

	count, err := reader.ReadUint32(r, binary.BigEndian)
	if err != nil {
		return nil, err
	}
	index.Count = count

	for i := 0; i < int(index.Count); i++ {
		id, err := reader.ReadUint32(r, binary.BigEndian)
		if err != nil {
			return nil, err
		}

		offset, err := reader.ReadUint32(r, binary.BigEndian)
		if err != nil {
			return nil, err
		}

		index.Offsets[id] = offset
	}

	return index, nil
}
