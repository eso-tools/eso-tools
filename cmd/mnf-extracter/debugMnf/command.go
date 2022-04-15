package debugMnf

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"github.com/eso-tools/eso-tools/extracter"
	"github.com/eso-tools/eso-tools/format"
	"github.com/eso-tools/eso-tools/mnf"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
	"path/filepath"
)

type Config struct {
	Input  string `long:"input" short:"i" required:"true"`
	Output string `long:"output" short:"o" required:"true"`
}

func Command(ctx context.Context, args []string) error {
	var config Config
	_, err := flags.ParseArgs(&config, args[1:])
	if err != nil {
		return nil
	}

	inputFilePath, err := filepath.Abs(filepath.Clean(config.Input))
	if err != nil {
		return fmt.Errorf("filepath.Abs: %s", err)
	}

	inputFileInfo, err := os.Stat(inputFilePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("'%s' does not exist", inputFilePath)
	}

	if inputFileInfo.IsDir() {
		return fmt.Errorf("'%s' is not a file", inputFilePath)
	}

	mnfData, err := mnf.Parse(inputFilePath)
	if err != nil {
		return fmt.Errorf("mnf.Parse: %s", err)
	}

	if len(mnfData.Index3.Block2Records) != len(mnfData.Index3.Block3Records) {
		return fmt.Errorf("len(mnfData.Index3.Block2Records) != len(mnfData.Index3.Block3Records)")
	}

	f, err := os.Create(config.Output)
	if err != nil {
		return fmt.Errorf("os.Create: %s", err)
	}
	defer f.Close()

	zosftData, err := mnfData.GetZosft()
	if err != nil {
		return fmt.Errorf("mnfData.GetZosft: %s", err)
	}

	fileNames := map[uint32]string{}
	if zosftData != nil {
		fileNames = zosftData.GetFileNamesById()
	}

	twoZeroBytes := []byte{0x00, 0x00}

	log.Printf("Writing \"%s\"...", config.Output)

	csvWriter := csv.NewWriter(f)
	//csvReader.Comma

	csvWriter.Write([]string{
		"Index",

		"",

		"Id",
		"ItemId",
		"Flags",

		"",

		"UncompressedSize",
		"CompressedSize",
		"Hash",
		"Offset",
		"NextOffset",
		"ArchiveIndex",
		"ArchiveBasedIndex",
		"UniqueId",
		"CompressionType",

		"",

		"Filename",
		"Ext",
		"Byte10",
	})

	indexes := map[uint16]int{}
	for i := uint16(0); i < mnfData.ArchiveCount; i++ {
		indexes[i] = 0
	}

	uniqueIds := map[uint16]map[string]int64{}
	for i := uint16(0); i < mnfData.ArchiveCount; i++ {
		uniqueIds[i] = map[string]int64{}
	}

	for i, _ := range mnfData.Index3.Block2Records {
		block2Record := mnfData.Index3.Block2Records[i]
		block3Record := mnfData.Index3.Block3Records[i]

		id := fmt.Sprintf("%d-%d-%d-%d-%d", block2Record.Id, block2Record.Field2[0], block2Record.Field2[1], block2Record.Flags[0], block2Record.Flags[1])

		uniqueIds[block3Record.ArchiveIndex][id]++
	}

	isDepot := mnfData.IsDepot()
	skip := isDepot

	for i, _ := range mnfData.Index3.Block2Records {
		block2Record := mnfData.Index3.Block2Records[i]
		block3Record := mnfData.Index3.Block3Records[i]

		if isDepot && skip && block3Record.ArchiveIndex != 0 {
			skip = false
		}

		archive, ok := mnfData.Archives[block3Record.ArchiveIndex]
		if !ok {
			return fmt.Errorf("not valid archiveIndex: %d", block3Record.ArchiveIndex)
		}
		if !archive.IsValid(block3Record) {
			continue
		}

		var fileName string

		_, ok = fileNames[block2Record.Id]
		if ok {
			if bytes.Equal(twoZeroBytes, block2Record.Field2) && !skip {
				fileName = fileNames[block2Record.Id]
				delete(fileNames, block2Record.Id)
			}
		}

		var byte10 = []byte("")
		data, err := mnfData.Read(block3Record)
		if err != nil {
			return fmt.Errorf("mnfData.read: %s", err)
		}

		if len(data) < 10 {
			byte10 = data[:]
		} else {
			byte10 = data[0:10]
		}

		ext := extracter.GetExtension(byte10)
		indexes[block3Record.ArchiveIndex]++

		var unique bool
		id := fmt.Sprintf("%d-%d-%d-%d-%d", block2Record.Id, block2Record.Field2[0], block2Record.Field2[1], block2Record.Flags[0], block2Record.Flags[1])
		if uniqueIds[block3Record.ArchiveIndex][id] == 1 {
			unique = true
		}

		csvWriter.Write([]string{
			fmt.Sprintf("%d", i+1),

			"",

			fmt.Sprintf("%d", block2Record.Id),
			fmt.Sprintf("%s", format.BytesFormat(block2Record.Field2)),
			fmt.Sprintf("%s", format.BytesFormat(block2Record.Flags)),

			"",

			fmt.Sprintf("%d", block3Record.UncompressedSize),
			fmt.Sprintf("%d", block3Record.CompressedSize),
			fmt.Sprintf("0x%08x", block3Record.Hash),
			fmt.Sprintf("%d", block3Record.Offset),
			fmt.Sprintf("%d", block3Record.Offset+block3Record.CompressedSize),
			fmt.Sprintf("%d", block3Record.ArchiveIndex),
			fmt.Sprintf("%d", indexes[block3Record.ArchiveIndex]),
			fmt.Sprintf("%t", unique),
			fmt.Sprintf("%d", block3Record.CompressionType),

			"",

			fmt.Sprintf("%s", fileName),
			fmt.Sprintf("%s", ext),
			fmt.Sprintf("%s", format.BytesFormat(byte10)),
		})
	}

	csvWriter.Flush()

	return nil
}
