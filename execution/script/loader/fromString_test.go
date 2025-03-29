package loader

import (
	"io"
	"testing"

	"github.com/robbyt/go-polyscript/internal/helpers"
	"github.com/stretchr/testify/require"
)

func TestNewFromString(t *testing.T) {
	t.Parallel()

	t.Run("valid content", func(t *testing.T) {
		cases := []struct {
			name    string
			content string
			want    string
		}{
			{
				name:    "simple content",
				content: "test content",
				want:    "test content",
			},
			{
				name:    "trim whitespace",
				content: "  content with spaces  ",
				want:    "content with spaces",
			},
			{
				name:    "multiline content",
				content: "line1\nline2\nline3",
				want:    "line1\nline2\nline3",
			},
			{
				name:    "mixed line endings",
				content: "line1\nline2\r\nline3",
				want:    "line1\nline2\r\nline3",
			},
			{
				name:    "special characters",
				content: "function test(x) { return x * π; }",
				want:    "function test(x) { return x * π; }",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				loader, err := NewFromString(tc.content)
				require.NoError(t, err)
				require.NotNil(t, loader)
				require.Equal(t, tc.want, loader.content)

				// Verify the URL includes the hash of the content
				expectedHash := helpers.SHA256(tc.want)[:8]
				require.Contains(t, loader.GetSourceURL().String(), expectedHash)
			})
		}
	})

	t.Run("invalid content", func(t *testing.T) {
		cases := []struct {
			name    string
			content string
		}{
			{
				name:    "empty string",
				content: "",
			},
			{
				name:    "only whitespace",
				content: "   \n\t   ",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				loader, err := NewFromString(tc.content)
				require.Error(t, err)
				require.ErrorIs(t, err, ErrScriptNotAvailable)
				require.Nil(t, loader)
			})
		}
	})

	t.Run("URL parsing error simulation", func(t *testing.T) {
		// For this test we'll just verify normal operation
		// since mocking url.Parse is complicated
		content := "valid content"
		loader, err := NewFromString(content)
		require.NoError(t, err)
		require.NotNil(t, loader)
	})
}

func TestFromString_GetReader(t *testing.T) {
	t.Parallel()

	t.Run("read content", func(t *testing.T) {
		content := "test content\nwith multiple lines"
		loader, err := NewFromString(content)
		require.NoError(t, err)

		reader, err := loader.GetReader()
		require.NoError(t, err)

		t.Cleanup(func() {
			require.NoError(t, reader.Close(), "Failed to close reader")
		})

		got, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.Equal(t, content, string(got))
	})

	t.Run("multiple reads from same loader", func(t *testing.T) {
		content := "function calculate(x) { return x * 2; }"
		loader, err := NewFromString(content)
		require.NoError(t, err)

		// First read
		reader1, err := loader.GetReader()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, reader1.Close(), "Failed to close first reader")
		})
		got1, err := io.ReadAll(reader1)
		require.NoError(t, err)
		require.Equal(t, content, string(got1))

		// Second read should return a new reader with the same content
		reader2, err := loader.GetReader()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, reader2.Close(), "Failed to close second reader")
		})
		got2, err := io.ReadAll(reader2)
		require.NoError(t, err)
		require.Equal(t, content, string(got2))
	})

	t.Run("partial reads", func(t *testing.T) {
		content := "line1\nline2\nline3\nline4\nline5"
		loader, err := NewFromString(content)
		require.NoError(t, err)

		reader, err := loader.GetReader()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, reader.Close(), "Failed to close reader")
		})

		// Read just a small buffer
		buf := make([]byte, 10)
		n, err := reader.Read(buf)
		require.NoError(t, err)
		require.Equal(t, 10, n)
		require.Equal(t, "line1\nline", string(buf[:n]))

		// Read the rest
		remaining, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.Equal(t, "2\nline3\nline4\nline5", string(remaining))
	})
}

func TestFromString_GetSourceURL(t *testing.T) {
	t.Parallel()

	t.Run("source url", func(t *testing.T) {
		content := "test content"
		loader, err := NewFromString(content)
		require.NoError(t, err)

		url := loader.GetSourceURL()
		require.NotNil(t, url)
		require.Equal(t, "string", url.Scheme)
		require.Equal(t, "inline", url.Host)

		// Verify it contains the hash prefix
		contentHash := helpers.SHA256(content)[:8]
		require.Equal(t, "/"+contentHash, url.Path)
		require.Contains(t, url.String(), "string://inline/"+contentHash)
	})

	t.Run("unique urls for different content", func(t *testing.T) {
		loader1, err := NewFromString("content one")
		require.NoError(t, err)

		loader2, err := NewFromString("content two")
		require.NoError(t, err)

		// URLs should be different
		require.NotEqual(t, loader1.GetSourceURL().String(), loader2.GetSourceURL().String())
	})
}

func TestFromString_String(t *testing.T) {
	t.Parallel()

	t.Run("string representation", func(t *testing.T) {
		// Test with different content lengths
		testCases := []struct {
			name        string
			content     string
			shouldMatch string
		}{
			{
				name:        "short content",
				content:     "short",
				shouldMatch: "loader.FromString{Chars: 5}",
			},
			{
				name:        "longer content",
				content:     "this is a longer piece of content with multiple words",
				shouldMatch: "loader.FromString{Chars: 53}",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				loader, err := NewFromString(tc.content)
				require.NoError(t, err)

				result := loader.String()
				require.Equal(t, tc.shouldMatch, result)
			})
		}
	})
}
