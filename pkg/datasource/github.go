package datasource

import (
	"bytes"
	"context"

	"github.com/aquaproj/aqua/v2/pkg/config"
	"github.com/aquaproj/aqua/v2/pkg/controller"
	"github.com/binary-install/binstaller/pkg/httpclient"
	"github.com/binary-install/binstaller/pkg/spec"
	log "github.com/sirupsen/logrus"
)

// GitHubAdapter implements SourceAdapter for GitHub release using `aqua
// generatea-registry` internally. Note: No aqua CLI dependency.
type GitHubAdapter struct {
	repo string // Used for GitHub fetch, e.g. "owner/name"
}

// NewGitHubAdapter creates an adapter that generate aqua registry YAML from
// GitHub release and then convert it to binstalelr's InstallSpec.
func NewGitHubAdapter(repo string) *GitHubAdapter {
	return &GitHubAdapter{repo: repo}
}

func (g *GitHubAdapter) GenerateInstallSpec(ctx context.Context) (*spec.InstallSpec, error) {
	param := &config.Param{Limit: 1}
	logE := log.NewEntry(log.New())
	var registry bytes.Buffer
	ctrl := controller.InitializeGenerateRegistryCommandController(ctx, logE, param, httpclient.NewGitHubClient(), &registry)
	if err := ctrl.GenerateRegistry(ctx, param, logE, g.repo); err != nil {
		return nil, err
	}
	return genSpecFromRegistryYAML(ctx, &registry)
}
