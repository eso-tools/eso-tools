package extracter

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/eso-tools/eso-tools/mnf"
)

type Record struct {
	Record2  *mnf.Block2Record
	Record3  *mnf.Block3Record
	FileName string
	Data     []byte
}

func (record *Record) GetExtension() string {
	return GetExtension(record.Data)
}

func (record *Record) GetRawFilename() string {
	return fmt.Sprintf("0x%08x-%08x.%s", record.Record2.Id, append(record.Record2.Field2, record.Record2.Flags...), record.GetExtension())
}

var twoZeroBytes = []byte{0x00, 0x00}

func CombineRecords(mnfData *mnf.Mnf, recordChan chan *Record, errorChan chan error) {
	defer func() {
		close(recordChan)
		close(errorChan)
	}()

	zosftData, err := mnfData.GetZosft()
	if err != nil {
		errorChan <- err
		return
	}

	fileNames := map[uint32]string{}
	if zosftData != nil {
		fileNames = zosftData.GetFileNamesById()
	}

	rowCount2 := len(mnfData.Index3.Block2Records)
	rowCount3 := len(mnfData.Index3.Block3Records)
	if rowCount2 != rowCount3 {
		errorChan <- errors.New("rowCount2 != rowCount3")
		return
	}

	isDepot := mnfData.IsDepot()
	skip := isDepot

	for i := 0; i < rowCount2; i++ {
		record := &Record{
			Record2: mnfData.Index3.Block2Records[i],
			Record3: mnfData.Index3.Block3Records[i],
		}

		if isDepot && skip && record.Record3.ArchiveIndex != 0 {
			skip = false
		}

		if skip {
			continue
		}

		archive, ok := mnfData.Archives[record.Record3.ArchiveIndex]
		if !ok {
			errorChan <- fmt.Errorf("not valid archiveIndex: %d", record.Record3.ArchiveIndex)
			return
		}

		if !archive.IsValid(record.Record3) {
			continue
		}

		_, ok = fileNames[record.Record2.Id]
		if ok {
			if bytes.Equal(twoZeroBytes, record.Record2.Field2) {
				record.FileName = fileNames[record.Record2.Id]
				delete(fileNames, record.Record2.Id)
			}
		}

		recordChan <- record
	}
}

func GetExtension(data []byte) string {
	byte2 := getChunkStart(data, 2)
	switch true {
	case bytes.Equal(byte2, []byte{0xe5, 0x9b}):
		return "gr2"

	case bytes.Equal(byte2, []byte{0x1e, 0x0d}):
		return "hkt"
	}

	byte3 := getChunkStart(data, 3)
	switch true {
	case bytes.Equal(byte3, []byte("DDS")):
		return "dds"

	case bytes.Equal(byte3, []byte("XRF")):
		return "xref"
	}

	byte4 := getChunkStart(data, 4)
	switch true {
	case bytes.Equal(byte4, []byte("ANFT")):
		return "anft"

	case bytes.Equal(byte4, []byte("BKHD")):
		return "bnk"

	case bytes.Equal(byte4, []byte{0xfa, 0xfa, 0xeb, 0xeb}):
		return "db"

	case bytes.Equal(byte4, []byte{0xfb, 0xfb, 0xec, 0xec}):
		return "index"

	case bytes.Equal(byte4, []byte{0x1b, 0x4c, 0x75, 0x61}):
		return "luac"

	case bytes.Equal(byte4, []byte{0x89, 0x50, 0x4e, 0x47}):
		return "png"

	case bytes.Equal(byte4, []byte("PSB2")):
		return "psb"

	case bytes.Equal(byte4, []byte("RIFF")):
		return "wem"
	}

	byte5 := getChunkStart(data, 5)
	switch true {
	case bytes.Equal(byte5, []byte("ZOSFT")):
		return "zosft"
	}

	byte8 := getChunkStart(data, 8)
	switch true {
	case bytes.Equal(byte8, []byte{0x5f, 0x5f, 0x66, 0x66, 0x78, 0x00, 0x00, 0x01}):
		return "ffxactor"

	case bytes.Equal(byte8, []byte{0x5f, 0x5f, 0x66, 0x66, 0x78, 0x00, 0x00, 0x02}):
		return "ffxbones"

		//case bytes.Equal(byte8, []byte{0x5f, 0x5f, 0x66, 0x66, 0x78, 0x00, 0x00, 0x03}):
		//	return ""
	}

	return "dat"
}

func getChunkStart(data []byte, chunkLen int) []byte {
	dataLen := len(data)
	if chunkLen < dataLen {
		return data[0:chunkLen]
	}

	return data[:]
}

func getChunkEnd(data []byte, chunkLen int) []byte {
	dataLen := len(data)
	if chunkLen < dataLen {
		return data[dataLen-chunkLen:]
	}

	return data[:]
}
