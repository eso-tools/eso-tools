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
	"sort"
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

	parsedLangData, err := language.Parse(inputFile)
	if err != nil {
		return fmt.Errorf("language.Parse: %s", err)
	}

	langData := &Language{
		Id:      "",
		Domains: map[uint32]*Domain{},
	}

	for _, record := range parsedLangData.Records {
		_, ok := langData.Domains[record.DomainId]
		if !ok {
			langData.Domains[record.DomainId] = &Domain{
				Id:     record.DomainId,
				Groups: map[uint32]*Group{},
			}
		}

		_, ok = langData.Domains[record.DomainId].Groups[record.GroupId]
		if !ok {
			langData.Domains[record.DomainId].Groups[record.GroupId] = &Group{
				Id:      record.GroupId,
				Records: map[uint32]*language.Record{},
			}
		}

		langData.Domains[record.DomainId].Groups[record.GroupId].Records[record.Id] = record
	}

	outputPath, err := filepath.Abs(filepath.Clean(config.Output))
	if err != nil {
		return err
	}

	err = os.MkdirAll(outputPath, 0777)
	if err != nil {
		return err
	}

	for _, domain := range langData.GetDomains() {
		csvPath := filepath.Join(outputPath, fmt.Sprintf("0x%08x.csv", domain.Id))

		file, err := os.Create(csvPath)
		if err != nil {
			return err
		}

		log.Printf("Writing %q", filepath.Base(file.Name()))

		csvWriter := csv.NewWriter(file)

		for _, group := range domain.GetGroups() {
			for _, record := range group.GetRecords() {
				csvWriter.Write([]string{
					fmt.Sprintf("%d", group.Id),
					fmt.Sprintf("%d", record.Id),
					record.Text,
				})
			}
		}

		csvWriter.Flush()
		file.Close()
	}

	return nil
}

type Language struct {
	Id      string
	Domains map[uint32]*Domain
}

func (language *Language) GetDomains() []*Domain {
	keys := uint32Slice{}
	for key, _ := range language.Domains {
		keys = append(keys, key)
	}

	keys.Sort()

	records := make([]*Domain, keys.Len())
	for i, key := range keys {
		records[i] = language.Domains[key]
	}

	return records
}

type Domain struct {
	Id     uint32
	Groups map[uint32]*Group
}

func (domain *Domain) GetGroups() []*Group {
	keys := uint32Slice{}
	for key, _ := range domain.Groups {
		keys = append(keys, key)
	}

	keys.Sort()

	records := make([]*Group, keys.Len())
	for i, key := range keys {
		records[i] = domain.Groups[key]
	}

	return records
}

type Group struct {
	Id      uint32
	Records map[uint32]*language.Record
}

func (group *Group) GetRecords() []*language.Record {
	keys := uint32Slice{}
	for key, _ := range group.Records {
		keys = append(keys, key)
	}

	keys.Sort()

	records := make([]*language.Record, keys.Len())
	for i, key := range keys {
		records[i] = group.Records[key]
	}

	return records
}

type uint32Slice []uint32

func (p uint32Slice) Len() int {
	return len(p)
}

func (p uint32Slice) Less(i, j int) bool {
	return p[i] < p[j]
}

func (p uint32Slice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p uint32Slice) Sort() {
	sort.Sort(p)
}
