// Package update relie Vinyl au système de mise à jour de Wails : une source GitHub à deux canaux,
// stable pour tous, bêta pour les contributeurs (code valide).
package update

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

const (
	Repository   = "Peuraud-Pierre/vinyl"
	Checksums    = "SHA256SUMS"
	AssetMac     = "Vinyl-darwin-arm64.zip"
	AssetWindows = "Vinyl-windows-amd64.zip"
)

// Channel choisit, à chaque vérification, la source stable ou celle qui inclut les bêtas.
type Channel struct {
	stable, beta updater.Provider
	betaOn       func() bool
	mu           sync.Mutex
	last         updater.Provider // source de la dernière version proposée (pour la télécharger)
}

func NewChannel(stable, beta updater.Provider, betaOn func() bool) *Channel {
	return &Channel{stable: stable, beta: beta, betaOn: betaOn}
}

// NewGitHub : releases GitHub de Vinyl, empreintes vérifiées avec SHA256SUMS.
func NewGitHub(betaOn func() bool) (*Channel, error) {
	stable, err := github.New(github.Config{Repository: Repository, ChecksumAsset: Checksums})
	if err != nil {
		return nil, err
	}
	beta, err := github.New(github.Config{Repository: Repository, ChecksumAsset: Checksums, Prerelease: true})
	if err != nil {
		return nil, err
	}
	return NewChannel(stable, beta, betaOn), nil
}

func (c *Channel) Name() string { return "vinyl-github" }

func (c *Channel) Check(ctx context.Context, req updater.CheckRequest) (*updater.Release, error) {
	p := c.stable
	if c.betaOn() {
		p = c.beta
	}
	c.mu.Lock()
	c.last = p
	c.mu.Unlock()
	return p.Check(ctx, req)
}

func (c *Channel) Download(ctx context.Context, r *updater.Release, dst io.Writer, onProgress func(int64, int64)) error {
	c.mu.Lock()
	p := c.last
	c.mu.Unlock()
	if p == nil {
		return errors.New("update: aucune version proposée")
	}
	return p.Download(ctx, r, dst, onProgress)
}

var _ updater.Provider = (*Channel)(nil)
