package event

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Combine_Labels(t *testing.T) {
	a := assert.New(t)

	result := CombineLabels(
		[]string{"person"},
		[]string{"car", "person"},
	)

	a.Equal([]string{"car", "person"}, result)
}


func Test_Empty_Labels_Eq(t *testing.T) {
	a := assert.New(t)

	cases := []struct{
		left []string
		right []string
		eq bool
		message string
	}{
		{
			[]string{},
			[]string{},
			true,
			"empty slices are equal",
		},
		{
			[]string{""},
			[]string{""},
			true,
			"slices of empty strings are equal",
		},
		{
			[]string{"car"},
			[]string{"car"},
			true,
			"two slices of car labels are equal",
		},
		{
			[]string{"car"},
			[]string{"person"},
			false,
			"['car'] != ['person']",
		},
		{
			[]string{"car"},
			[]string{"car", "person"},
			false,
			"['car'] != ['car', 'person']",
		},
	}
	for _, c := range cases {
		result := LabelsEq(c.left, c.right)
		if c.eq {
			a.True(result, c.message)
		} else {
			a.False(result, c.message)
		}
	}
}