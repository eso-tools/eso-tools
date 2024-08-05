package dumpIndex

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"github.com/eso-tools/eso-tools/extracter"
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

var twoZeroBytes = []byte{0x00, 0x00}

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

	zosftData, err := mnfData.GetZosft()
	if err != nil {
		log.Fatalf("mnfData.GetZosft: %s", err)
	}

	fileNames := map[uint32]string{}
	if zosftData != nil {
		fileNames = zosftData.GetFileNamesById()
	}

	if len(mnfData.Index3.Block2Records) != len(mnfData.Index3.Block3Records) {
		return fmt.Errorf("len(mnfData.Index3.Block2Records) != len(mnfData.Index3.Block3Records)")
	}

	err = os.MkdirAll(filepath.Dir(config.Output), 0777)
	if err != nil {
		return fmt.Errorf("os.MkdirAll: %s", err)
	}

	f, err := os.Create(config.Output)
	if err != nil {
		return fmt.Errorf("os.Create: %s", err)
	}
	defer f.Close()

	log.Printf("Writing \"%s\"...", config.Output)

	csvWriter := csv.NewWriter(f)

	csvWriter.Write([]string{
		"rawName",
		"archive",
		"offset",
		"compType",
		"compSize",
		"uncompSize",
		"fileName",
	})

	isDepot := mnfData.IsDepot()
	skip := isDepot

	for i := 0; i < len(mnfData.Index3.Block2Records); i++ {
		record := &extracter.Record{
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
			log.Fatalf("not valid archiveIndex: %d", record.Record3.ArchiveIndex)
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

		//if record.Record3.ArchiveIndex != 0 {
		//	continue
		//}

		csvWriter.Write([]string{
			fmt.Sprintf("%s", record.GetRawFilename()),
			fmt.Sprintf("%d", record.Record3.ArchiveIndex),
			fmt.Sprintf("%d", record.Record3.Offset),
			fmt.Sprintf("%d", record.Record3.CompressionType),
			fmt.Sprintf("%d", record.Record3.CompressedSize),
			fmt.Sprintf("%d", record.Record3.UncompressedSize),
			fmt.Sprintf("%s", record.FileName),
		})
	}

	csvWriter.Flush()

	return nil
}
