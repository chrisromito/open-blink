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
