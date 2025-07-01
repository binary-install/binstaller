package datasource

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"text/template"

	"github.com/apex/log"
	"github.com/binary-install/binstaller/pkg/spec"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	gorelcontext "github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/goreleaser/goreleaser/v2/pkg/defaults"
	"github.com/pkg/errors"
)

var (
	archRegex = regexp.MustCompile(`eq \.Arch "([^"]+)"\s*-*}\}+\s*([^\s{]+)`)
	osRegex   = regexp.MustCompile(`eq \.Os "([^"]+)"\s*-*}\}+\s*([^\s{]+)`)
)

// goreleaserAdapter implements the SourceAdapter interface for GoReleaser config files.
type goreleaserAdapter struct {
	repo         string
	filePath     string
	commit       string
	nameOverride string
}

// NewGoReleaserAdapter creates a new adapter for GoReleaser sources.
func NewGoReleaserAdapter(repo, filePath, commit, nameOverride string) SourceAdapter {
	return &goreleaserAdapter{
		repo:         repo,
		filePath:     filePath,
		commit:       commit,
		nameOverride: nameOverride,
	}
}

// GenerateInstallSpec generates an InstallSpec from a GoReleaser configuration file.
// It can load the configuration from a local file path or a GitHub repository.
// It uses the fields provided at construction as overrides if provided.
func (a *goreleaserAdapter) GenerateInstallSpec(ctx context.Context) (*spec.InstallSpec, error) {
	log.Infof("generating InstallSpec using goreleaserAdapter")
	log.Debugf("Fields - FilePath: %s, Repo: %s, NameOverride: %s", a.filePath, a.repo, a.nameOverride)

	project, err := loadGoReleaserConfig(a.repo, a.filePath, a.commit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load goreleaser config")
	}

	gorelCtx := gorelcontext.New(*project)
	needs := map[string]bool{
		"building binaries":     true,
		"calculating checksums": true,
		"archives":              true,
	}
	for _, defaulter := range defaults.Defaulters {
		// Call building binaries defaulter to fill default build config
		// to populate correct supported_platforms.
		//
		// We should not call needless defaulter, especially "project name" because
		// it fills project name with git remote data and breaks config if
		// binstaller is called outside the target repo.
		if !needs[defaulter.String()] {
			continue
		}
		log.Debugf("setting defaults for %s", defaulter)
		if err := defaulter.Default(gorelCtx); err != nil {
			return nil, errors.Wrap(err, "failed to set defaults")
		}
	}
	project = &gorelCtx.Config

	// Map goreleaser config.Project to spec.InstallSpec, passing overrides
	installSpec, err := mapToGoInstallerSpec(project, a.nameOverride, a.repo)
	if err != nil {
		return nil, errors.Wrap(err, "failed to map goreleaser config to InstallSpec")
	}

	log.Info("successfully generated InstallSpec from goreleaser source")
	return installSpec, nil
}

