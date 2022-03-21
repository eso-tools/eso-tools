package zosft

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"github.com/eso-tools/eso-tools/reader"
	"io"
)

const signature = "ZOSFT"

type Zosft struct {
	Signature string
	Field2    []byte
	Count     uint32

	Index1 *Index1
	Index2 *Index2
	Index3 *Index3

	DataSize uint32

	OffsetFileName map[uint32]string
}

func (zosftData *Zosft) GetFileNamesById() map[uint32]string {
	fileNames := map[uint32]string{}

	for i := uint32(0); i < zosftData.Count; i++ {
		block22Record := zosftData.Index2.Block2Records[i]
		block23Record := zosftData.Index2.Block3Records[i]

		fileNames[block22Record.Id] = zosftData.OffsetFileName[block23Record.Offset]
	}

	return fileNames
}

type Index1 struct {
	Id     uint16
	Field2 uint32
	Count1 uint32
	Count2 uint32
	Count3 uint32

	UncompressedBlock1Size uint32
	CompressedBlock1Size   uint32
	Block1Records          []*Index1Block1Record

	UncompressedBlock2Size uint32
	CompressedBlock2Size   uint32
	Block2Records          []*Index1Block2Record

	UncompressedBlock3Size uint32
	CompressedBlock3Size   uint32
	Block3Records          []*Index1Block3Record
}

type Index2 struct {
	Id     uint16
	Field2 uint32
	Count1 uint32
	Count2 uint32
	Count3 uint32

	UncompressedBlock1Size uint32
	CompressedBlock1Size   uint32
	Block1Records          []*Index2Block1Record

	UncompressedBlock2Size uint32
	CompressedBlock2Size   uint32
	Block2Records          []*Index2Block2Record

	UncompressedBlock3Size uint32
	CompressedBlock3Size   uint32
	Block3Records          []*Index2Block3Record
}

type Index3 struct {
	Id     uint16
	Field2 uint32
	Count1 uint32
	Count2 uint32
	Count3 uint32

	UncompressedBlock1Size uint32
	CompressedBlock1Size   uint32
	Block1Records          []*Index3Block1Record

	UncompressedBlock2Size uint32
	CompressedBlock2Size   uint32
	Block2Records          []*Index3Block2Record

	UncompressedBlock3Size uint32
	CompressedBlock3Size   uint32
	Block3Records          []*Index3Block3Record
}

