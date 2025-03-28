package loader

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewFromDisk(t *testing.T) {
	t.Run("valid paths", func(t *testing.T) {
		tempDir := t.TempDir()
		absPath := filepath.Join(tempDir, "test.js")

		cases := []struct {
			name     string
			path     string
			wantPath string
		}{
			{
				name:     "absolute path",
				path:     absPath,
				wantPath: "file://" + absPath,
			},
			{
				name:     "with file scheme",
				path:     "file://" + absPath,
				wantPath: "file://" + absPath,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				loader, err := NewFromDisk(tc.path)
				require.NoError(t, err)
				require.NotNil(t, loader)
				require.Equal(t, tc.wantPath, loader.path)
				require.Equal(t, "file", loader.sourceURL.Scheme)
			})
		}
	})

	t.Run("invalid schemes", func(t *testing.T) {
		cases := []struct {
			name string
			path string
		}{
			{
				name: "http scheme",
				path: "http://example.com/script.js",
			},
			{
				name: "https scheme",
				path: "https://example.com/script.js",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				loader, err := NewFromDisk(tc.path)
				require.Error(t, err)
				require.ErrorIs(t, err, ErrSchemeUnsupported)
				require.Nil(t, loader)
			})
		}
	})

	t.Run("relative paths", func(t *testing.T) {
		cases := []struct {
			name string
			path string
		}{
			{name: "relative path", path: "test.js"},
			{name: "current dir", path: "./test.js"},
			{name: "parent dir", path: "../test.js"},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				loader, err := NewFromDisk(tc.path)
				require.Error(t, err)
				require.ErrorIs(t, err, ErrScriptNotAvailable)
				require.Nil(t, loader)
			})
		}
	})

	t.Run("empty or invalid paths", func(t *testing.T) {
		cases := []struct {
			name string
			path string
		}{
			{name: "empty path", path: ""},
			{name: "dot path", path: "."},
			{name: "root path", path: "/"},
			{name: "windows root", path: "\\"},
			{name: "parent dir", path: "../"},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if tc.path == "\\" && runtime.GOOS != "windows" {
					t.Skip("Skipping Windows-specific test on non-Windows platform")
				}
				loader, err := NewFromDisk(tc.path)
				require.Error(t, err)
				require.ErrorIs(t, err, ErrScriptNotAvailable)
				require.Nil(t, loader)
			})
		}
	})

	t.Run("url parsing errors", func(t *testing.T) {
		loader, err := NewFromDisk("file://[invalid-url")
		require.Error(t, err)
		require.ErrorContains(t, err, "relative paths are not supported")
		require.Nil(t, loader)
	})

	t.Run("non-file scheme", func(t *testing.T) {
		tempDir := t.TempDir()
		absPath := filepath.Join(tempDir, "test.js")
		loader, err := NewFromDisk("http://" + absPath)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrSchemeUnsupported)
		require.Nil(t, loader)
	})
}

func TestFromDisk_GetReader(t *testing.T) {
	t.Parallel()
	t.Run("read file contents", func(t *testing.T) {
		// Setup test file
		tempDir := t.TempDir()
		testContent := "test content\nwith multiple lines"
		testFile := filepath.Join(tempDir, "test.risor")

		err := os.WriteFile(testFile, []byte(testContent), 0o644)
		require.NoError(t, err, "Failed to write test file")

		// Create loader
		loader, err := NewFromDisk(testFile)
		require.NoError(t, err, "Failed to create loader")

		// Get and read from reader
		reader, err := loader.GetReader()
		require.NoError(t, err, "Failed to get reader")

		// Ensure reader is closed after test
		t.Cleanup(func() {
			if reader != nil {
				reader.Close()
			}
		})

		// Read content
		content, err := io.ReadAll(reader)
		require.NoError(t, err, "Failed to read content")
		require.Equal(t, testContent, string(content), "Content mismatch")
	})
}

func TestFromDisk_GetSourceURL(t *testing.T) {
	t.Parallel()
	t.Run("valid source URL", func(t *testing.T) {
		// Setup test file
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.risor")

		// Create loader
		loader, err := NewFromDisk(testFile)
		require.NoError(t, err, "Failed to create loader")

		// Ensure source URL is set
		require.Equal(t, "file://"+testFile, loader.GetSourceURL().String())
	})
}
