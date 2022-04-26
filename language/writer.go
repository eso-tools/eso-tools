package language

import (
	"encoding/binary"
	"io"
)

func Write(w io.Writer, lang *Language) error {
	lang.Signature = signature
	lang.Count = uint32(len(lang.Records))

	_, err := w.Write(lang.Signature)
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.BigEndian, lang.Count)
	if err != nil {
		return err
	}

	textOffsets := map[string]uint32{}

	var currentOffset uint32
	for _, record := range lang.Records {
		_, ok := textOffsets[record.Text]
		if !ok {
			textOffsets[record.Text] = currentOffset
			currentOffset = currentOffset + uint32(len(record.Text)) + 1
		}

		record.Offset = textOffsets[record.Text]

		err = binary.Write(w, binary.BigEndian, record.DomainId)
		if err != nil {
			return err
		}

		err = binary.Write(w, binary.BigEndian, record.GroupId)
		if err != nil {
			return err
		}

		err = binary.Write(w, binary.BigEndian, record.Id)
		if err != nil {
			return err
		}

		err = binary.Write(w, binary.BigEndian, record.Offset)
		if err != nil {
			return err
		}
	}

	nilByte := uint8(0)

	for _, record := range lang.Records {
		_, ok := textOffsets[record.Text]
		if ok {
			_, err = w.Write([]byte(record.Text))
			if err != nil {
				return err
			}
			err = binary.Write(w, binary.BigEndian, nilByte)
			if err != nil {
				return err
			}

			delete(textOffsets, record.Text)
		}
	}

	return nil
}
