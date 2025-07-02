// This file contains a minimal implementation of goreleaser defaults functionality.
//
// Why this exists:
// The goreleaser/v2/pkg/defaults package brings in massive dependencies including
// Google Cloud SDK, CEL expression language, and many other libraries that we don't need.
// By implementing only the defaults we actually use (building binaries, checksums, archives),
// we can significantly reduce the binary size.
//
// Binary size reduction: 121MB → 31MB (approximately 90MB reduction)
//
// This implementation provides the same functionality for our use case while avoiding
// unnecessary dependencies that bloat the binary.

package datasource

import (
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	gorelcontext "github.com/goreleaser/goreleaser/v2/pkg/context"
)

// applyMinimalDefaults applies only the essential defaults we need without importing the entire defaults package
func applyMinimalDefaults(ctx *gorelcontext.Context) error {
	// Apply building binaries defaults
	if err := applyBuildDefaults(ctx); err != nil {
		return err
	}

	// Apply checksum defaults
	applyChecksumDefaults(ctx)

	// Apply archives defaults
	applyArchiveDefaults(ctx)

	return nil
}

func applyBuildDefaults(ctx *gorelcontext.Context) error {
	project := &ctx.Config

	// If no builds are defined, create a default one
	if len(project.Builds) == 0 {
		project.Builds = []config.Build{{
			ID:     "default",
			Goos:   []string{"linux", "darwin", "windows"},
			Goarch: []string{"amd64", "arm64", "386"},
			Goarm:  []string{"6", "7"},
			Binary: "{{ .ProjectName }}",
		}}
		return nil
	}

	// Apply defaults to existing builds
	for i := range project.Builds {
		build := &project.Builds[i]

		// Default ID
		if build.ID == "" {
			build.ID = "default"
		}

		// Default binary name
		if build.Binary == "" {
			build.Binary = "{{ .ProjectName }}"
		}

		// Default OS/Arch if not specified
		if len(build.Goos) == 0 {
			build.Goos = []string{"linux", "darwin", "windows"}
		}
		if len(build.Goarch) == 0 {
			build.Goarch = []string{"amd64", "arm64", "386"}
		}
		if len(build.Goarm) == 0 {
			build.Goarm = []string{"6", "7"}
		}
	}

	return nil
}

func applyChecksumDefaults(ctx *gorelcontext.Context) {
	project := &ctx.Config

	// Default checksum settings
	if project.Checksum.NameTemplate == "" {
		project.Checksum.NameTemplate = "{{ .ProjectName }}_{{ .Version }}_checksums.txt"
	}
	if project.Checksum.Algorithm == "" {
		project.Checksum.Algorithm = "sha256"
	}
}

func applyArchiveDefaults(ctx *gorelcontext.Context) {
	project := &ctx.Config

	// If no archives are defined, create a default one
	if len(project.Archives) == 0 {
		project.Archives = []config.Archive{{
			ID:           "default",
			NameTemplate: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
			Format:       "tar.gz",
			Formats:      []string{"tar.gz", "zip"},
		}}
		return
	}

	// Apply defaults to existing archives
	for i := range project.Archives {
		archive := &project.Archives[i]

		// Default ID
		if archive.ID == "" {
			archive.ID = "default"
		}

		// Default name template
		if archive.NameTemplate == "" {
			archive.NameTemplate = "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}"
		}

		// Default formats
		if len(archive.Formats) == 0 && archive.Format == "" {
			archive.Formats = []string{"tar.gz", "zip"}
		}

	}
}
