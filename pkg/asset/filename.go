package asset

import (
	"fmt"
	"strings"

	"github.com/binary-install/binstaller/pkg/spec"
	"github.com/buildkite/interpolate"
)

// FilenameGenerator generates asset filenames based on templates and rules
type FilenameGenerator struct {
	Spec    *spec.InstallSpec
	Version string
}

// NewFilenameGenerator creates a new filename generator
func NewFilenameGenerator(spec *spec.InstallSpec, version string) *FilenameGenerator {
	return &FilenameGenerator{
		Spec:    spec,
		Version: version,
	}
}

// GenerateFilename creates an asset filename for a specific OS and Arch
func (g *FilenameGenerator) GenerateFilename(osInput, archInput string) (string, error) {
	if g.Spec == nil || g.Spec.Asset == nil || spec.StringValue(g.Spec.Asset.Template) == "" {
		return "", fmt.Errorf("asset template not defined in spec")
	}

	// Keep original values for rule matching
	osMatch := strings.ToLower(osInput)
	archMatch := strings.ToLower(archInput)

	// Create formatted values for template substitution
	osValue := osMatch
	archValue := archMatch

	// Apply OS/Arch naming conventions for template values
	if g.Spec.Asset.NamingConvention != nil {
		if spec.NamingConventionOSString(g.Spec.Asset.NamingConvention.OS) == "titlecase" {
			osValue = titleCase(osValue)
		}
	}

	// Apply rules to get the right extension and override OS/Arch if needed
	ext := spec.StringValue(g.Spec.Asset.DefaultExtension)
	template := spec.StringValue(g.Spec.Asset.Template)

	// Check if any rule applies - use osMatch/archMatch for condition checking
	for _, rule := range g.Spec.Asset.Rules {
		if rule.When != nil &&
			(spec.StringValue(rule.When.OS) == "" || spec.StringValue(rule.When.OS) == osMatch) &&
			(spec.StringValue(rule.When.Arch) == "" || spec.StringValue(rule.When.Arch) == archMatch) {
			if spec.StringValue(rule.OS) != "" {
				osValue = spec.StringValue(rule.OS)
			}
			if spec.StringValue(rule.Arch) != "" {
				archValue = spec.StringValue(rule.Arch)
			}
			if spec.StringValue(rule.EXT) != "" {
				ext = spec.StringValue(rule.EXT)
			}
			if spec.StringValue(rule.Template) != "" {
				template = spec.StringValue(rule.Template)
			}
		}
	}

	// Asset templates support OS, ARCH, and EXT in addition to NAME and VERSION
	additionalVars := map[string]string{
		"OS":   osValue,
		"ARCH": archValue,
		"EXT":  ext,
	}

	// Perform variable substitution in the template
	filename, err := g.interpolateTemplate(template, additionalVars)
	if err != nil {
		return "", fmt.Errorf("failed to interpolate asset template: %w", err)
	}

	return filename, nil
}

// GeneratePossibleFilenames generates all possible asset filenames based on the asset template
func (g *FilenameGenerator) GeneratePossibleFilenames() map[string]bool {
	if g.Spec == nil || g.Spec.Asset == nil || spec.StringValue(g.Spec.Asset.Template) == "" {
		return nil
	}

	// Use map for O(1) lookup performance
	filenames := make(map[string]bool)
	var platforms []spec.Platform

	// Determine which platforms to use
	if len(g.Spec.SupportedPlatforms) > 0 {
		platforms = g.Spec.SupportedPlatforms
	} else {
		// Generate all possible combinations from spec constants
		platforms = g.GetAllPossiblePlatforms()
	}

	// Generate filename for each platform
	for _, platform := range platforms {
		filename, err := g.GenerateFilename(spec.PlatformOSString(platform.OS), spec.PlatformArchString(platform.Arch))
		if err != nil {
			continue
		}
		if filename != "" {
			filenames[filename] = true
		}
	}

	return filenames
}

// GetAllPossiblePlatforms returns all possible OS/Arch combinations from spec constants
func (g *FilenameGenerator) GetAllPossiblePlatforms() []spec.Platform {
	// Get all OS and Arch values from spec constants
	osValues := GetAllOSValues()
	archValues := GetAllArchValues()

	// Generate all combinations
	var platforms []spec.Platform
	for _, os := range osValues {
		for _, arch := range archValues {
			osCopy := os
			archCopy := arch
			platforms = append(platforms, spec.Platform{
				OS:   &osCopy,
				Arch: &archCopy,
			})
		}
	}

	return platforms
}

// interpolateTemplate performs variable substitution in a template string
func (g *FilenameGenerator) interpolateTemplate(template string, additionalVars map[string]string) (string, error) {
	// Create base environment map with variables supported by all templates
	envMap := map[string]string{
		"NAME": spec.StringValue(g.Spec.Name),
		"TAG":  g.Version, // Original tag with 'v' prefix if present
	}

	// VERSION should be without 'v' prefix according to spec documentation
	version := g.Version
	if strings.HasPrefix(version, "v") {
		version = strings.TrimPrefix(version, "v")
	}
	envMap["VERSION"] = version

	// Merge additional variables (OS, ARCH, EXT for asset templates)
	for k, v := range additionalVars {
		envMap[k] = v
	}

	// Perform variable substitution
	env := interpolate.NewMapEnv(envMap)
	return interpolate.Interpolate(env, template)
}

// titleCase converts a string to title case (first letter uppercase, rest lowercase)
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}
