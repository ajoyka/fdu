package fastdu

import (
	"sort"
)

// implement sort of map which contains files and sizes

type sortedMap struct {
	m    map[string]int64
	keys []string
}

func (s *sortedMap) Len() int {
	return len(s.m)
}

func (s *sortedMap) Less(i, j int) bool {
	return s.m[s.keys[i]] < s.m[s.keys[j]]
}

func (s *sortedMap) Swap(i, j int) {
	s.keys[i], s.keys[j] = s.keys[j], s.keys[i]
}

// SortedMetaByDate interface types for SortedMetaByDate
type SortedMetaByDate []*Meta

func (s SortedMetaByDate) Len() int           { return len(s) }
func (s SortedMetaByDate) Less(i, j int) bool { return s[i].Modtime.Before(s[j].Modtime) }
func (s SortedMetaByDate) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// SortedFileBySize sorts files by size held in metadata
type SortedFileBySize []*Meta

func (s SortedFileBySize) Len() int           { return len(s) }
func (s SortedFileBySize) Less(i, j int) bool { return s[i].Size < s[j].Size }
func (s SortedFileBySize) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// SortedKeys takes a dictionary and returns a sorted slice of keys
func SortedKeys(m map[string]int64) []string {
	sm := &sortedMap{}
	sm.m = m

	// collect all keys that will eventually be sorted by value in m
	for key, _ := range m {
		//fmt.Println("key", key)
		sm.keys = append(sm.keys, key)
	}
	//fmt.Println("sorted ", sm.keys)

	sort.Sort(sort.Reverse(sm))
	//sort.Sort(sm)
	return sm.keys
}
