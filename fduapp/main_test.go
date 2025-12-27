package main

import (
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileCount_Inc(t *testing.T) {
	fc := &fileCount{}

	fc.Inc(100)
	fc.Inc(200)
	fc.Inc(300)

	files, nbytes := fc.Get()
	assert.Equal(t, int64(3), files)
	assert.Equal(t, int64(600), nbytes)
}

func TestFileCount_Get(t *testing.T) {
	fc := &fileCount{
		files:  5,
		nbytes: 1000,
	}

	files, nbytes := fc.Get()
	assert.Equal(t, int64(5), files)
	assert.Equal(t, int64(1000), nbytes)
}

func TestFileCount_Concurrent(t *testing.T) {
	fc := &fileCount{}
	var wg sync.WaitGroup

	// Simulate concurrent increments
	numWorkers := 10
	incrementsPerWorker := 100

	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerWorker; j++ {
				fc.Inc(10)
			}
		}()
	}
	wg.Wait()

	files, nbytes := fc.Get()
	expectedFiles := int64(numWorkers * incrementsPerWorker)
	expectedBytes := int64(numWorkers * incrementsPerWorker * 10)

	assert.Equal(t, expectedFiles, files)
	assert.Equal(t, expectedBytes, nbytes)
}

func TestFileCount_ZeroValues(t *testing.T) {
	fc := &fileCount{}

	files, nbytes := fc.Get()
	assert.Equal(t, int64(0), files)
	assert.Equal(t, int64(0), nbytes)
}

func TestFileCount_LargeNumbers(t *testing.T) {
	fc := &fileCount{}

	// Test with large file sizes
	fc.Inc(1000000000) // 1GB
	fc.Inc(2000000000) // 2GB
	fc.Inc(3000000000) // 3GB

	files, nbytes := fc.Get()
	assert.Equal(t, int64(3), files)
	assert.Equal(t, int64(6000000000), nbytes)
}

func TestFileCount_ConcurrentGetAndInc(t *testing.T) {
	fc := &fileCount{}
	var wg sync.WaitGroup

	// Concurrent increments
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			fc.Inc(100)
		}
	}()

	// Concurrent reads
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			fc.Get()
		}
	}()

	wg.Wait()

	files, nbytes := fc.Get()
	assert.Equal(t, int64(100), files)
	assert.Equal(t, int64(10000), nbytes)
}

func TestCreateBackup_FileExists(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := tmpDir + "/test.txt"
	backupFile := testFile + ".bak"

	// Create a test file
	content := []byte("test content")
	err := os.WriteFile(testFile, content, 0644)
	assert.NoError(t, err)

	// Create backup
	createBackup(testFile)

	// Verify backup exists
	backupContent, err := os.ReadFile(backupFile)
	assert.NoError(t, err)
	assert.Equal(t, content, backupContent)
}

func TestCreateBackup_FileNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentFile := tmpDir + "/nonexistent.txt"

	// Should not panic or error when file doesn't exist
	createBackup(nonExistentFile)

	// Verify backup was not created
	backupFile := nonExistentFile + ".bak"
	_, err := os.ReadFile(backupFile)
	assert.Error(t, err) // Backup should not exist
}
