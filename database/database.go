package database

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"github.com/eso-tools/eso-tools/reader"
	"io"
)

var databaseSignature = []byte{0xfa, 0xfa, 0xeb, 0xeb}

type Database struct {
	Signature []byte
	Field2    uint32
	Count     uint32
	Version   uint32
	Records   []*Record
}

type Record struct {
	UncomressedSize1 uint32
	UncomressedSize2 uint32
	ComressedSize    uint32
	Id               uint32
	NameSize         uint16
	Name             string
	Data             []byte
}

type RecordMeta struct {
	Field1   uint8
	Field2   uint32
	Field3   uint32
	Field4   uint16
	Field5   uint32
	DateTime uint32
}

func ParseDatabase(r io.Reader) (*Database, error) {
	var data []byte
	var err error

	db := &Database{
		Records: []*Record{},
	}

	data, err = reader.ReadBytes(r, len(databaseSignature))
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(data, databaseSignature) {
		return nil, errors.New("wrong signature")
	}
	db.Signature = data

	field2, err := reader.ReadUint32(r, binary.BigEndian)
	if err != nil {
		return nil, err
	}
	db.Field2 = field2

	field3, err := reader.ReadUint32(r, binary.BigEndian)
	if err != nil {
		return nil, err
	}
	db.Count = field3

	field4, err := reader.ReadUint32(r, binary.BigEndian)
	if err != nil {
		return nil, err
	}
	db.Version = field4

	for i := 0; i < int(db.Count); i++ {
		record := &Record{}

		uncomressedSize1, err := reader.ReadUint32(r, binary.BigEndian)
		if err != nil {
			return nil, err
		}
		record.UncomressedSize1 = uncomressedSize1

		uncomressedSize2, err := reader.ReadUint32(r, binary.BigEndian)
		if err != nil {
			return nil, err
		}
		record.UncomressedSize2 = uncomressedSize2

		dataSize, err := reader.ReadUint32(r, binary.BigEndian)
		if err != nil {
			return nil, err
		}
		record.ComressedSize = dataSize

		lr := io.LimitReader(r, int64(record.ComressedSize))
		zr, err := zlib.NewReader(lr)
		if err != nil {
			return nil, err
		}

		id, err := reader.ReadUint32(zr, binary.BigEndian)
		if err != nil {
			return nil, err
		}
		record.Id = id

		nameSize, err := reader.ReadUint16(zr, binary.BigEndian)
		if err != nil {
			return nil, err
		}
		record.NameSize = nameSize

		data, err = reader.ReadBytes(zr, int(record.NameSize))
		if err != nil {
			return nil, err
		}
		record.Name = string(data)

		data, err = io.ReadAll(zr)
		if err != nil {
			return nil, err
		}
		record.Data = data

		db.Records = append(db.Records, record)

		err = zr.Close()
		if err != nil {
			return nil, err
		}
	}

	return db, nil
}
