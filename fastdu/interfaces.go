package fastdu

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/h2non/filetype"
	"github.com/h2non/filetype/types"
)

const (
	dupFile = "duplicates.json"
)

// DUtil is an interface to describe utilities that are used
// for directory traversal and collect meta data
type DUtil interface {
	Inc(path string, size int64)       // increment totals per dir
	WriteMeta(file string)             // write Meta data in json format
	WriteMetaSortedByDate(file string) // write meta data sorted by date
	WriteMetaSortedBySize(file string) // write meta data sorted by file size

}

// DirCount is used to store byte totals for all files in specified dir
type DirCount struct {
	mu    sync.Mutex
	size  map[string]int64 // store cumulative totals of file sizes by dir hierarchy
	meta  map[string]*Meta // file name (not absolute path) -> meta data map
	dList []duplicates     // duplicate list for current search
}

// Meta stores metadata about the file such as os.stat info, filetype info
type Meta struct {
	Name    string // base file name
	Size    int64
	Modtime time.Time
	types.Type
	Dups []string // potential list of duplicates
}

type duplicates struct {
	types.Type
	Dups []string
}

var (

	// first 261 bytes is sufficient to identify file type
	fileBuf = make([]byte, 261)
)

// NewDirCount is a function that returns a new DirCount that
// implements DUtil
func NewDirCount() *DirCount {
	return &DirCount{size: make(map[string]int64),
		meta:  make(map[string]*Meta),
		dList: make([]duplicates, 0), // 0 cap slice since duplciates may not exist
	}
}

// WriteMeta is used to write meta data to specified file
func (d *DirCount) WriteMeta(file string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	writeJson(d.meta, file)
	// create duplicate files info if any exist
	for _, m := range d.meta {
		if len(m.Dups) > 1 {
			d.dList = append(d.dList,
				duplicates{m.Type, m.Dups})
		}
	}

	writeJson(d.dList, dupFile)
}

// write jsone data to specified file
func writeJson(d interface{}, file string) {

	fmt.Printf("Creating json file %s\n", file)
	b, err := json.MarshalIndent(d, "  ", "  ")
	if err != nil {
		fmt.Println("error:", err)
	}

	f, err := os.Create(file)
	if err != nil {
		log.Fatal(err)
	}
	f.Write(b)
}

// create json file with list of duplicates
func (d *DirCount) writeDups() {
	d.mu.Lock()
	defer d.mu.Unlock()

}

// GetTop returns aggregated totals for the top level
// directories
func (d *DirCount) GetTop() map[string]int64 {
	res := make(map[string]int64)
	for key, val := range d.size {
		top := strings.Split(key, "/")
		res[top[0]] += val
	}
	return res
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
func (d *DirCount) AddFile(file string, fInfo os.FileInfo) {
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
	var meta *Meta
	var ok bool

	if meta, ok = d.meta[base]; !ok {
		meta = &Meta{
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

// Inc increases the cumulative file size count by directory
func (d *DirCount) Inc(path string, size int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.size[path] += size
}

// WriteMetaSortedByDate prints meta data sorted by date
func (d *DirCount) WriteMetaSortedByDate(file string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	m := make(SortedMetaByDate, 0)
	for _, v := range d.meta {
		m = append(m, v)
	}

	sort.Sort(SortedMetaByDate(m))
	b, err := json.MarshalIndent(m, "  ", "  ")
	if err != nil {
		fmt.Println("error:", err)
	}

	f, err := os.Create(file)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Writing meta data json file %s\n", file)
	f.Write(b)
	f.Close()

}

// WriteMetaSortedBySize writes meta data sorted by file size
func (d *DirCount) WriteMetaSortedBySize(file string) {
	d.mu.Lock()
	d.mu.Unlock()

	m := make(SortedFileBySize, 0)
	for _, v := range d.meta {
		m = append(m, v)
	}

	sort.Sort(SortedFileBySize(m))
	writeMetaData(file, m)
}

// writeMetaData writes data to file
func writeMetaData(file string, data interface{}) {
	m, err := json.MarshalIndent(data, " ", " ")
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Create(file)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Writing size sorted json file %s\n", file)
	f.Write(m)
	f.Close()
}

// PrintFiles prints top files disk usage similar to du
func (d *DirCount) PrintFiles(topFiles int, summary bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	fmt.Println("size len:", len(d.size))

	sumKeys := d.GetTop()

	var keys []string
	var dc map[string]int64

	if summary {
		keys = SortedKeys(sumKeys)
		dc = sumKeys
	} else {
		keys = SortedKeys(d.size)
		dc = d.size
	}

	if topFiles > len(keys) || topFiles == -1 {
		fmt.Printf("Printing top available %d\n ", len(keys))
	} else {
		keys = keys[:topFiles]
		fmt.Printf("Printing top %d dirs/files\n", topFiles)
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
