package writeLng

import (
	"context"
	"encoding/csv"
	"fmt"
	"github.com/eso-tools/eso-tools/language"
	"github.com/jessevdk/go-flags"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
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

	inputPath, err := filepath.Abs(filepath.Clean(config.Input))
	if err != nil {
		return fmt.Errorf("filepath.Abs: %s", err)
	}

	inputInfo, err := os.Stat(inputPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("'%s' does not exist", inputPath)
	}

	if !inputInfo.IsDir() {
		return fmt.Errorf("'%s' is not a directory", inputPath)
	}

	outputFilePath, err := filepath.Abs(filepath.Clean(config.Output))
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(outputFilePath), 0777)
	if err != nil {
		return err
	}

	log.Printf("Finding .csv")

	rePak := regexp.MustCompile(`.csv$`)

	csvFiles := []string{}
	err = filepath.Walk(inputPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil || !rePak.MatchString(info.Name()) {
			return err
		}

		path, err = filepath.Abs(filepath.Clean(path))
		if err != nil {
			return err
		}

		csvFiles = append(csvFiles, path)

		return nil
	})
	if err != nil {
		return fmt.Errorf("filepath.Walk: %s", err)
	}

	log.Printf("Found %d .csv", len(csvFiles))

	langStore := &language.WriteStore{
		Records: []*language.WriteRecord{},
	}

	log.Printf("Parsing .csv")

	for _, csvFilePath := range csvFiles {
		csvFile, err := os.Open(csvFilePath)
		if err != nil {
			return fmt.Errorf("os.Open: %s", err)
		}

		csvFileName := strings.TrimPrefix(strings.TrimSuffix(filepath.Base(csvFilePath), ".csv"), "0x")
		u64, err := strconv.ParseUint(csvFileName, 16, 32)
		if err != nil {
			return fmt.Errorf("strconv.ParseUint: %s", err)
		}
		domainId := uint32(u64)

		csvReader := csv.NewReader(csvFile)

		for {
			fields, err := csvReader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("csvReader.Read: %s", err)
			}

			u64, err := strconv.ParseUint(fields[0], 10, 32)
			if err != nil {
				return fmt.Errorf("strconv.ParseUint: %s", err)
			}
			variant := uint32(u64)

			u64, err = strconv.ParseUint(fields[1], 10, 32)
			if err != nil {
				return fmt.Errorf("strconv.ParseUint: %s", err)
			}
			id := uint32(u64)

			langStore.Records = append(langStore.Records, &language.WriteRecord{
				DomainId: domainId,
				Variant:  variant,
				Id:       id,
				Value:    fields[2],
			})
		}

		csvFile.Close()
	}

	log.Printf("Writing .lang...")

	langFile, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("os.Create: %s", err)
	}
	defer langFile.Close()

	err = language.WriteWriteStore(langFile, langStore)
	if err != nil {
		return fmt.Errorf("language.Write: %s", err)
	}

	return nil
}