// mapToGoInstallerSpec converts a goreleaser config.Project to spec.InstallSpec.
// It applies overrides for name and repo if provided.
func mapToGoInstallerSpec(project *config.Project, nameOverride, repoOverride string) (*spec.InstallSpec, error) {
	if project == nil {
		return nil, errors.New("goreleaser project config is nil")
	}

	// --- Basic Info ---
	s := &spec.InstallSpec{}

	// Determine Name: Override > project.ProjectName
	if nameOverride != "" {
		s.Name = nameOverride
		log.Debugf("Using name override: %s", s.Name)
	} else if project.ProjectName != "" {
		s.Name = project.ProjectName
		log.Debugf("Using goreleaser project_name as name: %s", project.ProjectName)
	}
	// Name inference from Repo will happen after Repo is determined

	// Determine Repo: Override > release.github
	if repoOverride != "" {
		s.Repo = normalizeRepo(repoOverride) // Normalize the override
		log.Debugf("Using repo override: %s", s.Repo)
	} else if project.Release.GitHub.Owner != "" && project.Release.GitHub.Name != "" {
		s.Repo = fmt.Sprintf("%s/%s", project.Release.GitHub.Owner, project.Release.GitHub.Name)
	} else {
		log.Warnf("could not determine repository owner/name from goreleaser config or override. Use --repo flag.")
	}

	// If name was not determined yet, try to infer from the repository name
	if s.Name == "" && s.Repo != "" {
		parts := strings.Split(s.Repo, "/")
		if len(parts) == 2 && parts[1] != "" {
			s.Name = parts[1]
			log.Infof("goreleaser project_name missing, inferred name from repository: %s", s.Name)
		} else {
			log.Warnf("goreleaser project_name missing and could not infer name from repository '%s'. Use --name flag.", s.Repo)
		}
	} else if s.Name == "" {
		// If name is still empty and repo is also empty
		log.Warnf("goreleaser project_name missing and could not infer name from repository. Use --name flag.")
	}

	// --- Checksums ---
	if !project.Checksum.Disable {
		checksumTemplate, err := translateTemplate(project.Checksum.NameTemplate)
		if err != nil {
			log.WithError(err).Warnf("Failed to translate checksum template, using raw: %s", project.Checksum.NameTemplate)
			checksumTemplate = project.Checksum.NameTemplate // Fallback to raw
		}
		s.Checksums = &spec.ChecksumConfig{
			Template:  checksumTemplate,
			Algorithm: project.Checksum.Algorithm,
		}
	}

	// --- Archives / Assets / Unpack ---
	if len(project.Archives) > 0 {
		archive := project.Archives[0] // Focus on the first archive

		// Map default archive format to DefaultExtension
		format := archive.Format //nolint:staticcheck
		if len(archive.Formats) > 0 {
			format = archive.Formats[0]
		}
		s.Asset.DefaultExtension = formatToExtension(format)
		log.Debugf("Mapped default archive format '%s' to DefaultExtension '%s'", format, s.Asset.DefaultExtension)

		// Asset Template
		assetTemplate, err := translateTemplate(archive.NameTemplate)
		if err != nil {
			log.WithError(err).Warnf("Failed to translate asset template, using raw: %s", archive.NameTemplate)
			assetTemplate = archive.NameTemplate // Fallback to raw
		}
		s.Asset.Template = assetTemplate

		// Ensure the asset template includes the ${EXT} placeholder as per InstallSpec v1
		if !strings.HasSuffix(s.Asset.Template, "${EXT}") {
			s.Asset.Template += "${EXT}"
			log.Debugf("Appended ${EXT} to asset template: %s", s.Asset.Template)
		}

		// Infer NamingConvention from the asset template
		if strings.Contains(archive.NameTemplate, "title .Os") {
			s.Asset.NamingConvention = &spec.NamingConvention{
				OS: "titlecase",
				// Arch is assumed lowercase unless a complex template suggests otherwise,
				// which is too complex to infer reliably here.
				Arch: "lowercase", // Default, explicitly set for clarity
			}
			log.Debugf("Inferred OS naming convention as 'titlecase' from template: %s", archive.NameTemplate)
		} else {
			// If no explicit title casing for OS, rely on spec.SetDefaults for lowercase
			s.Asset.NamingConvention = &spec.NamingConvention{
				OS:   "lowercase", // Default, explicitly set for clarity
				Arch: "lowercase", // Default, explicitly set for clarity
			}
		}

		s.Asset.Rules = make([]spec.AssetRule, 0)

		// Asset Rules (Arch)
		for _, m := range archRegex.FindAllStringSubmatch(archive.NameTemplate, -1) {
			if len(m) == 3 && m[1] != "" && m[2] != "" {
				log.Debugf("Inferred Arch name alias (%s -> %s) from template: %s", m[1], m[2], m[0])
				s.Asset.Rules = append(s.Asset.Rules, spec.AssetRule{
					When: spec.PlatformCondition{Arch: m[1]},
					Arch: m[2],
				})
			}
		}

		// Asset Rules (OS)
		for _, m := range osRegex.FindAllStringSubmatch(archive.NameTemplate, -1) {
			if len(m) == 3 && m[1] != "" && m[2] != "" {
				log.Debugf("Inferred OS name alias (%s -> %s) from template: %s", m[1], m[2], m[0])
				s.Asset.Rules = append(s.Asset.Rules, spec.AssetRule{
					When: spec.PlatformCondition{OS: m[1]},
					OS:   m[2],
				})
			}
		}

		// Asset Rules (Format Overrides)
		if len(archive.FormatOverrides) > 0 {
			for _, override := range archive.FormatOverrides {
				format := override.Format //nolint:staticcheck
				if len(override.Formats) > 0 {
					format = override.Formats[0]
				}
				ext := formatToExtension(format)
				// Only add rule if it results in a meaningful extension override
				// or explicitly sets format to binary (empty ext)
				if ext != "" || format == "binary" {
					rule := spec.AssetRule{
						When: spec.PlatformCondition{OS: override.Goos},
						Ext:  ext,
					}
					s.Asset.Rules = append(s.Asset.Rules, rule)
				} else {
					log.Warnf("Ignoring format override for os '%s' with unknown format '%s'", override.Goos, format)
				}
			}
		}

		// Unpack Config
		if archive.WrapInDirectory == "true" {
			strip := 1
			s.Unpack = &spec.UnpackConfig{StripComponents: &strip}
		}
	} else {
		log.Warnf("no archives found in goreleaser config, asset information may be incomplete")
		s.Asset.Template = "${NAME}_${VERSION}_${OS}_${ARCH}${EXT}" // A basic default
	}

	// --- Supported Platforms (from Builds) ---
	s.SupportedPlatforms = deriveSupportedPlatforms(project.Builds) // Pass the whole slice

	log.Infof("initial mapping from goreleaser config complete")
	return s, nil
}

