package testZosft

import (
	"bytes"
	"context"
	"fmt"
	"github.com/eso-tools/eso-tools/extracter"
	"github.com/eso-tools/eso-tools/mnf"
	"github.com/eso-tools/eso-tools/zosft"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
	"path/filepath"
)

type Config struct {
	Input string `long:"input" short:"i" required:"true"`
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

	recordChan := make(chan *extracter.Record, 100)
	errorChan := make(chan error, 1)

	go func() {
		extracter.CombineRecords(mnfData, recordChan, errorChan)
	}()

	log.Printf("Scanning...")

Loop:
	for {
		select {
		case record, ok := <-recordChan:
			if !ok {
				break Loop
			}

			data, err := mnfData.Read(record.Record3)
			if err != nil {
				return fmt.Errorf("mnfData.Read: %s", err)
			}

			_, err = zosft.Parse(bytes.NewReader(data))
			if err == nil {
				log.Printf("zosft id: %d [0x%08x]", record.Record2.Id, record.Record2.Id)
				break Loop
			}

		case err, ok := <-errorChan:
			if ok {
				return fmt.Errorf("extracter.CombineRecords: %s", err)
			}

		default:

		}
	}

	return nil
}
