package mnf

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/eso-tools/eso-tools/reader"
	"github.com/eso-tools/eso-tools/zosft"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const signature = "MES2"
const (
	block1RecordSize uint32 = 4
	block2RecordSize uint32 = 8
	block3RecordSize uint32 = 20
)

var ErrorNotValidRecord = errors.New("not valid record")

type Mnf struct {
	Path     string
	Archives map[uint16]*Archive

	Signature    string
	Version      uint16
	ArchiveCount uint16
	ArchiveIds   map[uint16]uint16
	Field5       uint32
	DataSize     uint32
	Index0       *Index0
	Index3       *Index3
}

func Parse(path string) (*Mnf, error) {
	mnf := &Mnf{
		Path:     path,
		Archives: map[uint16]*Archive{},
	}

	err := mnf.parse()
	if err != nil {
		return nil, err
	}

	return mnf, nil
}

type Index0 struct {
	Field1 []byte

	Block1Size uint32
	Block1Data []byte

	Block2Size uint32
	Block2Data []byte
}

type Index3 struct {
	Field1 []byte
	Count1 uint32
	Count2 uint32
	Count3 uint32

	UncompressedBlock1Size uint32
	CompressedBlock1Size   uint32
	Block1Records          []*Block1Record

	UncompressedBlock2Size uint32
	CompressedBlock2Size   uint32
	Block2Records          []*Block2Record

	UncompressedBlock3Size uint32
	CompressedBlock3Size   uint32
	Block3Records          []*Block3Record
}

type Block1Record struct {
	Index uint32
	Flag  uint8
}

type Block2Record struct {
	Id     uint32
	Field2 []byte
	Flags  []byte
}

type Block3Record struct {
	UncompressedSize uint32
	CompressedSize   uint32
	Hash             uint32
	Offset           uint32
	ArchiveIndex     uint16
	CompressionType  uint16
}

