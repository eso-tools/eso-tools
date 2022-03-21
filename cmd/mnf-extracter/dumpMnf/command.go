package dumpMnf

import (
	"context"
	"encoding/csv"
	"fmt"
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

	log.Printf("Writing \"%s\"...", config.Output)

	csvWriter := csv.NewWriter(f)
	//csvReader.Comma

	csvWriter.Write([]string{
		"Index",

		"",

		"Id",
		"Field2",
		"Flags",

		"",

		"UncompressedSize",
		"CompressedSize",
		"Hash",
		"Offset",
		"ArchiveIndex",
		"CompressionType",
	})

	for i, _ := range mnfData.Index3.Block2Records {
		block2Record := mnfData.Index3.Block2Records[i]
		block3Record := mnfData.Index3.Block3Records[i]

		csvWriter.Write([]string{
			fmt.Sprintf("%d", i),

			"",

			fmt.Sprintf("%d", block2Record.Id),
			fmt.Sprintf("%s", format.BytesFormat(block2Record.Field2)),
			fmt.Sprintf("%s", format.BytesFormat(block2Record.Flags)),

			"",

			fmt.Sprintf("%d", block3Record.UncompressedSize),
			fmt.Sprintf("%d", block3Record.CompressedSize),
			fmt.Sprintf("0x%08x", block3Record.Hash),
			fmt.Sprintf("%d", block3Record.Offset),
			fmt.Sprintf("%d", block3Record.ArchiveIndex),
			fmt.Sprintf("%d", block3Record.CompressionType),
		})
	}

	csvWriter.Flush()

	return nil
}
