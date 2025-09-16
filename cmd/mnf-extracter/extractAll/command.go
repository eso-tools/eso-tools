package extractAll

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/eso-tools/eso-tools/extracter"
	"github.com/eso-tools/eso-tools/mnf"
	"github.com/jessevdk/go-flags"
	"github.com/new-world-tools/new-world-tools/hash"
	"github.com/new-world-tools/new-world-tools/profiler"
	"github.com/zelenin/go-texconv"
	workerpool "github.com/zelenin/go-worker-pool"
)

const (
	defaultThreads uint8 = 3
	maxThreads     uint8 = 5
)

var supportedFormatsForDdsConverting = map[string]bool{
	"jpg": true,
	"png": true,
	//"webp": true,
}

type Config struct {
	Input        string `long:"input" short:"i" required:"true"`
	Output       string `long:"output" short:"o" required:"true"`
	Threads      uint8  `long:"threads" short:"t"`
	HashSumFile  string `long:"hashSumFile" short:"h"`
	ConvertDdsTo string `long:"convert-dds-to"`
}

func Command(ctx context.Context, args []string) error {
	var config Config
	_, err := flags.ParseArgs(&config, args[1:])
	if err != nil {
		return nil
	}

	var (
		hashRegistry    *hash.Registry
		hashSumFilePath string
		pool            *workerpool.Pool
		pr              = profiler.New()
	)

	inputFilePath, err := filepath.Abs(filepath.Clean(config.Input))
	if err != nil {
		return fmt.Errorf("filepath.Abs: %s", err)
	}

	inputFileInfo, err := os.Stat(inputFilePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("%q does not exist", inputFilePath)
	}

	if inputFileInfo.IsDir() {
		return fmt.Errorf("%q is not a file", inputFilePath)
	}

	outputDirPath, err := filepath.Abs(filepath.Clean(config.Output))
	if err != nil {
		return fmt.Errorf("filepath.Abs: %s", err)
	}

	threads := config.Threads
	if threads < 1 || threads > maxThreads {
		threads = defaultThreads
	}

	if config.HashSumFile != "" {
		hashSumFilePath, err = filepath.Abs(filepath.Clean(config.HashSumFile))
		if err != nil {
			return fmt.Errorf("filepath.Abs: %s", err)
		}
		hashRegistry = hash.NewRegistry()
	}

	if config.ConvertDdsTo != "" && !supportedFormatsForDdsConverting[config.ConvertDdsTo] {
		log.Fatalf("Unsupported format for converting: %s", config.ConvertDdsTo)
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
				log.Printf("mnfData.Read: %s", err)
				return err
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

			dataReader := bytes.NewReader(data)

			var hashSum []byte

			if hashSumFilePath == "" {
				reader := dataReader

				_, err = io.Copy(dest, reader)
				if err != nil {
					log.Fatalf("io.Copy: %s", err)
				}
			} else {
				hasher := sha1.New()
				reader := io.TeeReader(dataReader, hasher)

				_, err = io.Copy(dest, reader)
				if err != nil {
					log.Fatalf("io.Copy: %s", err)
				}

				hashSum = hasher.Sum(nil)

				hashRegistry.Add(filepath.ToSlash(filepath.Join(fmt.Sprintf("%03d", file.Record3.ArchiveIndex), file.GetRawFilename())), hashSum)
			}

			dest.Close()

			if config.ConvertDdsTo != "" && filepath.Ext(fpath) == ".dds" {
				ddsPath := fpath
				// texconv does not accept absolute linux paths
				if runtime.GOOS == "linux" {
					curDir, err := os.Getwd()
					if err != nil {
						return err
					}

					relPath, err := filepath.Rel(curDir, fpath)
					if err != nil {
						return err
					}
					ddsPath = relPath
				}

				args := []string{
					"-ft",
					config.ConvertDdsTo,
					"-f",
					"R8G8B8A8_UNORM_SRGB",
					"-y",
					"-o",
					filepath.Dir(ddsPath),
					ddsPath,
				}

				_, err := texconv.Texconv(args, false, true, true)
				if err == nil {
					os.Remove(fpath)
					if hashSumFilePath != "" {
						hasher := sha1.New()
						f, err := os.Open(strings.TrimSuffix(fpath, ".dds") + "." + config.ConvertDdsTo)
						if err != nil {
							return err
						}
						_, err = io.Copy(hasher, f)
						if err != nil {
							return err
						}

						f.Close()

						hashRegistry.Add(filepath.ToSlash(strings.TrimSuffix(filepath.Join(fmt.Sprintf("%03d", file.Record3.ArchiveIndex), file.GetRawFilename()), ".dds")+"."+config.ConvertDdsTo), hasher.Sum(nil))
						hashRegistry.Remove(filepath.ToSlash(filepath.Join(fmt.Sprintf("%03d", file.Record3.ArchiveIndex), file.GetRawFilename())))
					}
				}
			}

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

				_, err = dataReader.Seek(0, io.SeekStart)
				if err != nil {
					log.Fatalf("dataReader.Seek: %s", err)
				}

				_, err = io.Copy(dest, dataReader)
				if err != nil {
					log.Fatalf("io.Copy: %s", err)
				}

				if hashSum != nil {
					hashRegistry.Add(filepath.ToSlash(file.FileName), hashSum)
				}

				dest.Close()

				if config.ConvertDdsTo != "" && filepath.Ext(fpath) == ".dds" {
					ddsPath := fpath
					// texconv does not accept absolute linux paths
					if runtime.GOOS == "linux" {
						curDir, err := os.Getwd()
						if err != nil {
							return err
						}

						relPath, err := filepath.Rel(curDir, fpath)
						if err != nil {
							return err
						}
						ddsPath = relPath
					}

					args := []string{
						"-ft",
						config.ConvertDdsTo,
						"-f",
						"R8G8B8A8_UNORM_SRGB",
						"-y",
						"-o",
						filepath.Dir(ddsPath),
						ddsPath,
					}

					_, err := texconv.Texconv(args, false, true, true)
					if err == nil {
						os.Remove(fpath)
						if hashSumFilePath != "" {
							hasher := sha1.New()
							f, err := os.Open(strings.TrimSuffix(fpath, ".dds") + "." + config.ConvertDdsTo)
							if err != nil {
								return err
							}
							_, err = io.Copy(hasher, f)
							if err != nil {
								return err
							}

							f.Close()

							hashRegistry.Add(filepath.ToSlash(strings.TrimSuffix(file.FileName, ".dds")+"."+config.ConvertDdsTo), hasher.Sum(nil))
							hashRegistry.Remove(filepath.ToSlash(file.FileName))
						}
					}
				}
			}

			return nil
		})
	}

	recordChan := make(chan *extracter.Record, 100)
	errorChan := make(chan error, 1)

	go func() {
		extracter.CombineRecords(mnfData, recordChan, errorChan)
	}()

	log.Printf("Extracting...")

	var i int64
