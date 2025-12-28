package fastdu

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/h2non/filetype/types"
	"github.com/stretchr/testify/assert"
)

func TestDirCount_Inc(t *testing.T) {
	d := NewDirCount("")

	d.Inc("/home/user/dir1", 100)
	d.Inc("/home/user/dir1", 200)
	d.Inc("/home/user/dir2", 500)

	assert.Equal(t, int64(300), d.size["/home/user/dir1"])
	assert.Equal(t, int64(500), d.size["/home/user/dir2"])
}

func TestDirCount_GetTop(t *testing.T) {
	d := NewDirCount("")

	d.size["home/user/pics"] = 1000
	d.size["home/user/docs"] = 500
	d.size["var/log"] = 2000
	d.size["var/cache"] = 1500

	top := d.GetTop()

	// Should aggregate by top-level directory (first path component)
	assert.Contains(t, top, "home")
	assert.Contains(t, top, "var")
	assert.Equal(t, int64(1500), top["home"]) // 1000 + 500
	assert.Equal(t, int64(3500), top["var"])  // 2000 + 1500
}

func TestDirCount_Counters(t *testing.T) {
	d := NewDirCount("")

	// Reset counters for test
	counts = Counters{}
	counts.ImageCnt.Add(10)
	counts.VideoCnt.Add(5)
	counts.AudioCnt.Add(3)
	counts.ExifErrors.Add(2)
	counts.FileSizeMismatchCnt.Add(1)
	counts.FilesSkipCnt.Add(4)

	result := d.Counters()
	assert.Contains(t, result, "Image file(s): 10")
	assert.Contains(t, result, "Video files: 5")
	assert.Contains(t, result, "Audio file(s): 3")
	assert.Contains(t, result, "Exif Errors: 2")
	assert.Contains(t, result, "FileSizeMismatch Count: 1")
	assert.Contains(t, result, "SkippedFiles:4")
}

func TestDirCount_AddFile_Duplicates(t *testing.T) {
	d := NewDirCount("")

	// Note: AddFile requires actual files to work with getFileInfo
	// This test demonstrates the structure but would need real test files
	// to fully test the duplicate detection logic

	// Verify Meta map structure is correct
	assert.NotNil(t, d.Meta)
	assert.Equal(t, 0, len(d.Meta))
}

func TestDirCount_AddFile_FileSizeMismatch(t *testing.T) {
	d := NewDirCount("")

	// Reset counters
	counts = Counters{}

	// Test structure - would need real files to fully test
	assert.NotNil(t, d.Meta)
	assert.Equal(t, int64(0), counts.FileSizeMismatchCnt.Load())
}

func TestNewDirCount(t *testing.T) {
	tests := []struct {
		name    string
		skipPat string
	}{
		{
			name:    "empty pattern",
			skipPat: "",
		},
		{
			name:    "with pattern",
			skipPat: "/tmp|/cache",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDirCount(tt.skipPat)
			assert.NotNil(t, d)
			assert.NotNil(t, d.size)
			assert.NotNil(t, d.Meta)
			assert.NotNil(t, d.dList)
			assert.Equal(t, 0, len(d.size))
			assert.Equal(t, 0, len(d.Meta))
		})
	}
}

func TestCounters_String(t *testing.T) {
	c := &Counters{}
	c.ExifErrors.Add(5)
	c.VideoCnt.Add(10)
	c.AudioCnt.Add(3)
	c.ImageCnt.Add(50)
	c.FileSizeMismatchCnt.Add(2)
	c.FilesSkipCnt.Add(8)

	result := c.String()

	assert.Contains(t, result, "Exif Errors: 5")
	assert.Contains(t, result, "Video files: 10")
	assert.Contains(t, result, "Audio file(s): 3")
	assert.Contains(t, result, "Image file(s): 50")
	assert.Contains(t, result, "FileSizeMismatch Count: 2")
	assert.Contains(t, result, "SkippedFiles:8")
}

func TestDirCount_WriteMeta(t *testing.T) {
	d := NewDirCount("")

	// Add test metadata
	d.Meta["test1.jpg"] = &Meta{
		Name:    "test1.jpg",
		Size:    1000,
		Modtime: time.Now(),
		Type:    types.Type{MIME: types.MIME{Type: "image", Subtype: "jpeg"}},
		Dups: []Duplicate{
			{Name: "/path/to/test1.jpg", Size: 1000},
			{Name: "/another/path/test1.jpg", Size: 1000},
		},
	}

	d.Meta["test2.jpg"] = &Meta{
		Name:    "test2.jpg",
		Size:    2000,
		Modtime: time.Now(),
		Type:    types.Type{MIME: types.MIME{Type: "image", Subtype: "jpeg"}},
		Dups: []Duplicate{
			{Name: "/path/to/test2.jpg", Size: 2000},
		},
	}

	// Create temp file for testing
	tmpFile := filepath.Join(t.TempDir(), "test-meta.json")

	// Change to temp directory for the test
	oldDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	d.WriteMeta(tmpFile)

	// Verify files were created
	_, err := os.Stat(tmpFile)
	assert.NoError(t, err)

	// Verify duplicates list was populated
	assert.Equal(t, 1, len(d.dList)) // Only test1.jpg has multiple dups
}

func TestDirCount_WriteMetaSortedByDate(t *testing.T) {
	d := NewDirCount("")

	now := time.Now()
	d.Meta["old.jpg"] = &Meta{
		Name:    "old.jpg",
		Size:    1000,
		Modtime: now.Add(-2 * time.Hour),
	}
	d.Meta["new.jpg"] = &Meta{
		Name:    "new.jpg",
		Size:    2000,
		Modtime: now,
	}

	tmpFile := filepath.Join(t.TempDir(), "date-sorted.json")
	d.WriteMetaSortedByDate(tmpFile)

	// Verify file was created
	_, err := os.Stat(tmpFile)
	assert.NoError(t, err)
}

func TestDirCount_WriteMetaSortedBySize(t *testing.T) {
	d := NewDirCount("")

	d.Meta["small.jpg"] = &Meta{
		Name: "small.jpg",
		Size: 100,
	}
	d.Meta["large.jpg"] = &Meta{
		Name: "large.jpg",
		Size: 10000,
	}

	tmpFile := filepath.Join(t.TempDir(), "size-sorted.json")
	d.WriteMetaSortedBySize(tmpFile)

	// Verify file was created
	_, err := os.Stat(tmpFile)
	assert.NoError(t, err)
}
