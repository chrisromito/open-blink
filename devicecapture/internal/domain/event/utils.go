package event

import (
	"maps"
	"slices"
)

func LabelsEq(left []string, right []string) bool {
	// Clone and sort both slices before checking equality
	//ls := Uniq(left)
	//slices.Sort(ls)
	//rs := Uniq(right)
	//slices.Sort(rs)
	return slices.Equal(Uniq(left), Uniq(right))
}

func Uniq(labels []string) []string {
	next := make(map[string]bool)
	for _, l := range labels {
		next[l] = true
	}
	return slices.Sorted(maps.Keys(next))
}

// CombineLabels combine 2 sets of label strings into a slice of unique strings
func CombineLabels(left []string, right []string) []string {
	next := make(map[string]bool)
	for _, l := range left {
		next[l] = true
	}
	for _, r := range right {
		next[r] = true
	}
	// Return unique sorted keys
	return slices.Sorted(maps.Keys(next))
}
