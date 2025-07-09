package checksums

import (
	"bufio"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/apex/log"
	"github.com/binary-install/binstaller/pkg/asset"
	"github.com/binary-install/binstaller/pkg/httpclient"
	"github.com/binary-install/binstaller/pkg/spec"
	"github.com/buildkite/interpolate"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
)

// EmbedMode represents the checksum acquisition mode
type EmbedMode string

const (
	// EmbedModeDownload downloads checksum files from GitHub releases
	EmbedModeDownload EmbedMode = "download"
	// EmbedModeChecksumFile uses a local checksum file
	EmbedModeChecksumFile EmbedMode = "checksum-file"
	// EmbedModeCalculate downloads assets and calculates checksums
	EmbedModeCalculate EmbedMode = "calculate"
)

// Embedder manages the process of embedding checksums
type Embedder struct {
	Mode         EmbedMode
	Version      string
	Spec         *spec.InstallSpec
	SpecAST      *ast.File
	ChecksumFile string
}

// Embed performs the checksum embedding process and returns the updated spec
func (e *Embedder) Embed() error {
	if e.Spec == nil {
		return fmt.Errorf("InstallSpec cannot be nil")
	}

	// If Checksums section doesn't exist, create it with defaults
	if e.Spec.Checksums == nil {
		sha256 := spec.Sha256
		e.Spec.Checksums = &spec.ChecksumConfig{
			Algorithm: &sha256, // Default algorithm
		}
	}

	// Validate checksum template
	// Note: ${ASSET_FILENAME} could technically be supported by looping through all supported OS/arch combinations,
	// but this would be equivalent to using 'calculate' mode. Since there are no plans to implement this,
	// we explicitly reject it to guide users to the appropriate solution.
	if e.Spec.Checksums.Template != nil && strings.Contains(spec.StringValue(e.Spec.Checksums.Template), "${ASSET_FILENAME}") {
		return fmt.Errorf("${ASSET_FILENAME} is not supported in checksum templates. Use 'binst embed-checksums --mode calculate' instead to generate checksums for all platforms")
	}

	// Resolve version if it's "latest"
	resolvedVersion, err := e.resolveVersion(e.Version)
	if err != nil {
		return fmt.Errorf("failed to resolve version: %w", err)
	}
	e.Version = resolvedVersion

	// Initialize embedded checksums map if it doesn't exist
	if e.Spec.Checksums.EmbeddedChecksums == nil {
		e.Spec.Checksums.EmbeddedChecksums = make(map[string][]spec.EmbeddedChecksum)
	}

	// Clear any existing checksums for this version to avoid duplicates
	e.Spec.Checksums.EmbeddedChecksums[e.Version] = nil

	// Perform checksums embedding based on the selected mode
	var checksums map[string]string
	var embedErr error

	switch e.Mode {
	case EmbedModeDownload:
		checksums, embedErr = e.downloadAndParseChecksumFile()
	case EmbedModeChecksumFile:
		checksums, embedErr = e.parseChecksumFile()
	case EmbedModeCalculate:
		checksums, embedErr = e.calculateChecksums()
	default:
		return fmt.Errorf("invalid mode: %s", e.Mode)
	}

	if embedErr != nil {
		return fmt.Errorf("failed to embed checksums: %w", embedErr)
	}

	// Convert the checksums to EmbeddedChecksum structs
	embeddedChecksums := make([]spec.EmbeddedChecksum, 0, len(checksums))
	for filename, hash := range checksums {
		ec := spec.EmbeddedChecksum{
			Filename: spec.StringPtr(filename),
			Hash:     spec.StringPtr(hash),
		}
		embeddedChecksums = append(embeddedChecksums, ec)
	}

	// Sort embedded checksums by filename for consistent output
	slices.SortStableFunc(embeddedChecksums, func(a, b spec.EmbeddedChecksum) int {
		return strings.Compare(spec.StringValue(a.Filename), spec.StringValue(b.Filename))
	})

	// Update the spec with the new checksums
	e.Spec.Checksums.EmbeddedChecksums[e.Version] = embeddedChecksums
	p, err := yaml.PathString("$.checksums")
	if err != nil {
		return err
	}
	// Create a checksumConfig with all existing checksums to preserve them
	checksumConfig := spec.ChecksumConfig{
		Algorithm:         e.Spec.Checksums.Algorithm,
		Template:          e.Spec.Checksums.Template,
		EmbeddedChecksums: e.Spec.Checksums.EmbeddedChecksums,
	}
	node, err := yaml.ValueToNode(checksumConfig)
	if err != nil {
		return err
	}
	// Try MergeFromNode first to preserve comments when checksums field exists
	// If that fails (e.g., checksums field doesn't exist), fallback to ReplaceWithNode
	if err := p.MergeFromNode(e.SpecAST, node); err != nil {
		// MergeFromNode failed, likely because checksums field doesn't exist
		// Use ReplaceWithNode to handle cases where the checksums field doesn't exist
		if err := p.ReplaceWithNode(e.SpecAST, node); err != nil {
			return err
		}
	}
	return nil
}