// deriveSupportedPlatforms generates a list of platforms from goreleaser build configurations.
func deriveSupportedPlatforms(builds []config.Build) []spec.Platform {
	platforms := make(map[string]spec.Platform) // Use map to deduplicate

	// Collect all ignore rules from all builds into a single map
	ignore := make(map[string]bool)
	for _, build := range builds {
		for _, ignoredBuild := range build.Ignore {
			platformKey := makePlatformKey(ignoredBuild.Goos, ignoredBuild.Goarch, ignoredBuild.Goarm)
			ignore[platformKey] = true
		}
	}

	// Iterate through target platforms for all builds and add if not ignored
	for _, build := range builds {
		for _, goos := range build.Goos {
			for _, goarch := range build.Goarch {
				if goarch == "arm" {
					for _, goarm := range build.Goarm {
						platformKey := makePlatformKey(goos, goarch, goarm)
						platformKeyWithoutArm := makePlatformKey(goos, goarch, "")
						if !ignore[platformKey] && !ignore[platformKeyWithoutArm] && isValidTarget(goos, goarch) {
							// Map arm version to Arch field directly for simplicity now
							// e.g., linux/arm/6 -> {OS: linux, Arch: armv6}
							platforms[platformKey] = spec.Platform{OS: goos, Arch: goarch + "v" + goarm}
						}
					}
				} else {
					platformKey := makePlatformKey(goos, goarch, "")
					if !ignore[platformKey] && isValidTarget(goos, goarch) {
						platforms[platformKey] = spec.Platform{OS: goos, Arch: goarch}
					}
				}
			}
		}
	}

	// Convert map to slice
	result := make([]spec.Platform, 0, len(platforms))
	for _, p := range platforms {
		result = append(result, p)
	}
	slices.SortStableFunc(result, func(i, j spec.Platform) int {
		return cmp.Or(
			cmp.Compare(i.OS, j.OS),
			cmp.Compare(i.Arch, j.Arch),
		)
	})
	return result
}

// makePlatformKey creates a unique string key for a platform combination.
func makePlatformKey(goos, goarch, goarm string) string {
	key := goos + "/" + goarch
	if goarch == "arm" && goarm != "" {
		key += "v" + goarm // Directly append arm version (e.g., linux/armv6, linux/armv7)
	}
	return key
}

