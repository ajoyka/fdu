package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/ajoyka/fdu/db"
	"github.com/ajoyka/fdu/fastdu"
)

const (
	_outputDateFile = "date-info.json"
	_outputFile     = "file-info.json"
	_outputSizeFile = "size-info.json"
)

type fileCount struct {
	mu     sync.Mutex
	files  int64
	nbytes int64
}

var (

	// fileSizes is used for communicating file size of leaf nodes for incrementing
	// fileCount
	fileSizes = make(chan int64)

	wg           sync.WaitGroup
	topFiles     = flag.Int("t", 10, "number of top files/directories to display")
	numOpenFiles = flag.Int("c", 20, "concurrency factor")
	summary      = flag.Bool("s", false, "print summary only")

	printInterval = flag.Duration("f", 5*time.Second, "print summary at frequency specified in seconds; default disabled with value 0")
	sema          chan struct{}
)

func main() {
	flag.Parse()
	createBackup(_outputFile)
	fastdu.SortedKeys(nil)
	sema = make(chan struct{}, *numOpenFiles)
	fmt.Println("concurrency factor", cap(sema), *numOpenFiles)
	dirCount := fastdu.NewDirCount()
	fileCount := &fileCount{}

	roots := flag.Args()

	var tick <-chan time.Time
	if *printInterval != 0 {
		tick = time.Tick(*printInterval)
	}

	var nbytes, files int64

	fmt.Println()
	go func() {
	loop:
		for {
			select {
			case size, ok := <-fileSizes:
				if !ok {
					break loop // fileSizes was closed
				}
				fileCount.Inc(size)

			case <-tick:
				files, nbytes = fileCount.Get()
				fmt.Printf("%d files, %.1fGB\n", files, float64(nbytes)/1e9)
			}
		}

	}()

	for _, root := range roots {
		wg.Add(1)
		go walkDir(root, dirCount, fileSizes)
	}

	wg.Wait()
	close(fileSizes)
	db, err := db.New()
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	// defer db.Close()

	dirCount.PrintFiles(*topFiles, *summary)
	files, nbytes = fileCount.Get()
	fmt.Printf("%d files, %.1fGB\n", files, float64(nbytes)/1e9)
	dirCount.WriteMeta(_outputFile)
	db.WriteMeta(dirCount.Meta)
	dirCount.WriteMetaSortedByDate(_outputDateFile)
	dirCount.WriteMetaSortedBySize(_outputSizeFile)
	fmt.Println(dirCount.Counters())

}

// create backup file
func createBackup(file string) {
	if _, err := os.Stat(file); err != nil {
		return
	}

	var r []byte
	var err error

	if r, err = os.ReadFile(file); err != nil {
		fmt.Println(err)
		return
	}

	err = os.WriteFile(file+".bak", r, 0644)
	if err != nil {
		fmt.Println(err)
	}
}

func walkDir(dir string, dirCount *fastdu.DirCount, fileSizes chan<- int64) {
	defer wg.Done()

	// handle case when fastdu is invoked including files as args like so: fastdu *
	// check if 'dir' is a file

	fInfo, err := os.Stat(dir)
	if err != nil {
		fmt.Print(err)
	} else if !fInfo.IsDir() { // 'dir' is a file
		dirCount.Inc(dir, fInfo.Size())
		dirCount.AddFile(dir, fInfo)
		fileSizes <- fInfo.Size()
		return
	}

	for _, entry := range dirents(dir) {
		if entry.IsDir() {
			wg.Add(1)
			go walkDir(filepath.Join(dir, entry.Name()), dirCount, fileSizes)
		} else {
			info, err := entry.Info()
			if err != nil {
				fmt.Printf("Error getting fileinfo %s: %v\n", entry.Name(), err)
				continue
			}
			dirCount.Inc(dir, info.Size())
			dirCount.AddFile(dir, info)
			fileSizes <- info.Size()
		}
	}
}

func (f *fileCount) Inc(size int64) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.files++
	f.nbytes += size
}

func (f *fileCount) Get() (int64, int64) {
	f.mu.Lock()
	f.mu.Unlock()

	return f.files, f.nbytes
}

func dirents(dir string) []os.DirEntry {
	sema <- struct{}{} // acquire token
	defer func() {
		<-sema // release token
	}()

	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, syscall.EMFILE) {
			fmt.Printf("\n**Error: %s\nReduce concurrency and retry\n", err)
			os.Exit(1)
		}
		fmt.Printf("%s, %v\n", dir, err)
		return nil
	}
	return dirEntries
}