func (mnfData *Mnf) parse() error {
	var data []byte
	var err error

	f, err := os.Open(mnfData.Path)
	if err != nil {
		return err
	}
	defer f.Close()

	r := bufio.NewReaderSize(f, 1024*1024)

	data, err = reader.ReadBytes(r, len([]byte(signature)))
	if err != nil {
		return err
	}

	if string(data) != signature {
		return errors.New("wrong signature")
	}
	mnfData.Signature = signature

	version, err := reader.ReadUint16(r, binary.LittleEndian)
	if err != nil {
		return err
	}
	if version != 3 {
		return errors.New("not supported version")
	}
	mnfData.Version = version

	archiveCount, err := reader.ReadUint16(r, binary.LittleEndian)
	if err != nil {
		return err
	}
	mnfData.ArchiveCount = archiveCount

	archiveIds := make(map[uint16]uint16, mnfData.ArchiveCount)
	for i := uint16(0); i < mnfData.ArchiveCount; i++ {
		value, err := reader.ReadUint16(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		archiveIds[i] = value
	}
	mnfData.ArchiveIds = archiveIds

	for archiveIndex, archiveId := range mnfData.ArchiveIds {
		baseName := filepath.Base(mnfData.Path)
		archivePath := filepath.Join(filepath.Dir(mnfData.Path), fmt.Sprintf("%s%04d.dat", strings.TrimSuffix(baseName, filepath.Ext(baseName)), archiveId))
		archive, err := NewArchive(archivePath)
		if err != nil {
			return err
		}

		mnfData.Archives[archiveIndex] = archive
	}

	field5, err := reader.ReadUint32(r, binary.LittleEndian)
	if err != nil {
		return err
	}
	mnfData.Field5 = field5

	dataSize, err := reader.ReadUint32(r, binary.LittleEndian)
	if err != nil {
		return err
	}
	mnfData.DataSize = dataSize

	// indexes
	indexId, err := reader.ReadUint16(r, binary.BigEndian)
	if err != nil {
		return err
	}

	if indexId == 0 {
		index0Data := &Index0{}
		field1, err := reader.ReadBytes(r, 2)
		if err != nil {
			return err
		}
		index0Data.Field1 = field1

		block1Size, err := reader.ReadUint32(r, binary.BigEndian)
		if err != nil {
			return err
		}
		index0Data.Block1Size = block1Size

		block1Data, err := reader.ReadBytes(r, int(index0Data.Block1Size))
		if err != nil {
			return err
		}
		index0Data.Block1Data = block1Data

		block2Size, err := reader.ReadUint32(r, binary.BigEndian)
		if err != nil {
			return err
		}
		index0Data.Block2Size = block2Size

		block2Data, err := reader.ReadBytes(r, int(index0Data.Block2Size))
		if err != nil {
			return err
		}
		index0Data.Block2Data = block2Data

		mnfData.Index0 = index0Data

		indexId, err = reader.ReadUint16(r, binary.BigEndian)
		if err != nil {
			return err
		}
	}

	if indexId == 3 {
		index3Data := &Index3{
			Block1Records: []*Block1Record{},
			Block2Records: []*Block2Record{},
			Block3Records: []*Block3Record{},
		}
		field1, err := reader.ReadBytes(r, 4)
		if err != nil {
			return err
		}
		index3Data.Field1 = field1

		count1, err := reader.ReadUint32(r, binary.BigEndian)
		if err != nil {
			return err
		}
		index3Data.Count1 = count1

		count2, err := reader.ReadUint32(r, binary.BigEndian)
		if err != nil {
			return err
		}
		index3Data.Count2 = count2

		count3, err := reader.ReadUint32(r, binary.BigEndian)
		if err != nil {
			return err
		}
		index3Data.Count3 = count3

		uncompressedBlock1Size, err := reader.ReadUint32(r, binary.BigEndian)
		if err != nil {
			return err
		}
		index3Data.UncompressedBlock1Size = uncompressedBlock1Size

		compressedBlock1Size, err := reader.ReadUint32(r, binary.BigEndian)
		if err != nil {
			return err
		}
		index3Data.CompressedBlock1Size = compressedBlock1Size

		lr := io.LimitReader(r, int64(index3Data.CompressedBlock1Size))
		zr, err := zlib.NewReader(lr)
		if err != nil {
			return err
		}

		var recordSize uint32

		block1Records := []*Block1Record{}
		recordSize = block1RecordSize
		for i := uint32(0); i < index3Data.UncompressedBlock1Size/recordSize; i++ {
			recordData, err := reader.ReadBytes(zr, int(recordSize))
			if err != nil {
				return err
			}
			record := &Block1Record{
				Index: binary.LittleEndian.Uint32(recordData) & 0xffffff,
				Flag:  recordData[3:4][0],
			}

			block1Records = append(block1Records, record)
		}
		index3Data.Block1Records = block1Records
		zr.Close()

		uncompressedBlock2Size, err := reader.ReadUint32(r, binary.BigEndian)
		if err != nil {
			return err
		}
		index3Data.UncompressedBlock2Size = uncompressedBlock2Size

		compressedBlock2Size, err := reader.ReadUint32(r, binary.BigEndian)
		if err != nil {
			return err
		}
		index3Data.CompressedBlock2Size = compressedBlock2Size

		lr = io.LimitReader(r, int64(index3Data.CompressedBlock2Size))
		zr, err = zlib.NewReader(lr)
		if err != nil {
			return err
		}

		block2Records := []*Block2Record{}
		recordSize = block2RecordSize
		for i := uint32(0); i < index3Data.UncompressedBlock2Size/recordSize; i++ {
			recordData, err := reader.ReadBytes(zr, int(recordSize))
			if err != nil {
				return err
			}
			record := &Block2Record{
				Id:     binary.LittleEndian.Uint32(recordData[0:4]),
				Field2: recordData[4:6],
				Flags:  recordData[6:8],
			}

			block2Records = append(block2Records, record)
		}
		index3Data.Block2Records = block2Records
		zr.Close()

		uncompressedBlock3Size, err := reader.ReadUint32(r, binary.BigEndian)
		if err != nil {
			return err
		}
		index3Data.UncompressedBlock3Size = uncompressedBlock3Size

		compressedBlock3Size, err := reader.ReadUint32(r, binary.BigEndian)
		if err != nil {
			return err
		}
		index3Data.CompressedBlock3Size = compressedBlock3Size

		lr = io.LimitReader(r, int64(index3Data.CompressedBlock3Size))
		zr, err = zlib.NewReader(lr)
		if err != nil {
			return err
		}

		block3Records := []*Block3Record{}
		recordSize = block3RecordSize
		for i := uint32(0); i < index3Data.UncompressedBlock3Size/recordSize; i++ {
			recordData, err := reader.ReadBytes(zr, int(recordSize))
			if err != nil {
				return err
			}
			record := &Block3Record{
				UncompressedSize: binary.LittleEndian.Uint32(recordData[0:4]),
				CompressedSize:   binary.LittleEndian.Uint32(recordData[4:8]),
				Hash:             binary.LittleEndian.Uint32(recordData[8:12]),
				Offset:           binary.LittleEndian.Uint32(recordData[12:16]),
				ArchiveIndex:     binary.LittleEndian.Uint16(recordData[16:18]),
				CompressionType:  binary.LittleEndian.Uint16(recordData[18:20]),
			}
			block3Records = append(block3Records, record)
		}
		index3Data.Block3Records = block3Records
		zr.Close()

		mnfData.Index3 = index3Data
	}

	return nil
}

func (mnfData *Mnf) Read(record *Block3Record) ([]byte, error) {
	archive, ok := mnfData.Archives[record.ArchiveIndex]
	if !ok {
		return nil, fmt.Errorf("not valid archiveIndex: %d", record.ArchiveIndex)
	}

	if !archive.IsValid(record) {
		return nil, ErrorNotValidRecord
	}

	data, err := archive.Read(record)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (mnfData *Mnf) ReadRaw(record *Block3Record) ([]byte, error) {
	archive, ok := mnfData.Archives[record.ArchiveIndex]
	if !ok {
		return nil, fmt.Errorf("not valid archiveIndex: %d", record.ArchiveIndex)
	}

	if !archive.IsValid(record) {
		return nil, ErrorNotValidRecord
	}

	data, err := archive.ReadRaw(record)
	if err != nil {
		return nil, err
	}

	return data, nil
}

var zosftDepotId uint32 = 0x00ffffff
var zosftGameId uint32 = 0x00000000
var anftDepotId uint32 = 0x01000000

func (mnfData *Mnf) GetZosft() (*zosft.Zosft, error) {
	var zosftRecord *Block3Record
	for i, record := range mnfData.Index3.Block2Records {
		if (mnfData.IsDepot() && record.Id == zosftDepotId) || (mnfData.IsGame() && record.Id == zosftGameId) {
			zosftRecord = mnfData.Index3.Block3Records[i]
			break
		}
	}

	if zosftRecord == nil {
		return nil, nil
	}

	data, err := mnfData.Read(zosftRecord)
	if err != nil {
		return nil, err
	}

	zosftData, err := zosft.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	return zosftData, nil
}

func (mnfData *Mnf) IsDepot() bool {
	baseName := filepath.Base(mnfData.Path)
	return baseName == "eso.mnf"
}

func (mnfData *Mnf) IsGame() bool {
	baseName := filepath.Base(mnfData.Path)
	return baseName == "game.mnf"
}