// translateTemplate converts the given name template to its equivalent in InstallSpec format.
// It uses text/template to evaluate the GoReleaser template syntax.
func translateTemplate(tmpl string) (string, error) {
	// Define the variable mapping from GoReleaser template variables to InstallSpec placeholders
	varmap := map[string]string{
		"ProjectName": "${NAME}",
		"Binary":      "${NAME}", // Assume Binary maps to spec Name
		"Version":     "${VERSION}",
		"Tag":         "${TAG}",
		"Os":          "${OS}",
		"Arch":        "${ARCH}",
		"Arm":         "", // Map Arm to empty string as per InstallSpec v1
		"Mips":        "", // Mips not directly mapped to a standard placeholder
		"Amd64":       "", // Amd64 maps to ARCH
	}

	// Create a function map for the template engine
	funcMap := template.FuncMap{
		"title": func(s string) string {
			// We intentionally don't use strings.Title or cases.Title here
			// because it can cause issues with template variables like ${OS}.
			// If we transform "OS" to "Os", the shell script will break.
			// We return the original string to preserve the casing for the placeholder.
			return s
		},
		"tolower": strings.ToLower,
		"toupper": strings.ToUpper,
		"trim":    strings.TrimSpace,
		// Add other functions as needed based on common goreleaser templates
		"replace":    strings.ReplaceAll, // Added replace function based on common usage
		"trimprefix": strings.TrimPrefix, // Added trimprefix
		"trimsuffix": strings.TrimSuffix, // Added trimsuffix
	}

	// Parse the template
	t, err := template.New("template").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse template: %s", tmpl)
	}

	// Execute the template with the variable map
	var buf bytes.Buffer
	err = t.Execute(&buf, varmap)
	if err != nil {
		return "", errors.Wrapf(err, "failed to execute template: %s", tmpl)
	}

	return buf.String(), nil
}

// loadGoReleaserConfig loads a goreleaser project configuration.
// It tries logading from a local file, then falls back to loading from a GitHub repo.
func loadGoReleaserConfig(repo, file, commitHash string) (project *config.Project, err error) {
	// Try loading from local file if file is provided
	if file != "" {
		log.Infof("attempting to load goreleaser config from local file: %s", file)
		project, err = loadFromFile(file)
		if err == nil {
			log.Infof("successfully loaded config from local file: %s", file)
			return project, nil
		}
		log.Warnf("failed to load config from local file %s: %v", file, err)
	}

	// Try loading from GitHub
	if repo != "" {
		repo = normalizeRepo(repo)
		log.Infof("attempting to load goreleaser config from github repo: %s", repo)
		for _, configPath := range []string{file, "goreleaser.yml", ".goreleaser.yml", "goreleaser.yaml", ".goreleaser.yaml"} {
			if configPath == "" {
				continue
			}
			project, err = loadFromGitHub(repo, configPath, commitHash)
			if err == nil {
				log.Info("successfully loaded config from github")
				return project, nil
			} else {
				log.Warnf("failed to load config from github repo %s (path: %s): %v", repo, configPath, err)
			}
		}
	}

	return nil, errors.New("failed to load goreleaser config")
}

// loadFromGitHub loads a project configuration from a GitHub repository.
// Adapted from main.go, simplified commit handling for now.
func loadFromGitHub(repo, configPath, specifiedCommitHash string) (*config.Project, error) {
	log.Infof("loading config for %s at path %s from github", repo, configPath)

	commitHash := "HEAD"
	if specifiedCommitHash != "" {
		commitHash = specifiedCommitHash
	}

	// Construct the raw URL
	if configPath == "" {
		return nil, errors.New("config path within repository must be specified")
	}
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", repo, commitHash, configPath)
	log.Infof("fetching config from URL: %s", url)
	resp, err := http.Get(url) // Basic GET, no token handling yet
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch config from %s", url)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch config from %s: status %d", url, resp.StatusCode)
	}

	// Read the content into a buffer first to allow parsing and potential hashing later
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return nil, errors.Wrap(err, "failed to read config content from response body")
	}
	contentBytes := buf.Bytes()

	// Parse the content using goreleaser's logic
	project, err := config.LoadReader(bytes.NewReader(contentBytes)) // Pass only the reader
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse goreleaser config from github")
	}
	return &project, nil
}

// loadFromFile loads a project configuration from a local file.
// Adapted from main.go.
func loadFromFile(file string) (*config.Project, error) {
	log.Infof("loading config from file %q", file)
	// Parse the file using goreleaser's logic
	project, err := config.Load(file) // Pass only the file path
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse goreleaser config from file %s", file)
	}

	return &project, nil
}

// normalizeRepo cleans up a repository string.
// Adapted from main.go.
func normalizeRepo(repo string) string {
	repo = strings.TrimPrefix(repo, "https://github.com/")
	repo = strings.TrimPrefix(repo, "http://github.com/")
	repo = strings.TrimPrefix(repo, "github.com/")
	repo = strings.Trim(repo, "/")
	return repo
}
