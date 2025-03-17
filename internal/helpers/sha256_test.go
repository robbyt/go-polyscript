package helpers

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type errorReader struct{}

func (r *errorReader) Read(p []byte) (int, error) {
	return 0, errors.New("forced read error")
}

func TestSHA256(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "empty string",
			in:   "",
			want: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name: "basic string",
			in:   "hello world",
			want: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SHA256(tt.in)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestSHA256Reader(t *testing.T) {
	t.Parallel()
	t.Run("table driven tests", func(t *testing.T) {
		tests := []struct {
			name    string
			input   io.Reader
			want    string
			wantErr bool
		}{
			{
				name:    "empty string reader",
				input:   strings.NewReader(""),
				want:    "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				wantErr: false,
			},
			{
				name:    "basic string input",
				input:   strings.NewReader("hello world"),
				want:    "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
				wantErr: false,
			},
			{
				name:    "error case",
				input:   &errorReader{},
				want:    "",
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := SHA256Reader(tt.input)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("large content", func(t *testing.T) {
		// Create 1MB of repeated data
		data := bytes.Repeat([]byte("a"), 1024*1024)
		reader := bytes.NewReader(data)

		hash, err := SHA256Reader(reader)
		require.NoError(t, err)

		// Pre-calculated SHA256 hash of 1MB of 'a' characters
		expected := "9bc1b2a288b26af7257a36277ae3816a7d4f16e89c1e7e77d0a5c48bad62b360"
		require.Equal(t, expected, hash)
	})
}
