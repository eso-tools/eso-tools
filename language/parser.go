package language

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/eso-tools/eso-tools/reader"
	"io"
)

var signature = []byte{0x00, 0x00, 0x00, 0x02}

type Language struct {
	Signature []byte
	Count     uint32
	Records   []*Record
}

type Record struct {
	DomainId uint32
	GroupId  uint32
	Id       uint32
	Offset   uint32
	Text     string
}

func Parse(r io.Reader) (*Language, error) {
	var data []byte
	var err error

	lang := &Language{}

	data, err = reader.ReadBytes(r, 4)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(data, signature) {
		return nil, errors.New("wrong signature")
	}
	lang.Signature = data

	count, err := reader.ReadUint32(r, binary.BigEndian)
	if err != nil {
		return nil, err
	}
	lang.Count = count

	recordsByOffset := map[uint32][]*Record{}

	for i := 0; i < int(lang.Count); i++ {
		record := &Record{}

		domainId, err := reader.ReadUint32(r, binary.BigEndian)
		if err != nil {
			return nil, err
		}
		record.DomainId = domainId

		groupId, err := reader.ReadUint32(r, binary.BigEndian)
		if err != nil {
			return nil, err
		}
		record.GroupId = groupId

		id, err := reader.ReadUint32(r, binary.BigEndian)
		if err != nil {
			return nil, err
		}
		record.Id = id

		offset, err := reader.ReadUint32(r, binary.BigEndian)
		if err != nil {
			return nil, err
		}
		record.Offset = offset

		lang.Records = append(lang.Records, record)

		_, ok := recordsByOffset[record.Offset]
		if !ok {
			recordsByOffset[record.Offset] = []*Record{}
		}
		recordsByOffset[record.Offset] = append(recordsByOffset[record.Offset], record)
	}

	var currentOffset uint32
	var i uint32
Loop:
	for {
		buf := bytes.NewBuffer(nil)
		for {
			i++
			data, err := reader.ReadBytes(r, 1)
			if err != nil {
				if err == io.EOF {
					break Loop
				}
				return nil, err
			}
			if data[0] == 0x00 {
				text := buf.String()
				buf.Reset()

				for _, record := range recordsByOffset[currentOffset] {
					record.Text = text
				}

				currentOffset = i

				break
			} else {
				err = buf.WriteByte(data[0])
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return lang, nil
}