Loop:
	for {
		select {
		case record, ok := <-recordChan:
			if !ok {
				break Loop
			}

			addTask(i+1, int(mnfData.Index3.Count3), record, mnfData)
			i++

		case err, ok := <-errorChan:
			if ok {
				return fmt.Errorf("extracter.CombineRecords: %s", err)
			}

		default:

		}
	}

	pool.Wait()

	if hashSumFilePath != "" {
		log.Printf("Writing %s", hashSumFilePath)

		hashes := hashRegistry.Hashes()
		sort.Slice(hashes, func(i, j int) bool {
			return hashes[i].FileName < hashes[j].FileName
		})

		hashSumsFile, err := os.Create(hashSumFilePath)
		if err != nil {
			return fmt.Errorf("os.Create: %s", err)
		}
		defer hashSumsFile.Close()

		for _, fileHash := range hashes {
			_, err = hashSumsFile.WriteString(fmt.Sprintf("%x *%s\n", fileHash.Hash, fileHash.FileName))
			if err != nil {
				return fmt.Errorf("hashSumsFile.WriteString: %s", err)
			}
		}
	}

	log.Printf("PeakMemory: %0.1fMb Duration: %s", float64(pr.GetPeakMemory())/1024/1024, pr.GetDuration().String())

	return nil
}
