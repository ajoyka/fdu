package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/ajoyka/fdu/fastdu"
	_ "github.com/mattn/go-sqlite3"
)

const (
	mediaDB    = "media.db"
	mediaTable = `
CREATE TABLE IF NOT EXISTS media (
	name TEXT PRIMARY KEY,
	size INTEGER,
	datetime DATETIME,
	exif_datetime_original DATETIME,
	mime_type TEXT,
	mime_subtype TEXT,
	mime_value TEXT,
	extension TEXT,
	count INTEGER, -- if > 1 then duplicate occurences
	file_size_mismatch INTEGER, -- 0 -> false, 1 -> true: sqlite does not have boolean type
	common_path TEXT, -- common suffix if duplicate paths exist
	filepath TEXT,
	exif_json TEXT
)
`
	insertMedia = `INSERT OR IGNORE INTO media 
 (name, size, datetime, exif_datetime_original, mime_type, mime_subtype, mime_value, extension, count, file_size_mismatch, common_path, filepath, exif_json) 
 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	duplicatesTable = `
CREATE TABLE IF NOT EXISTS duplicates (
	datetime DATETIME,
	name TEXT,
	size INTEGER,
	filepath TEXT PRIMARY KEY 
)`

	insertDuplicate = `INSERT OR IGNORE INTO duplicates
	(datetime, name, size, filepath)
	VALUES (?, ?, ?, ?)`
)

type DB interface {
	WriteMeta(meta map[string]*fastdu.Meta)       // write metadata to db
	WriteDuplicates(meta map[string]*fastdu.Meta) // write duplicates to db
	Close()                                       // close database
}

type DBImpl struct {
	media *sql.DB
	dups  *sql.DB // duplicate file db - for future use
}

// New creates a new db and tables associated with it if they don't exist
func New() (DB, error) {
	// set data source name
	// Check link for avoiding db lock errors: https://github.com/mattn/go-sqlite3?tab=readme-ov-file#faq
	dsn := fmt.Sprintf("file:%s?cache=shared", mediaDB)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		log.Fatal(err)
	}

	// ping database to verify connection
	// create table
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec(mediaTable)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(duplicatesTable)
	if err != nil {
		return nil, err
	}

	return &DBImpl{
		media: db,
	}, nil
}

// func createDB(dsn string, table string) (*sql.DB, error) {
// 	_, err = db.Exec(table)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return db, nil
// }

type job struct {
	file string
	meta *fastdu.Meta
}

func (d *DBImpl) WriteDuplicates(meta map[string]*fastdu.Meta) {
	var dupRows atomic.Uint64
	var newRows atomic.Uint64

	jobs := make(chan job)
	go func() {
		defer close(jobs)
		for file, m := range meta {
			jobs <- job{file, m}
		}
	}()

	numWorkers := 10
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	stmt, err := d.media.Prepare(insertDuplicate)
	if err != nil {
		log.Fatalf("Duplicates prepare: %v", err)
	}

	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			// each worker consumes from broadcast channel of jobs
			for job := range jobs {
				m := job.meta
				for _, dup := range m.Dups {
					result, err := stmt.Exec(m.Modtime, job.file, dup.Size, dup.Name)
					if err != nil {
						log.Fatalf("insert duplicate %v", err)
					}

					rowsAffected, err := result.RowsAffected()
					if err != nil {
						log.Fatalf("rows affected %v", err)
					}
					if rowsAffected == 0 {
						dupRows.Add(1)
					} else {
						newRows.Add(1)
					}
				}
			}
		}()
	}
	wg.Wait()
	log.Printf("duplicate filepath insertion skip count: %d", dupRows.Load())
	log.Printf("new duplicate row insertions: %d", newRows.Load())
	log.Println("Inserted to duplicate rows database successfully")

}

func (d *DBImpl) WriteMeta(meta map[string]*fastdu.Meta) {
	var dupRows atomic.Uint64
	var newRows atomic.Uint64

	numWorkers := 8

	// add all jobs to jobs channel - using unbuffered channel that many workers listen to
	jobs := make(chan job)
	go func() { // has to be go routine as we are using unbuffered channel
		defer close(jobs)
		for file, m := range meta {
			jobs <- job{file: file, meta: m}
		}
	}()

	var wg sync.WaitGroup
	// todo: user errGroup
	wg.Add(numWorkers)

	stmt, err := d.media.Prepare(insertMedia)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for i := 0; i < numWorkers; i++ {
		go func() { // spawn worker that consumes jobs from global channel
			defer wg.Done()
			for job := range jobs {
				m := job.meta
				filepath, _ := json.Marshal(m.Dups)
				count := len(m.Dups)

				exifData, _ := json.Marshal(m.Exif)
				var dateTimeOriginal sql.NullTime
				dateTimeOriginal.Valid = false
				if m.MIME.Type == "image" {
					dateTimeOriginal.Valid = true
					dateTimeOriginal.Time = m.Exif.DateTimeOriginal()
				}

				result, err := stmt.Exec(job.file, m.Size, m.Modtime,
					dateTimeOriginal,
					m.MIME.Type, m.MIME.Subtype, m.MIME.Value, m.Extension,
					count, m.FileSizeMismatch,
					findCommonPath(m.Dups),
					string(filepath),
					string(exifData))
				if err != nil {
					log.Fatalf("insertion error %v\n", err)
				}
				rowsAffected, err := result.RowsAffected()
				if err != nil {
					log.Fatalf("rows affected error %v", err)
				}
				if rowsAffected == 0 {
					dupRows.Add(1)
				} else {
					newRows.Add(1)
				}
			}
		}()
	}
	wg.Wait()
	log.Printf("skipped duplicate row insertions: %d", dupRows.Load())
	log.Printf("new rows added: %d", newRows.Load())
	log.Println("Inserted to media database successfully")
}

func (d *DBImpl) Close() {
	d.Close()
}

// findCommonPath finds the common path suffix from the bottom to the top
// ex: /a/b/c/x.jpg, /m/b/c/x.jpg as duplicates will return b/c/x.jpg
// this indicates a duplicate file/folder that can be removed/ignored
func findCommonPath(dups []fastdu.Duplicate) string {
	if len(dups) == 1 {
		return ""
	}
	dMap := map[string]int{}
	maxPathCnt := math.MinInt32
	maxPath := []string{}

	for _, dup := range dups {
		components := strings.Split(dup.Name, string(filepath.Separator))
		for _, comp := range components {
			dMap[comp] += 1
		}
		// we need to find max path and walk backwards subsequently to identify common suffix
		// ex: /a/b/c/d.jpg, /a/b/c/x/d.jpg cnts for a: 2, b:2, c:2, x:1, d.jpg:2; so taking longest
		// path will correctly identify the common suffix as d.jpg as the correct one

		if len(components) > maxPathCnt {
			maxPathCnt = len(components)
			slices.Reverse(components)
			maxPath = components
		}
	}
	cnt := dMap[maxPath[0]] // get reversed path leaf cnt to start with
	commonPath := []string{}
	for _, comp := range maxPath {
		if dMap[comp] < cnt {
			break
		}
		commonPath = append(commonPath, comp)
	}
	slices.Reverse(commonPath)
	return strings.Join(commonPath, string(filepath.Separator))
}
