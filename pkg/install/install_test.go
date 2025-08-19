package install

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveInstallDir(t *testing.T) {
	tests := []struct {
		name     string
		binDir   string
		setupEnv map[string]string
		want     string
		wantErr  bool
	}{
		{
			name:   "explicit directory",
			binDir: "/usr/local/bin",
			want:   "/usr/local/bin",
		},
		{
			name:   "expand home directory",
			binDir: "~/bin",
			setupEnv: map[string]string{
				"HOME": "/home/user",
			},
			want: "/home/user/bin",
		},
		{
			name:   "expand environment variable",
			binDir: "${CUSTOM_BIN}/tools",
			setupEnv: map[string]string{
				"CUSTOM_BIN": "/opt/bin",
			},
			want: "/opt/bin/tools",
		},
		{
			name:   "default with BINSTALLER_BIN set",
			binDir: "",
			setupEnv: map[string]string{
				"BINSTALLER_BIN": "/custom/bin",
			},
			want: "/custom/bin",
		},
		{
			name:   "default with HOME set",
			binDir: "",
			setupEnv: map[string]string{
				"HOME": "/home/user",
			},
			want: "/home/user/.local/bin",
		},
		{
			name:   "default with no HOME",
			binDir: "",
			setupEnv: map[string]string{
				"HOME": "", // Explicitly set HOME to empty
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore environment
			origEnv := make(map[string]string)
			for k := range tt.setupEnv {
				origEnv[k] = os.Getenv(k)
				os.Setenv(k, tt.setupEnv[k])
			}
			defer func() {
				for k, v := range origEnv {
					if v == "" {
						os.Unsetenv(k)
					} else {
						os.Setenv(k, v)
					}
				}
			}()

			got, err := ResolveInstallDir(tt.binDir)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInstallBinary(t *testing.T) {
	tests := []struct {
		name           string
		setupSource    func(t *testing.T) string
		setupTarget    func(t *testing.T) string
		targetName     string
		wantErr        bool
		validateResult func(t *testing.T, targetPath string)
	}{
		{
			name: "install new binary",
			setupSource: func(t *testing.T) string {
				source := filepath.Join(t.TempDir(), "source-binary")
				require.NoError(t, os.WriteFile(source, []byte("binary content"), 0755))
				return source
			},
			setupTarget: func(t *testing.T) string {
				return t.TempDir()
			},
			targetName: "mybinary",
			validateResult: func(t *testing.T, targetPath string) {
				assert.FileExists(t, targetPath)

				info, err := os.Stat(targetPath)
				require.NoError(t, err)

				// Check permissions
				if runtime.GOOS != "windows" {
					assert.Equal(t, os.FileMode(0755), info.Mode()&0777)
				}

				// Check content
				content, err := os.ReadFile(targetPath)
				require.NoError(t, err)
				assert.Equal(t, "binary content", string(content))
			},
		},
		{
			name: "overwrite existing binary",
			setupSource: func(t *testing.T) string {
				source := filepath.Join(t.TempDir(), "new-binary")
				require.NoError(t, os.WriteFile(source, []byte("new content"), 0755))
				return source
			},
			setupTarget: func(t *testing.T) string {
				dir := t.TempDir()
				existing := filepath.Join(dir, "mybinary")
				require.NoError(t, os.WriteFile(existing, []byte("old content"), 0755))
				return dir
			},
			targetName: "mybinary",
			validateResult: func(t *testing.T, targetPath string) {
				content, err := os.ReadFile(targetPath)
				require.NoError(t, err)
				assert.Equal(t, "new content", string(content))
			},
		},
		{
			name: "add .exe extension on Windows",
			setupSource: func(t *testing.T) string {
				source := filepath.Join(t.TempDir(), "source.exe")
				require.NoError(t, os.WriteFile(source, []byte("exe content"), 0755))
				return source
			},
			setupTarget: func(t *testing.T) string {
				return t.TempDir()
			},
			targetName: "tool",
			validateResult: func(t *testing.T, targetPath string) {
				if runtime.GOOS == "windows" {
					assert.True(t, strings.HasSuffix(targetPath, ".exe"))
				}
				assert.FileExists(t, targetPath)
			},
		},
		{
			name: "create target directory if missing",
			setupSource: func(t *testing.T) string {
				source := filepath.Join(t.TempDir(), "binary")
				require.NoError(t, os.WriteFile(source, []byte("content"), 0755))
				return source
			},
			setupTarget: func(t *testing.T) string {
				// Return a non-existent directory
				return filepath.Join(t.TempDir(), "new", "bin", "dir")
			},
			targetName: "tool",
			validateResult: func(t *testing.T, targetPath string) {
				assert.FileExists(t, targetPath)
				assert.DirExists(t, filepath.Dir(targetPath))
			},
		},
		{
			name: "source file not found",
			setupSource: func(t *testing.T) string {
				return "/nonexistent/file"
			},
			setupTarget: func(t *testing.T) string {
				return t.TempDir()
			},
			targetName: "tool",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourcePath := tt.setupSource(t)
			targetDir := tt.setupTarget(t)

			targetPath, err := InstallBinary(sourcePath, targetDir, tt.targetName)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, targetDir, filepath.Dir(targetPath))

			if tt.validateResult != nil {
				tt.validateResult(t, targetPath)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		setupEnv map[string]string
		want     string
	}{
		{
			name: "expand tilde to home",
			path: "~/bin",
			setupEnv: map[string]string{
				"HOME": "/home/user",
			},
			want: "/home/user/bin",
		},
		{
			name: "expand environment variable",
			path: "${GOPATH}/bin",
			setupEnv: map[string]string{
				"GOPATH": "/go",
			},
			want: "/go/bin",
		},
		{
			name: "expand multiple variables",
			path: "${HOME}/.local/${APP_NAME}/bin",
			setupEnv: map[string]string{
				"HOME":     "/home/user",
				"APP_NAME": "myapp",
			},
			want: "/home/user/.local/myapp/bin",
		},
		{
			name: "no expansion needed",
			path: "/usr/local/bin",
			want: "/usr/local/bin",
		},
		{
			name: "empty path",
			path: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore environment
			origEnv := make(map[string]string)
			for k := range tt.setupEnv {
				origEnv[k] = os.Getenv(k)
				os.Setenv(k, tt.setupEnv[k])
			}
			defer func() {
				for k, v := range origEnv {
					if v == "" {
						os.Unsetenv(k)
					} else {
						os.Setenv(k, v)
					}
				}
			}()

			got := expandPath(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAtomicInstall(t *testing.T) {
	t.Run("atomic replacement", func(t *testing.T) {
		dir := t.TempDir()
		targetPath := filepath.Join(dir, "binary")

		// Create existing file
		require.NoError(t, os.WriteFile(targetPath, []byte("old"), 0755))

		// Create new content in temp file
		tmpFile := filepath.Join(dir, "new.tmp")
		require.NoError(t, os.WriteFile(tmpFile, []byte("new"), 0755))

		// Perform atomic install
		err := atomicInstall(tmpFile, targetPath)
		require.NoError(t, err)

		// Verify content
		content, err := os.ReadFile(targetPath)
		require.NoError(t, err)
		assert.Equal(t, "new", string(content))

		// Temp file should be gone
		assert.NoFileExists(t, tmpFile)
	})
}

func TestDryRunOutput(t *testing.T) {
	tests := []struct {
		name       string
		sourcePath string
		targetPath string
		want       string
	}{
		{
			name:       "basic dry run message",
			sourcePath: "/tmp/download/binary",
			targetPath: "/usr/local/bin/tool",
			want:       "Would install /tmp/download/binary to /usr/local/bin/tool",
		},
		{
			name:       "with home directory",
			sourcePath: "/tmp/binary",
			targetPath: "~/bin/tool",
			want:       "Would install /tmp/binary to ~/bin/tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DryRunOutput(tt.sourcePath, tt.targetPath)
			assert.Equal(t, tt.want, got)
		})
	}
}
