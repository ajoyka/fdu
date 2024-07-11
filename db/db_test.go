package db

import (
	"testing"

	"github.com/ajoyka/fdu/fastdu"
	_ "github.com/mattn/go-sqlite3"
)

func Test_findCommonPath(t *testing.T) {
	tests := []struct {
		name string
		dups []fastdu.Duplicate
		want string
	}{
		{
			name: "basic",
			dups: []fastdu.Duplicate{
				{
					Name: "/Volumes/Aswadhati/Drive/foobar/ home_macbook backup_june19_2015/Library/Desktop Pictures/.thumbnails/Flower 10.jpg",
					Size: 9436,
				},
				{
					Name: "/Volumes/Aswadhati/Drive/foobar/ home_macbook backup_june19_2015/Library/Desktop Pictures/.thumbnails/Flower 10.jpg",
					Size: 9436,
				},
				{
					Name: "/Volumes/Aswadhati/Drive/foobar/ home_macbook backup_june19_2015/Library/Desktop Pictures/Flower 10.jpg",
					Size: 22387196,
				},
				{
					Name: "/Volumes/Aswadhati/Drive/foobar/ home_macbook backup_june19_2015/Library/Desktop Pictures/Flower 10.jpg",
					Size: 22387196,
				},
			},
			want: "Flower 10.jpg",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findCommonPath(tt.dups); got != tt.want {
				t.Errorf("findCommonPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