// githubRelease represents the minimal structure needed from GitHub release API
type githubRelease struct {
	TagName string `json:"tag_name"`
}

// resolveVersion resolves "latest" or empty version to an actual version string
func (e *Embedder) resolveVersion(version string) (string, error) {
	if version != "latest" && version != "" {
		return version, nil
	}

	if e.Spec == nil || spec.StringValue(e.Spec.Repo) == "" {
		return "", fmt.Errorf("repository not specified in spec")
	}

	// Use GitHub API to get the latest release
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", spec.StringValue(e.Spec.Repo))

	// Set up the request with Accept header for JSON response
	req, err := httpclient.NewRequestWithGitHubAuth("GET", url)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Send the request
	client := httpclient.NewGitHubClient()
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get latest release: %w", err)
	}
	defer resp.Body.Close()

	// Check for successful response
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get latest release, status code: %d", resp.StatusCode)
	}

	// Parse the JSON response
	var release githubRelease
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&release); err != nil {
		return "", fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	if release.TagName == "" {
		return "", fmt.Errorf("empty tag name returned from GitHub")
	}

	log.Infof("Resolved latest version: %s", release.TagName)
	return release.TagName, nil
}

// downloadAndParseChecksumFile downloads a checksum file from GitHub releases and parses it
func (e *Embedder) downloadAndParseChecksumFile() (map[string]string, error) {
	// Create the expected checksum URL using the spec template
	checksumFilename := e.createChecksumFilename()
	if checksumFilename == "" {
		return nil, fmt.Errorf("unable to generate checksum filename")
	}

	checksumURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s",
		spec.StringValue(e.Spec.Repo), e.Version, checksumFilename)

	log.Infof("Downloading checksums from %s", checksumURL)

	// Create a temporary file to store the checksum file
	tempDir, err := os.MkdirTemp("", "binstaller-checksums")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	tempFilePath := filepath.Join(tempDir, "checksums.txt")

	// Download the checksum file
	req, err := httpclient.NewRequestWithGitHubAuth("GET", checksumURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	client := httpclient.NewGitHubClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download checksum file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download checksum file, status code: %d", resp.StatusCode)
	}

	// Save the checksum file to a temporary file
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to save checksum file: %w", err)
	}

	// Parse the checksum file
	checksums, err := parseChecksumFileInternal(tempFilePath)
	if err != nil {
		return nil, err
	}

	// Filter checksums based on asset template
	return e.filterChecksums(checksums), nil
}

// parseChecksumFile parses a local checksum file
func (e *Embedder) parseChecksumFile() (map[string]string, error) {
	if e.ChecksumFile == "" {
		return nil, fmt.Errorf("checksum file path is required for checksum-file mode")
	}

	log.Infof("Parsing checksums from file: %s", e.ChecksumFile)
	checksums, err := parseChecksumFileInternal(e.ChecksumFile)
	if err != nil {
		return nil, err
	}

	// Filter checksums based on asset template
	return e.filterChecksums(checksums), nil
}

