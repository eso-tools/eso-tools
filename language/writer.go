package language

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/eso-tools/eso-tools/reader"
	"io"
	"maps"
	"slices"
)

type WriteStore struct {
	Signature uint32
	Count     uint32
	Records   []*WriteRecord
	recordMap map[uint32]map[uint32]map[uint32]*WriteRecord
}

func (store *WriteStore) GetValue(domainId uint32, id uint32, variant uint32) string {
	var ok bool

	_, ok = store.recordMap[domainId]
	if !ok {
		return ""
	}

	_, ok = store.recordMap[domainId][id]
	if !ok {
		return ""
	}

	_, ok = store.recordMap[domainId][id][variant]
	if !ok {
		return ""
	}

	return store.recordMap[domainId][id][variant].Value
}

func (store *WriteStore) GetDomainIds() []uint32 {
	return slices.Sorted(maps.Keys(store.recordMap))
}

func (store *WriteStore) GetIds(domainId uint32) []uint32 {
	return slices.Sorted(maps.Keys(store.recordMap[domainId]))
}

func (store *WriteStore) GetRecords(domainId uint32, id uint32) []*WriteRecord {
	return slices.SortedFunc(maps.Values(store.recordMap[domainId][id]), func(a *WriteRecord, b *WriteRecord) int {
		if a.Id == b.Id {
			return int(a.Variant) - int(b.Variant)
		}
		return int(a.Id) - int(b.Id)
	})
}

type WriteRecord struct {
	DomainId uint32
	Variant  uint32
	Id       uint32
	Offset   uint32
	Value    string
}

func ParseWriteStore(r io.Reader) (*WriteStore, error) {
	var (
		u32   uint32
		value string
		err   error
		ok    bool
	)

	store := &WriteStore{
		recordMap: map[uint32]map[uint32]map[uint32]*WriteRecord{},
	}

	buf := bufio.NewReaderSize(r, 1024*1024)

	u32, err = reader.ReadUint32(buf, binary.BigEndian)
	if err != nil {
		return nil, err
	}

	if u32 != signature {
		return nil, fmt.Errorf("wrong signature: %d", u32)
	}
	store.Signature = u32

	u32, err = reader.ReadUint32(buf, binary.BigEndian)
	if err != nil {
		return nil, err
	}
	store.Count = u32

	store.Records = make([]*WriteRecord, 0, store.Count)
	recordsByOffset := map[uint32][]*WriteRecord{}

	for i := 0; i < int(store.Count); i++ {
		record := &WriteRecord{}

		u32, err = reader.ReadUint32(buf, binary.BigEndian)
		if err != nil {
			return nil, err
		}
		record.DomainId = u32

		u32, err = reader.ReadUint32(buf, binary.BigEndian)
		if err != nil {
			return nil, err
		}
		record.Variant = u32

		u32, err = reader.ReadUint32(buf, binary.BigEndian)
		if err != nil {
			return nil, err
		}
		record.Id = u32

		u32, err = reader.ReadUint32(buf, binary.BigEndian)
		if err != nil {
			return nil, err
		}
		record.Offset = u32

		store.Records = append(store.Records, record)

		_, ok = store.recordMap[record.DomainId]
		if !ok {
			store.recordMap[record.DomainId] = make(map[uint32]map[uint32]*WriteRecord)
		}

		_, ok = store.recordMap[record.DomainId][record.Id]
		if !ok {
			store.recordMap[record.DomainId][record.Id] = make(map[uint32]*WriteRecord)
		}

		store.recordMap[record.DomainId][record.Id][record.Variant] = record

		_, ok = recordsByOffset[record.Offset]
		if !ok {
			recordsByOffset[record.Offset] = []*WriteRecord{}
		}

		recordsByOffset[record.Offset] = append(recordsByOffset[record.Offset], record)
	}

	var currentOffset uint32

	for {
		value, err = reader.ReadNullTerminatedString(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		for _, record := range recordsByOffset[currentOffset] {
			record.Value = value
		}

		currentOffset += uint32(len(value) + 1)
	}

	return store, nil
}

func WriteWriteStore(w io.Writer, store *WriteStore) error {
	store.Signature = signature
	store.Count = uint32(len(store.Records))

	buf := bufio.NewWriterSize(w, 1024*1024)
	w = buf

	var (
		err error
	)

	err = binary.Write(w, binary.BigEndian, store.Signature)
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.BigEndian, store.Count)
	if err != nil {
		return err
	}

	textOffsets := map[string]uint32{}

	var currentOffset uint32
	for _, record := range store.Records {
		_, ok := textOffsets[record.Value]
		if !ok {
			textOffsets[record.Value] = currentOffset
			currentOffset += uint32(len(record.Value)) + 1
		}

		record.Offset = textOffsets[record.Value]

		err = binary.Write(w, binary.BigEndian, record.DomainId)
		if err != nil {
			return err
		}

		err = binary.Write(w, binary.BigEndian, record.Variant)
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

	for _, record := range store.Records {
		_, ok := textOffsets[record.Value]
		if !ok {
			continue
		}

		_, err = w.Write([]byte(record.Value))
		if err != nil {
			return err
		}
		err = binary.Write(w, binary.BigEndian, nilByte)
		if err != nil {
			return err
		}

		delete(textOffsets, record.Value)
	}

	err = buf.Flush()
	if err != nil {
		return err
	}

	return nil
}
