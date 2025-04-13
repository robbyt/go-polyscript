package loader

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/robbyt/go-polyscript/internal/helpers"
	"github.com/stretchr/testify/require"
)

// ErrorReader is a mock reader that returns an error
type ErrorReader struct{}

func (r ErrorReader) Read(p []byte) (int, error) {
	return 0, errors.New("mock read error")
}

func TestNewFromIoReader(t *testing.T) {
	t.Parallel()

	t.Run("valid reader", func(t *testing.T) {
		tests := []struct {
			name       string
			reader     io.Reader
			sourceName string
			want       string
		}{
			{
				name:       "string reader with simple content",
				reader:     strings.NewReader(SimpleContent),
				sourceName: "test",
				want:       SimpleContent,
			},
			{
				name:       "bytes reader with multiline content",
				reader:     bytes.NewReader([]byte(MultilineContent)),
				sourceName: "multiline",
				want:       MultilineContent,
			},
			{
				name:       "special characters content",
				reader:     strings.NewReader("function test(x) { return x * π; }"),
				sourceName: "special",
				want:       "function test(x) { return x * π; }",
			},
			{
				name:       "no source name provided",
				reader:     strings.NewReader("content without source name"),
				sourceName: "",
				want:       "content without source name",
			},
		}

		for _, tc := range tests {
			tc := tc // Capture range variable
			t.Run(tc.name, func(t *testing.T) {
				loader, err := NewFromIoReader(tc.reader, tc.sourceName)
				require.NoError(t, err)
				require.NotNil(t, loader)
				require.Equal(t, tc.want, string(loader.content))

				// Verify the URL includes the hash of the content
				expectedHash := helpers.SHA256(tc.want)[:8]
				require.Contains(t, loader.GetSourceURL().String(), expectedHash)

				// Verify source name in URL if provided
				urlStr := loader.GetSourceURL().String()
				if tc.sourceName != "" {
					require.Contains(t, urlStr, tc.sourceName)
					require.Contains(t, urlStr, "reader://"+tc.sourceName+"/")
				} else {
					require.Contains(t, urlStr, "reader://unnamed/")
				}

				// Use GetReader to verify the content
				reader, err := loader.GetReader()
				require.NoError(t, err)
				content, err := io.ReadAll(reader)
				require.NoError(t, err)
				require.Equal(t, tc.want, string(content))
				require.NoError(t, reader.Close())

				// Test String() method
				strRep := loader.String()
				require.Contains(t, strRep, "loader.FromIoReader")
				require.Contains(t, strRep, fmt.Sprintf("Bytes: %d", len(tc.want)))
				require.Contains(t, strRep, "Source: "+urlStr)
			})
		}
	})

	t.Run("invalid reader", func(t *testing.T) {
		tests := []struct {
			name       string
			reader     io.Reader
			sourceName string
		}{
			{
				name:       "nil reader",
				reader:     nil,
				sourceName: "test",
			},
			{
				name:       "empty reader",
				reader:     strings.NewReader(""),
				sourceName: "empty",
			},
			{
				name:       "whitespace only",
				reader:     strings.NewReader("   \n\t   "),
				sourceName: "whitespace",
			},
			{
				name:       "reader with error",
				reader:     ErrorReader{},
				sourceName: "error",
			},
		}

		for _, tc := range tests {
			tc := tc // Capture range variable
			t.Run(tc.name, func(t *testing.T) {
				loader, err := NewFromIoReader(tc.reader, tc.sourceName)
				require.Error(t, err)
				if tc.reader == nil || tc.name == "empty reader" || tc.name == "whitespace only" {
					require.ErrorIs(t, err, ErrScriptNotAvailable)
				}
				require.Nil(t, loader)
			})
		}
	})
}

func TestFromIoReader_GetReader(t *testing.T) {
	t.Parallel()

	t.Run("read content multiple times", func(t *testing.T) {
		content := "test content\nwith multiple lines"
		loader, err := NewFromIoReader(strings.NewReader(content), "test")
		require.NoError(t, err)

		// First read
		reader1, err := loader.GetReader()
		require.NoError(t, err)
		content1, err := io.ReadAll(reader1)
		require.NoError(t, err)
		require.Equal(t, content, string(content1))
		require.NoError(t, reader1.Close())

		// Second read should work the same way
		reader2, err := loader.GetReader()
		require.NoError(t, err)
		content2, err := io.ReadAll(reader2)
		require.NoError(t, err)
		require.Equal(t, content, string(content2))
		require.NoError(t, reader2.Close())
	})

	t.Run("different readers are independent", func(t *testing.T) {
		content := "test content for independent readers"
		loader, err := NewFromIoReader(strings.NewReader(content), "independent")
		require.NoError(t, err)

		// Get two readers
		reader1, err := loader.GetReader()
		require.NoError(t, err)
		reader2, err := loader.GetReader()
		require.NoError(t, err)

		// Read partial content from first reader
		buf1 := make([]byte, 5)
		n1, err := reader1.Read(buf1)
		require.NoError(t, err)
		require.Equal(t, 5, n1)
		require.Equal(t, "test ", string(buf1))

		// Read full content from second reader
		content2, err := io.ReadAll(reader2)
		require.NoError(t, err)
		require.Equal(t, content, string(content2))

		// Continue reading from first reader
		remaining1, err := io.ReadAll(reader1)
		require.NoError(t, err)
		require.Equal(t, content[5:], string(remaining1))

		require.NoError(t, reader1.Close())
		require.NoError(t, reader2.Close())
	})
}

func TestFromIoReader_GetSourceURL(t *testing.T) {
	t.Parallel()

	content := "test content"
	sourceName := "test-source"
	expectedHash := helpers.SHA256(content)[:8]
	expectedPrefix := "reader://" + sourceName + "/"

	loader, err := NewFromIoReader(strings.NewReader(content), sourceName)
	require.NoError(t, err)

	sourceURL := loader.GetSourceURL()
	require.NotNil(t, sourceURL)
	require.Equal(t, "reader", sourceURL.Scheme)
	require.Contains(t, sourceURL.String(), expectedPrefix)
	require.Contains(t, sourceURL.String(), expectedHash)
}
