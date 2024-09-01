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

const (
	signature = 0x00000002
)

type ReadStore struct {
	Signature      uint32
	Count          uint32
	Records        []*ReadRecord
	valuesByOffset map[uint32]string
	recordMap      map[uint32]map[uint32]map[uint32]*ReadRecord
}

func (store *ReadStore) GetValueByRecord(record *ReadRecord) string {
	return store.valuesByOffset[record.Offset]
}

func (store *ReadStore) GetValue(domainId uint32, groupId uint32, id uint32) string {
	var ok bool

	_, ok = store.recordMap[domainId]
	if !ok {
		return ""
	}

	_, ok = store.recordMap[domainId][groupId]
	if !ok {
		return ""
	}

	_, ok = store.recordMap[domainId][groupId][id]
	if !ok {
		return ""
	}

	return store.GetValueByRecord(store.recordMap[domainId][groupId][id])
}

func (store *ReadStore) GetDomainIds() []uint32 {
	return slices.Sorted(maps.Keys(store.recordMap))
}

func (store *ReadStore) GetGroupIds(domainId uint32) []uint32 {
	return slices.Sorted(maps.Keys(store.recordMap[domainId]))
}

func (store *ReadStore) GetRecords(domainId uint32, groupId uint32) []*ReadRecord {
	return slices.SortedFunc(maps.Values(store.recordMap[domainId][groupId]), func(a *ReadRecord, b *ReadRecord) int {
		return int(a.Id) - int(b.Id)
	})
}

type ReadRecord struct {
	DomainId uint32
	GroupId  uint32
	Id       uint32
	Offset   uint32
}

func ParseReadStore(r io.Reader) (*ReadStore, error) {
	var (
		u32   uint32
		value string
		err   error
		ok    bool
	)

	store := &ReadStore{
		valuesByOffset: map[uint32]string{},
		recordMap:      map[uint32]map[uint32]map[uint32]*ReadRecord{},
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

	store.Records = make([]*ReadRecord, 0, store.Count)

	for i := 0; i < int(store.Count); i++ {
		record := &ReadRecord{}

		u32, err = reader.ReadUint32(buf, binary.BigEndian)
		if err != nil {
			return nil, err
		}
		record.DomainId = u32

		u32, err = reader.ReadUint32(buf, binary.BigEndian)
		if err != nil {
			return nil, err
		}
		record.GroupId = u32

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
			store.recordMap[record.DomainId] = make(map[uint32]map[uint32]*ReadRecord)
		}

		_, ok = store.recordMap[record.DomainId][record.GroupId]
		if !ok {
			store.recordMap[record.DomainId][record.GroupId] = make(map[uint32]*ReadRecord)
		}

		store.recordMap[record.DomainId][record.GroupId][record.Id] = record
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

		store.valuesByOffset[currentOffset] = value

		currentOffset += uint32(len(value) + 1)
	}

	return store, nil
}
