package anft

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	gobinary "github.com/zelenin/go-binary"
	"io"
)

var signature = []byte{0x41, 0x4e, 0x46, 0x54} // ANFT

type AnftData struct {
	Signature    []byte
	Version      uint8
	Count        uint32
	Animations   []*Animation
	EndSignature []byte
}

type Animation struct {
	Field1 uint32
	Field2 uint32
	FileId uint32
	Field4 uint32
}

func Parse(r io.Reader) (*AnftData, error) {
	var (
		data []byte
		u32  uint32
		err  error
	)

	br := gobinary.NewReader(r, binary.LittleEndian)

	anftData := &AnftData{}

	data, err = br.ReadBytes(int64(len(signature)))
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(data, signature) {
		return nil, errors.New("wrong signature")
	}
	anftData.Signature = data

	u8, err := br.ReadUint8()
	if err != nil {
		return nil, err
	}
	if u8 != 0x01 {
		return nil, fmt.Errorf("unsupported version: %d", u8)
	}
	anftData.Version = u8

	u32, err = br.ReadUint32()
	if err != nil {
		return nil, err
	}
	anftData.Count = u32

	anftData.Animations = make([]*Animation, anftData.Count)

	for i := 0; i < int(anftData.Count); i++ {
		datum := &Animation{}

		u32, err = br.ReadUint32()
		if err != nil {
			return nil, err
		}
		datum.Field1 = u32

		u32, err = br.ReadUint32()
		if err != nil {
			return nil, err
		}
		datum.Field2 = u32

		u32, err = br.ReadUint32()
		if err != nil {
			return nil, err
		}
		datum.FileId = u32

		u32, err = br.ReadUint32()
		if err != nil {
			return nil, err
		}
		datum.Field4 = u32

		anftData.Animations[i] = datum
	}

	data, err = br.ReadBytes(int64(len(signature)))
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(data, signature) {
		return nil, errors.New("wrong end signature")
	}
	anftData.EndSignature = data

	return anftData, nil
}
