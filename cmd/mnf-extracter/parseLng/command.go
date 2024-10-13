package parseLng

import (
	"context"
	"encoding/csv"
	"fmt"
	"github.com/eso-tools/eso-tools/language"
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

	inputFile, err := os.Open(inputFilePath)
	if err != nil {
		return fmt.Errorf("os.Open: %s", err)
	}
	defer inputFile.Close()

	log.Printf("Parsing %q...", filepath.Base(inputFile.Name()))

	langStore, err := language.ParseReadStore(inputFile)
	if err != nil {
		return fmt.Errorf("language.ParseReadStore: %s", err)
	}

	outputPath, err := filepath.Abs(filepath.Clean(config.Output))
	if err != nil {
		return err
	}

	err = os.MkdirAll(outputPath, 0777)
	if err != nil {
		return err
	}

	for _, domainId := range langStore.GetDomainIds() {
		csvPath := filepath.Join(outputPath, fmt.Sprintf("0x%08x.csv", domainId))

		file, err := os.Create(csvPath)
		if err != nil {
			return err
		}

		log.Printf("Writing %q", filepath.Base(file.Name()))

		csvWriter := csv.NewWriter(file)

		for _, id := range langStore.GetIds(domainId) {
			for _, record := range langStore.GetRecords(domainId, id) {
				csvWriter.Write([]string{
					fmt.Sprintf("%d", record.Id),
					fmt.Sprintf("%d", record.Variant),
					langStore.GetValueByRecord(record),
				})
			}
		}

		csvWriter.Flush()
		file.Close()
	}

	return nil
}
