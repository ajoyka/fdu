package fastdu

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSortedKeys(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]int64
		want []string
	}{
		{
			name: "basic sorting",
			m: map[string]int64{
				"small":  100,
				"medium": 500,
				"large":  1000,
			},
			want: []string{"large", "medium", "small"}, // sorted by value descending
		},
		{
			name: "empty map",
			m:    map[string]int64{},
			want: nil, // SortedKeys returns nil for empty map
		},
		{
			name: "single element",
			m: map[string]int64{
				"only": 42,
			},
			want: []string{"only"},
		},
		{
			name: "equal values",
			m: map[string]int64{
				"a": 100,
				"b": 100,
				"c": 100,
			},
			want: nil, // order is not guaranteed for equal values, just check length
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SortedKeys(tt.m)
			if tt.want == nil {
				assert.Equal(t, len(tt.m), len(got))
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestSortedMetaByDate(t *testing.T) {
	now := time.Now()
	meta1 := &Meta{Name: "file1.jpg", Modtime: now.Add(-2 * time.Hour)}
	meta2 := &Meta{Name: "file2.jpg", Modtime: now.Add(-1 * time.Hour)}
	meta3 := &Meta{Name: "file3.jpg", Modtime: now}

	tests := []struct {
		name  string
		metas []*Meta
		want  []*Meta
	}{
		{
			name:  "sort by date ascending",
			metas: []*Meta{meta3, meta1, meta2},
			want:  []*Meta{meta1, meta2, meta3},
		},
		{
			name:  "already sorted",
			metas: []*Meta{meta1, meta2, meta3},
			want:  []*Meta{meta1, meta2, meta3},
		},
		{
			name:  "empty slice",
			metas: []*Meta{},
			want:  []*Meta{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sorted := SortedMetaByDate(tt.metas)
			// Sort in place
			for i := 0; i < sorted.Len()-1; i++ {
				for j := i + 1; j < sorted.Len(); j++ {
					if sorted.Less(j, i) {
						sorted.Swap(i, j)
					}
				}
			}
			assert.Equal(t, tt.want, tt.metas)
		})
	}
}

func TestSortedFileBySize(t *testing.T) {
	meta1 := &Meta{Name: "small.jpg", Size: 100}
	meta2 := &Meta{Name: "medium.jpg", Size: 500}
	meta3 := &Meta{Name: "large.jpg", Size: 1000}

	tests := []struct {
		name  string
		metas []*Meta
		want  []*Meta
	}{
		{
			name:  "sort by size ascending",
			metas: []*Meta{meta3, meta1, meta2},
			want:  []*Meta{meta1, meta2, meta3},
		},
		{
			name:  "already sorted",
			metas: []*Meta{meta1, meta2, meta3},
			want:  []*Meta{meta1, meta2, meta3},
		},
		{
			name:  "empty slice",
			metas: []*Meta{},
			want:  []*Meta{},
		},
		{
			name:  "same size",
			metas: []*Meta{{Name: "a.jpg", Size: 100}, {Name: "b.jpg", Size: 100}},
			want:  []*Meta{{Name: "a.jpg", Size: 100}, {Name: "b.jpg", Size: 100}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sorted := SortedFileBySize(tt.metas)
			// Sort in place
			for i := 0; i < sorted.Len()-1; i++ {
				for j := i + 1; j < sorted.Len(); j++ {
					if sorted.Less(j, i) {
						sorted.Swap(i, j)
					}
				}
			}
			assert.Equal(t, tt.want, tt.metas)
		})
	}
}

func TestSortedMap(t *testing.T) {
	sm := &sortedMap{
		m: map[string]int64{
			"a": 100,
			"b": 500,
			"c": 200,
		},
		keys: []string{"a", "b", "c"},
	}

	assert.Equal(t, 3, sm.Len())
	assert.True(t, sm.Less(0, 1))  // 100 < 500
	assert.False(t, sm.Less(1, 0)) // 500 > 100

	sm.Swap(0, 1)
	assert.Equal(t, []string{"b", "a", "c"}, sm.keys)
}
