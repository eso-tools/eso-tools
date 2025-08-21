package extractFile

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/eso-tools/eso-tools/extracter"
	"github.com/eso-tools/eso-tools/mnf"
	"github.com/jessevdk/go-flags"
	"github.com/new-world-tools/new-world-tools/profiler"
	workerpool "github.com/zelenin/go-worker-pool"
)

const (
	defaultThreads uint8 = 3
	maxThreads     uint8 = 5
)

type Config struct {
	Input   string `long:"input" short:"i" required:"true"`
	Output  string `long:"output" short:"o" required:"true"`
	Id      string `long:"id" required:"true"`
	Threads uint8  `long:"threads" short:"t"`
}

var re = regexp.MustCompile(`(?i)^(0x)?([0-9a-f]{8})-([0-9a-f]{8})`)

func Command(ctx context.Context, args []string) error {
	var config Config
	_, err := flags.ParseArgs(&config, args[1:])
	if err != nil {
		return nil
	}

	var (
		pool         *workerpool.Pool
		pr           = profiler.New()
		searchRecord mnf.Block2Record
	)

	matches := re.FindStringSubmatch(config.Id)
	if matches == nil {
		return fmt.Errorf("Invalid id: %s", config.Id)
	}

	fmt.Sscanf(matches[2], `%08x`, &searchRecord.Id)
	fmt.Sscanf(matches[3], `%04x%04x`, &searchRecord.Field2, &searchRecord.Flags)

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

	outputDirPath, err := filepath.Abs(filepath.Clean(config.Output))
	if err != nil {
		return fmt.Errorf("filepath.Abs: %s", err)
	}

	threads := config.Threads
	if threads < 1 || threads > maxThreads {
		threads = defaultThreads
	}

	err = os.MkdirAll(outputDirPath, 0755)
	if err != nil {
		return fmt.Errorf("MkdirAll: %s", err)
	}

	log.Printf("Parsing %q...", inputFilePath)
	mnfData, err := mnf.Parse(inputFilePath)
	if err != nil {
		return fmt.Errorf("mnf.Parse: %s", err)
	}

	pool = workerpool.NewPool(int64(threads), 1000)

	go func() {
		errorChan := pool.Errors()

		for {
			err, ok := <-errorChan
			if !ok {
				break
			}

			log.Printf("%s", err)
		}
	}()

	log.Printf("Prepare records...")

	addTask := func(id int64, total int, file *extracter.Record, mnfData *mnf.Mnf) {
		pool.AddTask(func(ctx context.Context) error {
			if id%10000 == 0 {
				log.Printf("Task %d/%d", id, total)
			}

			data, err := mnfData.Read(file.Record3)
			if err != nil {
				log.Fatalf("mnfData.Read: %s", err)
			}

			file.Data = data

			fpath := filepath.Join(outputDirPath, fmt.Sprintf("%03d", file.Record3.ArchiveIndex), file.GetRawFilename())
			err = os.MkdirAll(filepath.Dir(fpath), 0755)
			if err != nil {
				log.Fatalf("os.MkdirAll: %s", err)
			}

			dest, err := os.Create(fpath)
			if err != nil {
				log.Fatalf("os.Create: %s", err)
			}

			reader := bytes.NewReader(data)

			_, err = io.Copy(dest, reader)
			if err != nil {
				log.Fatalf("io.Copy: %s", err)
			}

			dest.Close()

			if file.FileName != "" {
				fpath = filepath.Join(outputDirPath, file.FileName)
				err = os.MkdirAll(filepath.Dir(fpath), 0755)
				if err != nil {
					log.Fatalf("os.MkdirAll: %s", err)
				}

				dest, err := os.Create(fpath)
				if err != nil {
					log.Fatalf("os.Create: %s", err)
				}

				_, err = reader.Seek(0, io.SeekStart)
				if err != nil {
					log.Fatalf("dataReader.Seek: %s", err)
				}

				_, err = io.Copy(dest, reader)
				if err != nil {
					log.Fatalf("io.Copy: %s", err)
				}

				dest.Close()
			}

			return nil
		})
	}

	recordChan := make(chan *extracter.Record, 100)
	errorChan := make(chan error, 1)

	go func() {
		extracter.CombineRecords(mnfData, recordChan, errorChan)
	}()

	log.Printf("Searching...")

	var i int64
Loop:
	for {
		select {
		case record, ok := <-recordChan:
			if !ok {
				break Loop
			}

			if record.Record2.Id == searchRecord.Id && bytes.Equal(record.Record2.Field2, searchRecord.Field2) && bytes.Equal(record.Record2.Flags, searchRecord.Flags) {
				log.Printf("Extracting...")
				addTask(i+1, int(mnfData.Index3.Count3), record, mnfData)
				break Loop
			}

			i++

		case err, ok := <-errorChan:
			if ok {
				return fmt.Errorf("extracter.CombineRecords: %s", err)
			}

		default:

		}
	}

	pool.Wait()

	log.Printf("PeakMemory: %0.1fMb Duration: %s", float64(pr.GetPeakMemory())/1024/1024, pr.GetDuration().String())

	return nil
}
