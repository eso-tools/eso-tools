package mnf

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/eso-tools/eso-tools/reader"
	"github.com/new-world-tools/go-oodle"
	"io"
	"os"
	"sync"
)

func NewArchive(path string) (*Archive, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return &Archive{
		file: file,
	}, nil
}

type Archive struct {
	file *os.File
	mu   sync.Mutex
}

func (archive *Archive) Close() error {
	return archive.file.Close()
}

func (archive *Archive) GetSize() int64 {
	fi, err := archive.file.Stat()
	if err != nil {
		return 0
	}

	return fi.Size()
}

func (archive *Archive) IsValid(record *Block3Record) bool {
	if (record.Offset + record.CompressedSize) > uint32(archive.GetSize()) {
		return false
	}

	return true
}

func (archive *Archive) Read(record *Block3Record) ([]byte, error) {
	data, err := archive.read(record)
	if err != nil {
		return nil, err
	}

	switch record.CompressionType {
	case 0:

	case 1: // ?
		r := bytes.NewReader(data)
		zlibReader, err := zlib.NewReader(r)
		if err != nil {
			return nil, err
		}
		defer zlibReader.Close()

		data, err = io.ReadAll(zlibReader)
		if err != nil {
			return nil, err
		}

	case 4, 8:
		data, err = oodle.Decompress(data, int64(record.UncompressedSize))
		if err != nil {
			return nil, err
		}

	default:
		return nil, errors.New(fmt.Sprintf("unsupported compressionType: %d", record.CompressionType))
	}

	if data[0] == 0x00 && len(data) >= 16 {
		cursor := uint32(0)
		u := binary.BigEndian.Uint32(data[cursor : cursor+4])
		if u == 0 {
			cursor += 4
			cursor += binary.BigEndian.Uint32(data[cursor : cursor+4])
			cursor += 4
			cursor += binary.BigEndian.Uint32(data[cursor : cursor+4])
			cursor += 4
		}
		data = data[cursor:]
	}

	return data, nil
}

func (archive *Archive) ReadRaw(record *Block3Record) ([]byte, error) {
	data, err := archive.read(record)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (archive *Archive) read(record *Block3Record) ([]byte, error) {
	archive.mu.Lock()
	defer archive.mu.Unlock()

	_, err := archive.file.Seek(int64(record.Offset), io.SeekStart)
	if err != nil {
		return nil, err
	}

	data, err := reader.ReadBytes(archive.file, int(record.CompressedSize))
	if err != nil {
		return nil, err
	}

	return data, nil
}
