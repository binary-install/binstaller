package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/binary-install/binstaller/pkg/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		setup      func(t *testing.T) string
		wantErr    bool
		validate   func(t *testing.T, cfg *spec.InstallSpec)
	}{
		{
			name:       "load explicit config file",
			configPath: "testdata/gh.yml",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				configPath := filepath.Join(dir, "testdata", "gh.yml")
				require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
				content := `schema: v1
name: gh
repo: cli/cli
asset:
  template: ${NAME}_${VERSION}_${OS}_${ARCH}${EXT}
  default_extension: .tar.gz
checksums:
  template: ${NAME}_${VERSION}_checksums.txt
  algorithm: sha256`
				require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))
				return configPath
			},
			validate: func(t *testing.T, cfg *spec.InstallSpec) {
				assert.NotNil(t, cfg.Name)
				assert.Equal(t, "gh", *cfg.Name)
				assert.NotNil(t, cfg.Repo)
				assert.Equal(t, "cli/cli", *cfg.Repo)
				if cfg.Asset != nil && cfg.Asset.DefaultExtension != nil {
					assert.Equal(t, ".tar.gz", *cfg.Asset.DefaultExtension)
				}
			},
		},
		{
			name:       "config file not found",
			configPath: "nonexistent.yml",
			setup: func(t *testing.T) string {
				return "nonexistent.yml"
			},
			wantErr: true,
		},
		{
			name:       "invalid yaml",
			configPath: "invalid.yml",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				configPath := filepath.Join(dir, "invalid.yml")
				content := `invalid yaml content: [`
				require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))
				return configPath
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			cfg, err := Load(path)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, cfg)
			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestDiscover(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		wantErr  bool
		wantPath string
	}{
		{
			name: "find config in .config/binstaller.yml",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				configPath := filepath.Join(dir, ".config", "binstaller.yml")
				require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
				content := `schema: v1
name: test
repo: test/test`
				require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))
				return dir
			},
			wantPath: ".config/binstaller.yml",
		},
		{
			name: "find config in parent directory",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				configPath := filepath.Join(dir, ".config", "binstaller.yml")
				require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
				content := `schema: v1
name: test
repo: test/test`
				require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

				// Create a subdirectory and change to it
				subdir := filepath.Join(dir, "subdir")
				require.NoError(t, os.MkdirAll(subdir, 0755))
				require.NoError(t, os.Chdir(subdir))

				return dir
			},
			wantPath: "../.config/binstaller.yml",
		},
		{
			name: "no config found",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				require.NoError(t, os.Chdir(dir))
				return dir
			},
			wantErr: true,
		},
	}

	// Save current working directory
	origWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origWd)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt.setup(t)

			path, err := Discover()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Verify the path is correct relative to the setup
			if tt.wantPath != "" {
				// The path should end with the expected config file
				assert.True(t, strings.HasSuffix(path, ".config/binstaller.yml"))
			}
		})
	}
}

func TestLoadOrDiscover(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		setup      func(t *testing.T) (string, string) // returns (explicit path, discovered path)
		wantErr    bool
		validate   func(t *testing.T, cfg *spec.InstallSpec, path string)
	}{
		{
			name:       "use explicit config path",
			configPath: "explicit.yml",
			setup: func(t *testing.T) (string, string) {
				dir := t.TempDir()

				// Create explicit config
				explicitPath := filepath.Join(dir, "explicit.yml")
				explicitContent := `schema: v1
name: explicit
repo: test/explicit`
				require.NoError(t, os.WriteFile(explicitPath, []byte(explicitContent), 0644))

				// Also create default config (should be ignored)
				defaultPath := filepath.Join(dir, ".config", "binstaller.yml")
				require.NoError(t, os.MkdirAll(filepath.Dir(defaultPath), 0755))
				defaultContent := `schema: v1
name: default
repo: test/default`
				require.NoError(t, os.WriteFile(defaultPath, []byte(defaultContent), 0644))

				require.NoError(t, os.Chdir(dir))
				return explicitPath, defaultPath
			},
			validate: func(t *testing.T, cfg *spec.InstallSpec, path string) {
				assert.Equal(t, "explicit", *cfg.Name)
				assert.Contains(t, path, "explicit.yml")
			},
		},
		{
			name:       "discover config when no explicit path",
			configPath: "",
			setup: func(t *testing.T) (string, string) {
				dir := t.TempDir()

				// Create default config
				defaultPath := filepath.Join(dir, ".config", "binstaller.yml")
				require.NoError(t, os.MkdirAll(filepath.Dir(defaultPath), 0755))
				content := `schema: v1
name: discovered
repo: test/discovered`
				require.NoError(t, os.WriteFile(defaultPath, []byte(content), 0644))

				require.NoError(t, os.Chdir(dir))
				return "", defaultPath
			},
			validate: func(t *testing.T, cfg *spec.InstallSpec, path string) {
				assert.Equal(t, "discovered", *cfg.Name)
				assert.Contains(t, path, "binstaller.yml")
			},
		},
	}

	// Save current working directory
	origWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origWd)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			explicitPath, _ := tt.setup(t)

			pathToUse := tt.configPath
			if pathToUse == "explicit.yml" {
				pathToUse = explicitPath
			}

			cfg, path, err := LoadOrDiscover(pathToUse)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, cfg)

			if tt.validate != nil {
				tt.validate(t, cfg, path)
			}
		})
	}
}
