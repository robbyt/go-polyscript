package loader

import (
	"io"
	"testing"

	"github.com/robbyt/go-polyscript/internal/helpers"
	"github.com/stretchr/testify/require"
)

func TestNewFromBytes(t *testing.T) {
	t.Parallel()

	t.Run("valid content", func(t *testing.T) {
		tests := []struct {
			name    string
			content []byte
			want    []byte
		}{
			{
				name:    "simple content",
				content: []byte(SimpleContent),
				want:    []byte(SimpleContent),
			},
			{
				name:    "trim whitespace",
				content: []byte("  content with spaces  "),
				want:    []byte("  content with spaces  "),
			},
			{
				name:    "multiline content",
				content: []byte(MultilineContent),
				want:    []byte(MultilineContent),
			},
			{
				name:    "mixed line endings",
				content: []byte("line1\nline2\r\nline3"),
				want:    []byte("line1\nline2\r\nline3"),
			},
			{
				name:    "special characters",
				content: []byte("function test(x) { return x * π; }"),
				want:    []byte("function test(x) { return x * π; }"),
			},
			{
				name:    "binary data",
				content: []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE},
				want:    []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				loader, err := NewFromBytes(tc.content)
				require.NoError(t, err)
				require.NotNil(t, loader)
				require.Equal(t, tc.want, loader.content)

				// Verify the URL includes the hash of the content
				expectedHash := helpers.SHA256(string(tc.want))[:8]
				require.Contains(t, loader.GetSourceURL().String(), expectedHash)

				// Verify loader with the helper function
				verifyLoader(t, loader, "bytes://inline/"+expectedHash)
			})
		}
	})

	t.Run("invalid content", func(t *testing.T) {
		tests := []struct {
			name    string
			content []byte
		}{
			{
				name:    "empty bytes",
				content: []byte{},
			},
			{
				name:    "only whitespace",
				content: []byte("   \n\t   "),
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				loader, err := NewFromBytes(tc.content)
				require.Error(t, err)
				require.ErrorIs(t, err, ErrScriptNotAvailable)
				require.Nil(t, loader)
			})
		}
	})
}

func TestFromBytes_GetReader(t *testing.T) {
	t.Parallel()

	t.Run("read content", func(t *testing.T) {
		content := []byte("test content\nwith multiple lines")
		loader, err := NewFromBytes(content)
		require.NoError(t, err)

		reader, err := loader.GetReader()
		require.NoError(t, err)

		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.Equal(t, content, data)
		require.NoError(t, reader.Close())
	})

	t.Run("multiple reads from same loader", func(t *testing.T) {
		content := []byte(FunctionContent)
		loader, err := NewFromBytes(content)
		require.NoError(t, err)

		// First read
		reader1, err := loader.GetReader()
		require.NoError(t, err)
		data1, err := io.ReadAll(reader1)
		require.NoError(t, err)
		require.Equal(t, content, data1)
		require.NoError(t, reader1.Close())

		// Second read
		reader2, err := loader.GetReader()
		require.NoError(t, err)
		data2, err := io.ReadAll(reader2)
		require.NoError(t, err)
		require.Equal(t, content, data2)
		require.NoError(t, reader2.Close())
	})

	t.Run("partial reads", func(t *testing.T) {
		content := []byte("line1\nline2\nline3\nline4\nline5")
		loader, err := NewFromBytes(content)
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
		require.Equal(t, []byte("line1\nline"), buf[:n])

		// Read the rest
		remaining, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.Equal(t, []byte("2\nline3\nline4\nline5"), remaining)
	})

	t.Run("binary data reads", func(t *testing.T) {
		content := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE}
		loader, err := NewFromBytes(content)
		require.NoError(t, err)

		reader, err := loader.GetReader()
		require.NoError(t, err)
		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.Equal(t, content, data)
		require.NoError(t, reader.Close())
	})
}

func TestFromBytes_GetSourceURL(t *testing.T) {
	t.Parallel()

	t.Run("source url", func(t *testing.T) {
		content := []byte(SimpleContent)
		loader, err := NewFromBytes(content)
		require.NoError(t, err)

		url := loader.GetSourceURL()
		require.NotNil(t, url)
		require.Equal(t, "bytes", url.Scheme)
		require.Equal(t, "inline", url.Host)

		// Verify it contains the hash prefix
		contentHash := helpers.SHA256(string(content))[:8]
		require.Equal(t, "/"+contentHash, url.Path)
		require.Contains(t, url.String(), "bytes://inline/"+contentHash)
	})

	t.Run("unique urls for different content", func(t *testing.T) {
		loader1, err := NewFromBytes([]byte("content one"))
		require.NoError(t, err)

		loader2, err := NewFromBytes([]byte("content two"))
		require.NoError(t, err)

		// URLs should be different
		require.NotEqual(t, loader1.GetSourceURL().String(), loader2.GetSourceURL().String())
	})

	t.Run("binary content hash", func(t *testing.T) {
		content := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE}
		loader, err := NewFromBytes(content)
		require.NoError(t, err)

		url := loader.GetSourceURL()
		require.NotNil(t, url)

		// Verify correct hash
		contentHash := helpers.SHA256(string(content))[:8]
		require.Contains(t, url.String(), contentHash)
	})
}

func TestFromBytes_String(t *testing.T) {
	t.Parallel()

	t.Run("string representation", func(t *testing.T) {
		tests := []struct {
			name        string
			content     []byte
			shouldMatch string
		}{
			{
				name:        "short content",
				content:     []byte("short"),
				shouldMatch: "loader.FromBytes{Bytes: 5}",
			},
			{
				name:        "longer content",
				content:     []byte("this is a longer piece of content with multiple words"),
				shouldMatch: "loader.FromBytes{Bytes: 53}",
			},
			{
				name:        "binary content",
				content:     []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE},
				shouldMatch: "loader.FromBytes{Bytes: 6}",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				loader, err := NewFromBytes(tc.content)
				require.NoError(t, err)

				result := loader.String()
				require.Equal(t, tc.shouldMatch, result)
			})
		}
	})
}

func TestFromBytes_ImplementsLoader(t *testing.T) {
	var _ Loader = (*FromBytes)(nil)
}
