package db

import (
	"database/sql"
	"log"

	"github.com/ajoyka/fdu/fastdu"
	_ "github.com/mattn/go-sqlite3"
)

const (
	createTable = `
CREATE TABLE IF NOT EXISTS media (
	name TEXT,
	size INTEGER,
	datetime DATETIME,
	mime_type TEXT,
	mime_subtype TEXT,
	mime_value TEXT,
	extension TEXT
)
`
	insert = `INSERT INTO media (name, size, datetime, mime_type, mime_subtype, mime_value, extension) 
 VALUES (?, ?, ?, ?, ?, ?, ?)`
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

	// create table
	_, err = db.Exec(createTable)
	if err != nil {
		return nil, err
	}
	return &Db{db}, nil
}

func (d *Db) WriteMeta(meta map[string]*fastdu.Meta) {
	stmt, err := d.Prepare(insert)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for file, m := range meta {
		_, err = stmt.Exec(file, m.Size, m.Modtime, m.MIME.Type, m.MIME.Subtype, m.MIME.Value, m.Extension)
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Println("Inserted to database successfully")
}

func (d *Db) Close() {
	d.Close()
}
