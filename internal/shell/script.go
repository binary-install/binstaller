package shell

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/binary-install/binstaller/pkg/spec"
	"github.com/pkg/errors"
)

// templateData holds the data passed to the shell script template execution.
// It only includes static data from the spec.
type templateData struct {
	*spec.InstallSpec        // Embed the original spec for access to fields like Name, Repo, Asset, Checksums, etc.
	Shlib             string // The content of the shell function library
	HashFunctions     string
	ShellFunctions    string
	TargetVersion     string // Fixed version when --target-version is specified
}

// Generate creates the installer shell script content based on the InstallSpec.
// The generated script will dynamically determine OS, Arch, and Version at runtime.
func Generate(installSpec *spec.InstallSpec) ([]byte, error) {
	return GenerateWithVersion(installSpec, "")
}

// GenerateWithVersion creates the installer shell script content based on the InstallSpec.
// If targetVersion is specified, the script will be generated for that specific version only.
func GenerateWithVersion(installSpec *spec.InstallSpec, targetVersion string) ([]byte, error) {
	if installSpec == nil {
		return nil, errors.New("install spec cannot be nil")
	}
	// Apply spec defaults first
	installSpec.SetDefaults()

	// Filter embedded checksums if target version is specified
	if targetVersion != "" {
		installSpec = filterChecksumsForVersion(installSpec, targetVersion)
	}

	// --- Prepare Template Data ---
	// Only pass static data known at generation time, plus the shell functions
	data := templateData{
		InstallSpec:    installSpec,
		Shlib:          shlib,
		HashFunctions:  hashFunc(installSpec),
		ShellFunctions: shellFunctions,
		TargetVersion:  targetVersion,
	}

	// --- Prepare Template ---
	// The template now needs to contain the logic for runtime detection and asset resolution
	funcMap := createFuncMap() // Keep helper funcs like default, tolower etc.

	tmpl, err := template.New("installer").Funcs(funcMap).Parse(mainScriptTemplate) // Parse only the main template
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse installer template")
	}

	// --- Execute Template ---
	var buf bytes.Buffer
	// Execute the template with the data struct.
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute installer template")
	}

	return buf.Bytes(), nil
}

// filterChecksumsForVersion filters embedded checksums to only include the specified version
// This function modifies the original installSpec to filter checksums
func filterChecksumsForVersion(installSpec *spec.InstallSpec, targetVersion string) *spec.InstallSpec {
	if installSpec.Checksums == nil || installSpec.Checksums.EmbeddedChecksums == nil || len(installSpec.Checksums.EmbeddedChecksums) == 0 {
		return installSpec
	}

	// Filter embedded checksums in place - only keep the target version
	if checksums, exists := installSpec.Checksums.EmbeddedChecksums[targetVersion]; exists {
		// Replace the entire map with only the target version
		installSpec.Checksums.EmbeddedChecksums = map[string][]spec.EmbeddedChecksum{
			targetVersion: checksums,
		}
	} else {
		// Target version not found, clear all embedded checksums
		installSpec.Checksums.EmbeddedChecksums = make(map[string][]spec.EmbeddedChecksum)
	}

	return installSpec
}

func hashFunc(installSpec *spec.InstallSpec) string {
	algo := ""
	if installSpec.Checksums != nil {
		algo = installSpec.Checksums.Algorithm
	}
	switch algo {
	case "sha1":
		return hashSHA1
	case "md5":
		return hashMD5
	case "sha256":
		return hashSHA256
	case "sha512":
		return hashSHA512
	}
	return hashSHA256
}

// createFuncMap defines the functions available to the Go template.
func createFuncMap() template.FuncMap {
	return template.FuncMap{
		"default": func(def, val interface{}) interface{} {
			sVal := fmt.Sprintf("%v", val)
			if sVal == "" || sVal == "0" || sVal == "<nil>" || sVal == "false" {
				return def
			}
			return val
		},
		"hasBinaryOverride": func(asset spec.AssetConfig) bool {
			for _, rule := range asset.Rules {
				if len(rule.Binaries) > 0 {
					return true
				}
			}
			return false
		},
		"trimPrefix": func(s, prefix string) string {
			return strings.TrimPrefix(s, prefix)
		},
	}
}