func Parse(r io.Reader) (*Zosft, error) {
	var data []byte
	var err error

	zosftData := &Zosft{
		OffsetFileName: map[uint32]string{},
	}

	data, err = reader.ReadBytes(r, len([]byte(signature)))
	if err != nil {
		return nil, err
	}

	if string(data) != signature {
		return nil, errors.New("wrong signature")
	}
	zosftData.Signature = signature

	data, err = reader.ReadBytes(r, 10)
	if err != nil {
		return nil, err
	}
	zosftData.Field2 = data

	count, err := reader.ReadUint32(r, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	zosftData.Count = count

	// indexes
	index1Data, err := parseIndex1(r)
	if err != nil {
		return nil, err
	}
	zosftData.Index1 = index1Data

	index2Data, err := parseIndex2(r)
	if err != nil {
		return nil, err
	}
	zosftData.Index2 = index2Data

	index3Data, err := parseIndex3(r)
	if err != nil {
		return nil, err
	}
	zosftData.Index3 = index3Data

	dataSize, err := reader.ReadUint32(r, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	zosftData.DataSize = dataSize

	lr := io.LimitReader(r, int64(zosftData.DataSize))

	fileNameBuf := bytes.NewBuffer(nil)
	var offset uint32
	var i uint32
	for {
		data, err := reader.ReadBytes(lr, 1)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		b := data[0]

		if b == 0x00 {
			zosftData.OffsetFileName[offset] = fileNameBuf.String()
			fileNameBuf.Reset()
			offset = i + 1
		} else {
			fileNameBuf.WriteByte(b)
		}

		i++
	}

	data, err = reader.ReadBytes(r, len([]byte(signature)))
	if err != nil {
		return nil, err
	}

	if string(data) != signature {
		return nil, errors.New("wrong signature")
	}

	return zosftData, nil
}

type Index1Block1Record struct {
	Index11 uint32
	Flag    uint8
}

type Index1Block2Record struct {
	Field1 []byte
}

type Index1Block3Record struct {
	Id uint32
}

func parseIndex1(r io.Reader) (*Index1, error) {
	indexData := &Index1{}

	id, err := reader.ReadUint16(r, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	indexData.Id = id

	field2, err := reader.ReadUint32(r, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	indexData.Field2 = field2

	count1, err := reader.ReadUint32(r, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	indexData.Count1 = count1

	count2, err := reader.ReadUint32(r, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	indexData.Count2 = count2

	count3, err := reader.ReadUint32(r, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	indexData.Count3 = count3

	if indexData.Count1 > 0 {
		uncompressedBlock1Size, err := reader.ReadUint32(r, binary.LittleEndian)
		if err != nil {
			return nil, err
		}
		indexData.UncompressedBlock1Size = uncompressedBlock1Size

		compressedBlock1Size, err := reader.ReadUint32(r, binary.LittleEndian)
		if err != nil {
			return nil, err
		}
		indexData.CompressedBlock1Size = compressedBlock1Size

		lr := io.LimitReader(r, int64(indexData.CompressedBlock1Size))
		zr, err := zlib.NewReader(lr)
		if err != nil {
			return nil, err
		}
		defer zr.Close()

		indexData.Block1Records = []*Index1Block1Record{}
		recordSize := int(indexData.UncompressedBlock1Size / indexData.Count1)
		for i := uint32(0); i < indexData.Count1; i++ {
			recordData, err := reader.ReadBytes(zr, recordSize)
			if err != nil {
				return nil, err
			}
			indexData.Block1Records = append(indexData.Block1Records, &Index1Block1Record{
				Index11: binary.LittleEndian.Uint32(recordData) & 0xffffff,
				Flag:    recordData[3],
			})
		}
	}

	if indexData.Count2 > 0 {
		uncompressedBlock2Size, err := reader.ReadUint32(r, binary.LittleEndian)
		if err != nil {
			return nil, err
		}
		indexData.UncompressedBlock2Size = uncompressedBlock2Size

		compressedBlock2Size, err := reader.ReadUint32(r, binary.LittleEndian)
		if err != nil {
			return nil, err
		}
		indexData.CompressedBlock2Size = compressedBlock2Size

		lr := io.LimitReader(r, int64(indexData.CompressedBlock2Size))
		zr, err := zlib.NewReader(lr)
		if err != nil {
			return nil, err
		}
		defer zr.Close()

		indexData.Block2Records = []*Index1Block2Record{}
		recordSize := int(indexData.UncompressedBlock2Size / indexData.Count2)
		for i := uint32(0); i < indexData.Count2; i++ {
			recordData, err := reader.ReadBytes(zr, recordSize)
			if err != nil {
				return nil, err
			}
			indexData.Block2Records = append(indexData.Block2Records, &Index1Block2Record{
				Field1: recordData,
			})
		}
	}

	if indexData.Count3 > 0 {
		uncompressedBlock3Size, err := reader.ReadUint32(r, binary.LittleEndian)
		if err != nil {
			return nil, err
		}
		indexData.UncompressedBlock3Size = uncompressedBlock3Size

		compressedBlock3Size, err := reader.ReadUint32(r, binary.LittleEndian)
		if err != nil {
			return nil, err
		}
		indexData.CompressedBlock3Size = compressedBlock3Size

		lr := io.LimitReader(r, int64(indexData.CompressedBlock3Size))
		zr, err := zlib.NewReader(lr)
		if err != nil {
			return nil, err
		}
		defer zr.Close()

		indexData.Block3Records = []*Index1Block3Record{}
		recordSize := int(indexData.UncompressedBlock3Size / indexData.Count3)
		for i := uint32(0); i < indexData.Count3; i++ {
			recordData, err := reader.ReadBytes(zr, recordSize)
			if err != nil {
				return nil, err
			}
			indexData.Block3Records = append(indexData.Block3Records, &Index1Block3Record{
				Id: binary.LittleEndian.Uint32(recordData),
			})
		}
	}

	return indexData, nil
}

type Index2Block1Record struct {
	Index21 uint32
	Flag    uint8
}

type Index2Block2Record struct {
	Id uint32
}

type Index2Block3Record struct {
	Id     uint32
	Offset uint32
	Field3 []byte
}

func parseIndex2(r io.Reader) (*Index2, error) {
	indexData := &Index2{}

	id, err := reader.ReadUint16(r, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	indexData.Id = id

	field2, err := reader.ReadUint32(r, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	indexData.Field2 = field2

	count1, err := reader.ReadUint32(r, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	indexData.Count1 = count1

	count2, err := reader.ReadUint32(r, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	indexData.Count2 = count2

	count3, err := reader.ReadUint32(r, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	indexData.Count3 = count3

	if indexData.Count1 > 0 {
		uncompressedBlock1Size, err := reader.ReadUint32(r, binary.LittleEndian)
		if err != nil {
			return nil, err
		}
		indexData.UncompressedBlock1Size = uncompressedBlock1Size

		compressedBlock1Size, err := reader.ReadUint32(r, binary.LittleEndian)
		if err != nil {
			return nil, err
		}
		indexData.CompressedBlock1Size = compressedBlock1Size

		lr := io.LimitReader(r, int64(indexData.CompressedBlock1Size))
		zr, err := zlib.NewReader(lr)
		if err != nil {
			return nil, err
		}
		defer zr.Close()

		indexData.Block1Records = []*Index2Block1Record{}
		recordSize := int(indexData.UncompressedBlock1Size / indexData.Count1)
		for i := uint32(0); i < indexData.Count1; i++ {
			recordData, err := reader.ReadBytes(zr, recordSize)
			if err != nil {
				return nil, err
			}
			indexData.Block1Records = append(indexData.Block1Records, &Index2Block1Record{
				Index21: binary.LittleEndian.Uint32(recordData) & 0xffffff,
				Flag:    recordData[3],
			})
		}
	}

	if indexData.Count2 > 0 {
		uncompressedBlock2Size, err := reader.ReadUint32(r, binary.LittleEndian)
		if err != nil {
			return nil, err
		}
		indexData.UncompressedBlock2Size = uncompressedBlock2Size

		compressedBlock2Size, err := reader.ReadUint32(r, binary.LittleEndian)
		if err != nil {
			return nil, err
		}
		indexData.CompressedBlock2Size = compressedBlock2Size

		lr := io.LimitReader(r, int64(indexData.CompressedBlock2Size))
		zr, err := zlib.NewReader(lr)
		if err != nil {
			return nil, err
		}
		defer zr.Close()

		indexData.Block2Records = []*Index2Block2Record{}
		recordSize := int(indexData.UncompressedBlock2Size / indexData.Count2)
		for i := uint32(0); i < indexData.Count2; i++ {
			recordData, err := reader.ReadBytes(zr, recordSize)
			if err != nil {
				return nil, err
			}
			indexData.Block2Records = append(indexData.Block2Records, &Index2Block2Record{
				Id: binary.LittleEndian.Uint32(recordData),
			})
		}
	}

	if indexData.Count3 > 0 {
		uncompressedBlock3Size, err := reader.ReadUint32(r, binary.LittleEndian)
		if err != nil {
			return nil, err
		}
		indexData.UncompressedBlock3Size = uncompressedBlock3Size

		compressedBlock3Size, err := reader.ReadUint32(r, binary.LittleEndian)
		if err != nil {
			return nil, err
		}
		indexData.CompressedBlock3Size = compressedBlock3Size

		lr := io.LimitReader(r, int64(indexData.CompressedBlock3Size))
		zr, err := zlib.NewReader(lr)
		if err != nil {
			return nil, err
		}
		defer zr.Close()

		indexData.Block3Records = []*Index2Block3Record{}
		recordSize := int(indexData.UncompressedBlock3Size / indexData.Count3)
		for i := uint32(0); i < indexData.Count3; i++ {
			recordData, err := reader.ReadBytes(zr, recordSize)
			if err != nil {
				return nil, err
			}
			indexData.Block3Records = append(indexData.Block3Records, &Index2Block3Record{
				Id:     binary.LittleEndian.Uint32(recordData[0:4]),
				Offset: binary.LittleEndian.Uint32(recordData[4:8]),
				Field3: recordData[8:16],
			})
		}
	}

	return indexData, nil
}

type Index3Block1Record struct {
	Field1 uint32
	Flag   uint8
}

type Index3Block2Record struct {
	Field1 []byte
}

type Index3Block3Record struct {
	Field1 []byte
}

func parseIndex3(r io.Reader) (*Index3, error) {
	indexData := &Index3{}

	id, err := reader.ReadUint16(r, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	indexData.Id = id

	field2, err := reader.ReadUint32(r, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	indexData.Field2 = field2

	count1, err := reader.ReadUint32(r, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	indexData.Count1 = count1

	count2, err := reader.ReadUint32(r, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	indexData.Count2 = count2

	count3, err := reader.ReadUint32(r, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	indexData.Count3 = count3

	if indexData.Count1 > 0 {
		uncompressedBlock1Size, err := reader.ReadUint32(r, binary.LittleEndian)
		if err != nil {
			return nil, err
		}
		indexData.UncompressedBlock1Size = uncompressedBlock1Size

		compressedBlock1Size, err := reader.ReadUint32(r, binary.LittleEndian)
		if err != nil {
			return nil, err
		}
		indexData.CompressedBlock1Size = compressedBlock1Size

		lr := io.LimitReader(r, int64(indexData.CompressedBlock1Size))
		zr, err := zlib.NewReader(lr)
		if err != nil {
			return nil, err
		}
		defer zr.Close()

		indexData.Block1Records = []*Index3Block1Record{}
		recordSize := int(indexData.UncompressedBlock1Size / indexData.Count1)
		for i := uint32(0); i < indexData.Count1; i++ {
			recordData, err := reader.ReadBytes(zr, recordSize)
			if err != nil {
				return nil, err
			}
			indexData.Block1Records = append(indexData.Block1Records, &Index3Block1Record{
				Field1: binary.LittleEndian.Uint32(recordData) & 0xffffff,
				Flag:   recordData[3],
			})
		}
	}

	if indexData.Count2 > 0 {
		uncompressedBlock2Size, err := reader.ReadUint32(r, binary.LittleEndian)
		if err != nil {
			return nil, err
		}
		indexData.UncompressedBlock2Size = uncompressedBlock2Size

		compressedBlock2Size, err := reader.ReadUint32(r, binary.LittleEndian)
		if err != nil {
			return nil, err
		}
		indexData.CompressedBlock2Size = compressedBlock2Size

		lr := io.LimitReader(r, int64(indexData.CompressedBlock2Size))
		zr, err := zlib.NewReader(lr)
		if err != nil {
			return nil, err
		}
		defer zr.Close()

		indexData.Block2Records = []*Index3Block2Record{}
		recordSize := int(indexData.UncompressedBlock2Size / indexData.Count2)
		for i := uint32(0); i < indexData.Count2; i++ {
			recordData, err := reader.ReadBytes(zr, recordSize)
			if err != nil {
				return nil, err
			}
			indexData.Block2Records = append(indexData.Block2Records, &Index3Block2Record{
				Field1: recordData,
			})
		}
	}

	if indexData.Count3 > 0 {
		uncompressedBlock3Size, err := reader.ReadUint32(r, binary.LittleEndian)
		if err != nil {
			return nil, err
		}
		indexData.UncompressedBlock3Size = uncompressedBlock3Size

		compressedBlock3Size, err := reader.ReadUint32(r, binary.LittleEndian)
		if err != nil {
			return nil, err
		}
		indexData.CompressedBlock3Size = compressedBlock3Size

		lr := io.LimitReader(r, int64(indexData.CompressedBlock3Size))
		zr, err := zlib.NewReader(lr)
		if err != nil {
			return nil, err
		}
		defer zr.Close()

		indexData.Block3Records = []*Index3Block3Record{}
		recordSize := int(indexData.UncompressedBlock3Size / indexData.Count3)
		for i := uint32(0); i < indexData.Count3; i++ {
			recordData, err := reader.ReadBytes(zr, recordSize)
			if err != nil {
				return nil, err
			}
			indexData.Block3Records = append(indexData.Block3Records, &Index3Block3Record{
				Field1: recordData,
			})
		}
	}

	return indexData, nil
}
