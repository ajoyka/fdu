package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ajoyka/fdu/fastdu"
	"github.com/h2non/filetype"

	"github.com/h2non/filetype/types"
)

const (
	_outputDateFile = "date-info.json"
	_outputFile     = "file-info.json"
)

// dirCount is used to store byte totals for all files in specified dir
type dirCount struct {
	mu   sync.Mutex
	size map[string]int64
	meta map[string]*fastdu.Meta // file name (not absolute path) -> meta data map
}

type fileCount struct {
	mu     sync.Mutex
	files  int64
	nbytes int64
}

var (

	// fileSizes is used for communicating file size of leaf nodes for incrementing
	// fileCount
	fileSizes = make(chan int64)

	// first 261 bytes is sufficient to identify file type
	fileBuf = make([]byte, 261)

	wg           sync.WaitGroup
	topFiles     = flag.Int("t", 10, "number of top files/directories to display")
	numOpenFiles = flag.Int("c", 20, "concurrency factor")
	summary      = flag.Bool("s", false, "print summary only")

	printInterval = flag.Duration("f", 0*time.Second, "print summary at frequency specified in seconds; default disabled with value 0")
	sema          chan struct{}
)

func main() {
	flag.Parse()
	createBackup(_outputFile)
	fastdu.SortedKeys(nil)
	sema = make(chan struct{}, *numOpenFiles)
	fmt.Println("concurrency factor", cap(sema), *numOpenFiles)
	dirCount := &dirCount{size: make(map[string]int64),
		meta: make(map[string]*fastdu.Meta)}
	fileCount := &fileCount{}

	roots := flag.Args()

	var tick <-chan time.Time
	if *printInterval != 0 {
		tick = time.Tick(*printInterval)
	}

	var nbytes, files int64

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
				fmt.Printf("\n%d files, %.1fGB\n", files, float64(nbytes)/1e9)
			}
		}

	}()

	for _, root := range roots {
		wg.Add(1)
		go walkDir(root, dirCount, fileSizes)
	}

	wg.Wait()
	close(fileSizes)

	printFiles(dirCount)
	files, nbytes = fileCount.Get()
	fmt.Printf("%d files, %.1fGB\n", files, float64(nbytes)/1e9)
	printMeta(dirCount)
	printSortedByDateMeta(dirCount)

}

// create backup file
func createBackup(file string) {
	if _, err := os.Stat(file); err != nil {
		return
	}

	var r []byte
	var err error

	if r, err = ioutil.ReadFile(file); err != nil {
		fmt.Println(err)
		return
	}

	err = ioutil.WriteFile(file+".bak", r, 0644)
	if err != nil {
		fmt.Println(err)
	}
}

// getTop returns aggregated totals for the top level
// directories
func (d *dirCount) getTop() map[string]int64 {
	res := make(map[string]int64)
	for key, val := range d.size {
		top := strings.Split(key, "/")
		res[top[0]] += val
	}
	return res
}

func walkDir(dir string, dirCount *dirCount, fileSizes chan<- int64) {
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
			dirCount.Inc(dir, entry.Size())
			dirCount.AddFile(dir, entry)
			fileSizes <- entry.Size()
		}
	}
}

func getFileType(file string) types.Type {
	fd, _ := os.Open(file)
	defer fd.Close()
	fd.Read(fileBuf)
	// b, _ := ioutil.ReadFile(file)
	kind, _ := filetype.Match(fileBuf)
	return kind
}

// AddFile can accept a path to dir or file as first argument
func (d *dirCount) AddFile(file string, fInfo os.FileInfo) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !fInfo.IsDir() { // if leaf node
		file = filepath.Join(file, fInfo.Name())
	}

	fileType := getFileType(file)
	switch fileType.MIME.Type {
	case "image", "audio", "video":
		// continue
	default:
		return
	}

	base := filepath.Base(file)
	var meta *fastdu.Meta
	var ok bool

	if meta, ok = d.meta[base]; !ok {
		meta = &fastdu.Meta{
			base,
			fInfo.Size(),
			fInfo.ModTime(),
			fileType,
			make([]string, 0),
		}
		d.meta[base] = meta
	}

	meta.Dups = append(meta.Dups, file)
}

func (d *dirCount) Inc(path string, size int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.size[path] += size
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

func dirents(dir string) []os.FileInfo {
	sema <- struct{}{} // acquire token
	defer func() {
		<-sema // release token
	}()

	info, err := ioutil.ReadDir(dir)
	if err != nil {
		if errors.Is(err, syscall.EMFILE) {
			fmt.Printf("\n**Error: %s\nReduce concurrency and retry\n", err)
			os.Exit(1)
		}
		fmt.Printf("%s, %v\n", dir, err)
		return nil
	}
	return info
}

func printMeta(dirCount *dirCount) {
	dirCount.mu.Lock()
	defer dirCount.mu.Unlock()

	b, err := json.MarshalIndent(dirCount.meta, "  ", "  ")
	if err != nil {
		fmt.Println("error:", err)
	}

	f, err := os.Create(_outputFile)
	if err != nil {
		log.Fatal(err)
	}
	f.Write(b)

	fmt.Println("printing duplicate entries (if any)")
	for n, m := range dirCount.meta {
		if len(m.Dups) > 1 {
			fmt.Printf("%s dups: %v %+v\n", n, m.Dups, m.Type)
		}
	}
}

func printSortedByDateMeta(dirCount *dirCount) {
	dirCount.mu.Lock()
	defer dirCount.mu.Unlock()

	m := make(fastdu.SortedMetaByDate, 0)
	for _, v := range dirCount.meta {
		m = append(m, v)
	}

	sort.Sort(fastdu.SortedMetaByDate(m))
	b, err := json.MarshalIndent(m, "  ", "  ")
	if err != nil {
		fmt.Println("error:", err)
	}

	f, err := os.Create(_outputDateFile)
	if err != nil {
		log.Fatal(err)
	}
	f.Write(b)

}

func printFiles(dirCount *dirCount) {
	dirCount.mu.Lock()
	defer dirCount.mu.Unlock()
	fmt.Println("size len:", len(dirCount.size))

	sumKeys := dirCount.getTop()

	var keys []string
	var dc map[string]int64

	if *summary {
		keys = fastdu.SortedKeys(sumKeys)
		dc = sumKeys
	} else {
		keys = fastdu.SortedKeys(dirCount.size)
		dc = dirCount.size
	}

	if *topFiles > len(keys) || *topFiles == -1 {
		fmt.Printf("Printing top available %d\n ", len(keys))
	} else {
		keys = keys[:*topFiles]
		fmt.Printf("Printing top %d dirs/files\n", *topFiles)
	}

	for _, key := range keys {
		size := float64(dc[key])
		sizeGB := size / 1e9
		sizeMB := size / 1e6
		sizeKB := size / 1e3
		var units string

		switch {
		case sizeGB > 0.09:
			size = sizeGB
			units = "GB"
		case sizeMB > 0.09:
			size = sizeMB
			units = "MB"
		default:
			size = sizeKB
			units = "KB"

		}
		fmt.Printf("%.1f%s, %s\n", size, units, key)
	}
}
