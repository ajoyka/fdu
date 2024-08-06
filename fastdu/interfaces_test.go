package fastdu

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDirCount_AddFile(t *testing.T) {
	type fields struct {
		mu    sync.Mutex
		size  map[string]int64
		Meta  map[string]*Meta
		dList []duplicates
	}
	type args struct {
		dir   string
		fname string
	}
	tests := []struct {
		name    string
		args    args
		checker func(*DirCount)
	}{
		{
			name: "skip-thumbs", // skip this file
			args: args{
				dir:   "../testdata/Thumbs",
				fname: "skip.png",
			},
			checker: func(got *DirCount) {
				assert.NotContains(t, got.Meta, "skip.png")
			},
		},
		{
			name: "skip-eaDir", // skip this file
			args: args{
				dir:   "../testdata/@eaDir",
				fname: "skip.png",
			},
			checker: func(got *DirCount) {
				assert.NotContains(t, got.Meta, "skip.png")
			},
		},
		{
			name: "check-or-regex", // skip this file
			args: args{
				dir:   "../testdata/rep/ssd",
				fname: "skip.png",
			},
			checker: func(got *DirCount) {
				assert.NotContains(t, got.Meta, "skip.png")
			},
		},
		{
			name: "dont-skip-thumb", // don't skip
			args: args{
				dir:   "../testdata/Thumb",
				fname: "dont_skip.png",
			},
			checker: func(got *DirCount) {
				assert.Contains(t, got.Meta, "dont_skip.png")
			},
		},
	}
	d := &DirCount{
		Meta:  map[string]*Meta{},
		dList: []duplicates{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tt.args.dir, tt.args.fname)
			fInfo, _ := os.Stat(filePath)
			d.AddFile(tt.args.dir, fInfo)
			tt.checker(d)
			fmt.Printf("meta:%v\n", d)
		})
	}
}
