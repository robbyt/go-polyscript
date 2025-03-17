package loader

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewFromString(t *testing.T) {
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
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				loader, err := NewFromString(tc.content)
				require.NoError(t, err)
				require.NotNil(t, loader)
				require.Equal(t, tc.want, loader.content)
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
}

func TestFromString_GetReader(t *testing.T) {
	t.Run("read content", func(t *testing.T) {
		content := "test content\nwith multiple lines"
		loader, err := NewFromString(content)
		require.NoError(t, err)

		reader, err := loader.GetReader()
		require.NoError(t, err)

		t.Cleanup(func() {
			reader.Close()
		})

		got, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.Equal(t, content, string(got))
	})
}

func TestFromString_GetSourceURL(t *testing.T) {
	t.Run("source url", func(t *testing.T) {
		loader, err := NewFromString("test content")
		require.NoError(t, err)

		url := loader.GetSourceURL()
		require.NotNil(t, url)
		require.Equal(t, "string", url.Scheme)
		require.Equal(t, "string:", url.String())
	})
}
