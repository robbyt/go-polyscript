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

func TestIsOnlyWhitespace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{
			name:     "empty input",
			input:    []byte{},
			expected: true,
		},
		{
			name:     "space only",
			input:    []byte("     "),
			expected: true,
		},
		{
			name:     "tabs only",
			input:    []byte("\t\t\t"),
			expected: true,
		},
		{
			name:     "newlines only",
			input:    []byte("\n\n\n"),
			expected: true,
		},
		{
			name:     "carriage returns only",
			input:    []byte("\r\r\r"),
			expected: true,
		},
		{
			name:     "form feeds only",
			input:    []byte("\f\f\f"),
			expected: true,
		},
		{
			name:     "vertical tabs only",
			input:    []byte("\v\v\v"),
			expected: true,
		},
		{
			name:     "mixed whitespace",
			input:    []byte(" \t\n\r\f\v"),
			expected: true,
		},
		{
			name:     "contains non-whitespace",
			input:    []byte(" \t a \n"),
			expected: false,
		},
		{
			name:     "single non-whitespace",
			input:    []byte("x"),
			expected: false,
		},
		{
			name:     "non-ascii character",
			input:    []byte(" \t π \n"),
			expected: false,
		},
		{
			name:     "control characters",
			input:    []byte{0x01, 0x02, 0x03},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isOnlyWhitespace(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestHasBinaryCharacters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{
			name:     "empty input",
			input:    []byte{},
			expected: false,
		},
		{
			name:     "plain text",
			input:    []byte("Hello, world!"),
			expected: false,
		},
		{
			name:     "text with newlines",
			input:    []byte("Line 1\nLine 2\nLine 3"),
			expected: false,
		},
		{
			name:     "text with tabs",
			input:    []byte("Column1\tColumn2\tColumn3"),
			expected: false,
		},
		{
			name:     "text with carriage returns",
			input:    []byte("Windows\r\nLine endings"),
			expected: false,
		},
		{
			name:     "contains null byte",
			input:    []byte{'H', 'e', 'l', 'l', 'o', 0, 'w', 'o', 'r', 'l', 'd'},
			expected: true,
		},
		{
			name:     "has control character",
			input:    []byte{'T', 'e', 's', 't', 0x01, '!'},
			expected: true,
		},
		{
			name:     "leading null byte",
			input:    []byte{0, 'D', 'a', 't', 'a'},
			expected: true,
		},
		{
			name:     "trailing null byte",
			input:    []byte{'D', 'a', 't', 'a', 0},
			expected: true,
		},
		{
			name:     "control characters below 32",
			input:    []byte{0x02, 0x05, 0x10},
			expected: true,
		},
		{
			name:     "allowed control chars (tab, CR, LF)",
			input:    []byte{'\t', '\r', '\n'},
			expected: false,
		},
		{
			name:     "binary with printable chars",
			input:    []byte{0xFF, 0xFE, 'A', 'B', 0x00},
			expected: true,
		},
		{
			name:     "extended ASCII",
			input:    []byte{0x80, 0x90, 0xA0},
			expected: false, // Extended ASCII isn't considered binary
		},
		{
			name:     "UTF-8 characters",
			input:    []byte("UTF-8 π Æ ¢ €"),
			expected: false, // UTF-8 isn't considered binary
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := hasBinaryCharacters(tc.input)
			require.Equal(t, tc.expected, result, "Unexpected result for input %v", tc.input)
		})
	}
}

func TestWhitespaceAndBinaryInteraction(t *testing.T) {
	t.Parallel()

	// These test cases focus on how the two functions interact together
	tests := []struct {
		name           string
		input          []byte
		isBinary       bool
		isWhitespace   bool
		shouldBeLoaded bool
	}{
		{
			name:           "regular text",
			input:          []byte("Regular text"),
			isBinary:       false,
			isWhitespace:   false,
			shouldBeLoaded: true,
		},
		{
			name:           "only whitespace",
			input:          []byte("  \t\n  "),
			isBinary:       false,
			isWhitespace:   true,
			shouldBeLoaded: false, // Whitespace-only content isn't loaded
		},
		{
			name:           "binary data",
			input:          []byte{0x00, 0x01, 0xFF},
			isBinary:       true,
			isWhitespace:   false, // Irrelevant for binary
			shouldBeLoaded: true,  // Binary always loads regardless of whitespace
		},
		{
			name:           "binary with some whitespace",
			input:          []byte{' ', '\t', 0x00, ' '},
			isBinary:       true,
			isWhitespace:   false, // Not only whitespace due to binary check
			shouldBeLoaded: true,  // Binary always loads
		},
		{
			name:           "empty input",
			input:          []byte{},
			isBinary:       false,
			isWhitespace:   true, // Empty is considered whitespace
			shouldBeLoaded: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isBinary := hasBinaryCharacters(tc.input)
			isWhitespace := isOnlyWhitespace(tc.input)

			require.Equal(t, tc.isBinary, isBinary)
			require.Equal(t, tc.isWhitespace, isWhitespace)

			// Test the loading condition used in FromBytes
			willBeLoaded := len(tc.input) != 0 && (isBinary || !isWhitespace)
			require.Equal(t, tc.shouldBeLoaded, willBeLoaded)
		})
	}
}

func TestFromBytes_ImplementsLoader(t *testing.T) {
	var _ Loader = (*FromBytes)(nil)
}
