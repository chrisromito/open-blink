package postgres

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetDevices(t *testing.T) {
	t.Skip("This is annoying")
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "valid_url",
			url:  testDatabaseURL,
			want: testDatabaseURL,
		},
		{
			name: "empty_url",
			url:  "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := assert.New(t)
			appDb := NewAppDb()
			defer appDb.Db.Close()
			a.NotNil(appDb)
		})
	}
}
