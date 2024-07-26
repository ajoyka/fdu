package db

import (
	"testing"

	"github.com/ajoyka/fdu/fastdu"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func Test_findCommonPath(t *testing.T) {
	tests := []struct {
		name       string
		dups       []fastdu.Duplicate
		wantSuffix string
		wantCommon string
	}{
		{
			name: "basic",
			dups: []fastdu.Duplicate{
				{
					Name: "/a/b/Drive/foobar/c/Desktop Pictures/.thumbnails/Flower 10.jpg",
					Size: 9436,
				},
				{
					Name: "/a/b/Drive/foobar/c/Desktop Pictures/Flower 10.jpg",
					Size: 22387196,
				},
			},
			wantSuffix: "Flower 10.jpg",
			wantCommon: "/a/b/Drive/foobar/c/Desktop Pictures",
		},
		{
			name: "common-path-in-middle",
			dups: []fastdu.Duplicate{
				{
					Name: "/x/y/Drive/foobar/c/Desktop Pictures/.thumbnails/Flower 10.jpg",
					Size: 9436,
				},
				{
					Name: "/m/n/Drive/foobar/c/Desktop Pictures/Flower 10.jpg",
					Size: 22387196,
				},
			},
			wantSuffix: "Flower 10.jpg",
			wantCommon: "/Drive/foobar/c/Desktop Pictures",
		},
		{
			name: "no-common-path",
			dups: []fastdu.Duplicate{
				{
					Name: "/a/b/Drive/foobar/c/Desktop Pictures/.thumbnails/Flower 10.jpg",
					Size: 9436,
				},
				{
					Name: "/x/y/z/m/n/o/Flower 10.jpg",
					Size: 22387196,
				},
			},
			wantSuffix: "Flower 10.jpg",
			wantCommon: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCPath, gotMaxPath := findCommonPath(tt.dups)
			assert.Equal(t, tt.wantSuffix, gotCPath)
			assert.Equal(t, tt.wantCommon, gotMaxPath)
		})
	}
}
