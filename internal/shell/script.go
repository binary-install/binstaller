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
}

// Generate creates the installer shell script content based on the InstallSpec.
// The generated script will dynamically determine OS, Arch, and Version at runtime.
func Generate(installSpec *spec.InstallSpec) ([]byte, error) {
	if installSpec == nil {
		return nil, errors.New("install spec cannot be nil")
	}
	// Apply spec defaults first
	installSpec.SetDefaults()

	// --- Prepare Template Data ---
	// Only pass static data known at generation time, plus the shell functions
	data := templateData{
		InstallSpec:    installSpec,
		Shlib:          shlib,
		HashFunctions:  hashFunc(installSpec),
		ShellFunctions: shellFunctions,
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