// parseChecksumFileInternal parses a checksum file and returns a map of filename to hash
func parseChecksumFileInternal(checksumFile string) (map[string]string, error) {
	checksums := make(map[string]string)

	file, err := os.Open(checksumFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open checksum file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse the line as a checksum entry
		// Format: <hash> [*]<filename>
		parts := strings.Fields(line)
		if len(parts) < 2 {
			log.Warnf("Ignoring invalid checksum line: %s", line)
			continue
		}

		hash := parts[0]
		filename := parts[1] // Take the second field as filename

		// If the filename starts with *, remove it (common in standard checksums)
		filename = strings.TrimPrefix(filename, "*")

		checksums[filename] = hash
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading checksum file: %w", err)
	}

	if len(checksums) == 0 {
		return nil, fmt.Errorf("no checksums found in file")
	}

	return checksums, nil
}

// interpolateTemplate performs variable substitution in a template string
func (e *Embedder) interpolateTemplate(template string, additionalVars map[string]string) (string, error) {
	// Create base environment map with variables supported by all templates
	envMap := map[string]string{
		"NAME": spec.StringValue(e.Spec.Name),
		"TAG":  e.Version, // Original tag with 'v' prefix if present
	}

	// VERSION should be without 'v' prefix according to spec documentation
	version := strings.TrimPrefix(e.Version, "v")
	envMap["VERSION"] = version

	// Merge additional variables (OS, ARCH, EXT for asset templates)
	for k, v := range additionalVars {
		envMap[k] = v
	}

	// Perform variable substitution
	env := interpolate.NewMapEnv(envMap)
	return interpolate.Interpolate(env, template)
}

// createChecksumFilename creates the checksum filename using the template from the spec
func (e *Embedder) createChecksumFilename() string {
	if e.Spec.Checksums == nil || spec.StringValue(e.Spec.Checksums.Template) == "" {
		return ""
	}

	template := spec.StringValue(e.Spec.Checksums.Template)

	// Check for unsupported ASSET_FILENAME variable
	if strings.Contains(template, "${ASSET_FILENAME}") {
		log.Errorf("${ASSET_FILENAME} is not supported in checksum templates. Use 'binst embed-checksums --mode calculate' instead.")
		return ""
	}

	// Note: Checksum templates only support NAME and VERSION according to schema
	filename, err := e.interpolateTemplate(template, nil)
	if err != nil {
		log.Errorf("Failed to interpolate checksum template: %v", err)
		return ""
	}
	return filename
}

// filterChecksums filters checksums to only include files matching possible asset filenames
func (e *Embedder) filterChecksums(checksums map[string]string) map[string]string {
	// Generate all possible asset filenames
	generator := asset.NewFilenameGenerator(e.Spec, e.Version)
	possibleFilenames := generator.GeneratePossibleFilenames()
	if len(possibleFilenames) == 0 {
		log.Warn("No possible asset filenames could be generated, returning all checksums")
		return checksums
	}

	// Filter checksums
	filtered := make(map[string]string)
	for filename, hash := range checksums {
		if possibleFilenames[filename] {
			filtered[filename] = hash
		} else {
			log.Debugf("Filtering out checksum for non-matching file: %s", filename)
		}
	}

	log.Infof("Filtered checksums: %d out of %d entries match asset template", len(filtered), len(checksums))
	return filtered
}

// ComputeHash computes the hash of a file using the specified algorithm
func ComputeHash(filepath string, algorithm string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var h hash.Hash
	switch strings.ToLower(algorithm) {
	case "sha256":
		h = sha256.New()
	case "sha1":
		h = sha1.New()
	case "sha512":
		h = sha512.New()
	default:
		return "", fmt.Errorf("unsupported hash algorithm: %s", algorithm)
	}

	if _, err := io.Copy(h, file); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
