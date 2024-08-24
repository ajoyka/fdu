package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	// "github.com/google/gops/agent"

	fdb "github.com/ajoyka/fdu/db"
	"github.com/ajoyka/fdu/fastdu"
	_ "github.com/mattn/go-sqlite3"
)

// Create date based dirs and copy over files from source dirs

const (
	mediaDB = "../fduapp/media.db" // todo: make it command line option
)

var (
	outDirPrefix = flag.String("p", ".", "Prefix root directory path to create output directory. Default is to use current directory")
)

type job struct {
	dir string
	src string
	dst string
}

func main() {
	// import profiling agent
	// Start the gops agent, report errors if any
	// if err := agent.Listen(agent.Options{}); err != nil {
	// 	log.Fatal(err)
	// }

	// Check link for avoiding db lock errors: https://github.com/mattn/go-sqlite3?tab=readme-ov-file#faq
	flag.Parse()
	fmt.Printf("outp->%s\n", *outDirPrefix)
	dsn := fmt.Sprintf("file:%s?cache=shared", mediaDB)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		log.Fatal(err)
	}

	// ping database to verify connection
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	query := `select %s from media where size > 20000 and mime_type = 'image' or mime_type = 'video' ;`
	query = fmt.Sprintf(query, fdb.MediaDBCols)
	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("failed to query %s got error %v", query, err)
	}

	jobs := make(chan job)
	// todo: find largest file path in slice
	// todo: don't create dir if it exists by keeping them in hashmap
	go func() {
		defer close(jobs)
		createCopyJobs(rows, jobs)
	}()

	numWorkers := 10
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for job := range jobs {
				dirPath := job.dir
				srcPath := job.src
				dstPath := job.dst
				fmt.Printf("%s->%s\n", srcPath, dstPath)
				if err := os.MkdirAll(dirPath, 0755); err != nil {
					log.Fatal(err)
				}
				if err := copyFile(srcPath, dstPath); err != nil {
					log.Fatal(err)
				}
			}

		}()
	}
	wg.Wait()
}

func createCopyJobs(rows *sql.Rows, jobsOut chan<- job) {
	for rows.Next() {
		var name, size, mime_type,
			mime_subtype, mime_value, extension,
			suffix_common_path, max_common_path, filepath, exif_json string
		var datetime time.Time
		var exif_datetime_original sql.NullTime
		var count, file_size_mismatch int
		err := rows.Scan(&name, &size, &datetime, &exif_datetime_original,
			&mime_type, &mime_subtype, &mime_value, &extension, &count,
			&file_size_mismatch, &suffix_common_path, &max_common_path, &filepath, &exif_json,
		)
		if err != nil {
			log.Fatal(err)
		}
		var dups []fastdu.Duplicate
		err = json.Unmarshal([]byte(filepath), &dups)
		if err != nil {
			log.Fatalf("name: %s, filepath: %s, %v", name, filepath, err)
		}
		// fmt.Printf("%s, %d, %v, %v\n", name, count, datetime, dups)

		year := datetime.Year()
		month := datetime.Month()
		day := datetime.Day()

		dirPath := fmt.Sprintf("%s/%d/%02d/%02d", *outDirPrefix, year, month, day)
		srcPath := getOriginalIfExists(dups)
		dstPath := dirPath + "/" + name

		jobsOut <- job{
			dir: dirPath,
			src: srcPath,
			dst: dstPath,
		}
		fmt.Printf("%s->%s\n", srcPath, dstPath)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			log.Fatal(err)
		}
		if err := copyFile(srcPath, dstPath); err != nil {
			log.Fatal(err)
		}

	}
}

func getOriginalIfExists(dups []fastdu.Duplicate) string {
	for _, dup := range dups {
		if strings.Contains(dup.Name, "Originals") { // get unedited photo from iphoto library
			return dup.Name
		}
	}
	return dups[0].Name
}

func copyFile(srcFile, dstFile string) error {
	src, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstFile)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}
	err = dst.Sync()
	if err != nil {
		return err
	}
	return nil
}
