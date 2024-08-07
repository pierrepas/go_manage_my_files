package filegorithms

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

type fileInfo struct {
	path string
	hash string
}

func CheckDuplicateFiles(outputFile string, pathToSearch string) {
	// Parses all the files in the pathToSearch directory and prints a list of duplicates to outputFile.

	log.Println("Checking for duplicate files.")
	log.Println("Searching the path:", pathToSearch)
	log.Println("Writing to file:", outputFile)
	log.Println()
	duplicateCount := 0

	f, err := os.OpenFile(outputFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	filesChan := make(chan string)
	resultsChan := make(chan fileInfo)
	var wg sync.WaitGroup

	// Start worker goroutines
	numWorkers := runtime.NumCPU()
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(filesChan, resultsChan, &wg)
	}

	// Start a goroutine to walk the directory
	go func() {
		err := filepath.Walk(pathToSearch, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				filesChan <- path
			}
			return nil
		})
		if err != nil {
			log.Printf("Error walking through directory: %v\n", err)
		}
		close(filesChan)
	}()

	// Use a separate goroutine to collect results
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results in the main goroutine
	fileHashes := make(map[string][]string)
	for result := range resultsChan {
		fileHashes[result.hash] = append(fileHashes[result.hash], result.path)
	}

	// Write results
	writer := bufio.NewWriter(f)
	for hash, paths := range fileHashes {
		if len(paths) > 1 {
			log.Println("Duplicate files found (hash:", hash, "):")
			duplicateCount++
			for _, path := range paths {
				log.Println(path)
				writer.WriteString(path + "\n")
			}
			log.Println()
			writer.WriteString("\n")
		}
	}

	writer.Flush()
	dupCntStr := fmt.Sprint(duplicateCount)
	log.Println(dupCntStr + " duplicates found.")
}

func worker(files <-chan string, results chan<- fileInfo, wg *sync.WaitGroup) {
	defer wg.Done()
	for file := range files {
		hash, err := calculateMD5(file)
		if err != nil {
			log.Printf("Error calculating hash for %s: %v\n", file, err)
			continue
		}
		results <- fileInfo{path: file, hash: hash}
	}
}

func calculateMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
