package db

import (
	"database/sql"
	"log"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/ajoyka/fdu/fastdu"
	_ "github.com/mattn/go-sqlite3"
)

const (
	createTable = `
CREATE TABLE IF NOT EXISTS media (
	name TEXT PRIMARY KEY,
	size INTEGER,
	datetime DATETIME,
	mime_type TEXT,
	mime_subtype TEXT,
	mime_value TEXT,
	extension TEXT,
	filepath TEXT
)
`
	insert = `INSERT OR IGNORE INTO media (name, size, datetime, mime_type, mime_subtype, mime_value, extension, filepath) 
 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
)

type DB interface {
	WriteMeta(meta map[string]*fastdu.Meta) // write metadata to db
	Close()                                 // close database
}

type Db struct {
	*sql.DB
}

// New creates a new db and tables associated with it if they don't exist
func New() (DB, error) {
	db, err := sql.Open("sqlite3", "media.db")
	if err != nil {
		log.Fatal(err)
	}

	// ping database to verify connection
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	// create table
	_, err = db.Exec(createTable)
	if err != nil {
		return nil, err
	}
	return &Db{db}, nil
}

func (d *Db) WriteMeta(meta map[string]*fastdu.Meta) {
	var dupRows atomic.Uint64

	type job struct {
		file string
		meta *fastdu.Meta
	}

	numWorkers := 10
	numJobs := len(meta)

	// add all jobs to jobs channel - using channel buffering
	jobs := make(chan job, numJobs)
	for file, m := range meta {
		jobs <- job{file: file, meta: m}
	}
	close(jobs)

	var wg sync.WaitGroup
	// todo: user errGroup
	wg.Add(numWorkers)

	stmt, err := d.Prepare(insert)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for i := 0; i < numWorkers; i++ {
		go func() { // spawn worker
			defer wg.Done()
			for job := range jobs {
				m := job.meta
				filepath := strings.Join(m.Dups, ",")
				// log.Printf("processing %s", filepath)
				result, err := stmt.Exec(job.file, m.Size, m.Modtime,
					m.MIME.Type, m.MIME.Subtype, m.MIME.Value, m.Extension, filepath)
				if err != nil {
					log.Fatal(err)
				}
				rowsAffected, err := result.RowsAffected()
				if err != nil {
					log.Fatal(err)
				}
				if rowsAffected == 0 {
					dupRows.Add(1)
				}
			}
		}()
	}
	wg.Wait()
	log.Printf("skipped duplicate insertions count: %d", dupRows.Load())
	log.Println("Inserted to database successfully")
}

func (d *Db) Close() {
	d.Close()
}