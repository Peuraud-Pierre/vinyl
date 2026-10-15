package update

import (
	"context"
	"io"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

type fakeProvider struct {
	name       string
	version    string
	downloads  int
	unverified bool
}

func (f *fakeProvider) Name() string { return f.name }
func (f *fakeProvider) Check(context.Context, updater.CheckRequest) (*updater.Release, error) {
	if f.unverified {
		return &updater.Release{Version: f.version}, nil
	}
	return &updater.Release{Version: f.version, Verification: &updater.Verification{DigestAlgo: "sha256", Digest: []byte{1}}}, nil
}
func (f *fakeProvider) Download(context.Context, *updater.Release, io.Writer, func(int64, int64)) error {
	f.downloads++
	return nil
}

// Sans code contributeur : versions stables seulement ; avec : bêtas comprises ; code retiré : retour au stable.
func TestChannelFollowsBetaAccess(t *testing.T) {
	stable, beta := &fakeProvider{name: "s", version: "1.0.3"}, &fakeProvider{name: "b", version: "1.1.0-beta.1"}
	on := false
	c := NewChannel(stable, beta, func() bool { return on })
	ctx := context.Background()
	if r, _ := c.Check(ctx, updater.CheckRequest{}); r.Version != "1.0.3" {
		t.Errorf("sans code : %q", r.Version)
	}
	on = true
	r, _ := c.Check(ctx, updater.CheckRequest{})
	if r.Version != "1.1.0-beta.1" {
		t.Fatalf("avec code : %q", r.Version)
	}
	c.Download(ctx, r, io.Discard, func(int64, int64) {})
	if beta.downloads != 1 || stable.downloads != 0 {
		t.Errorf("téléchargement par la source qui a proposé la version")
	}
	on = false
	if r, _ := c.Check(ctx, updater.CheckRequest{}); r.Version != "1.0.3" {
		t.Errorf("code retiré : %q", r.Version)
	}
}

// Les noms des fichiers publiés sont bien ceux que Wails choisit pour chaque système.
func TestAssetNamesMatchWailsMatcher(t *testing.T) {
	assets := []github.ReleaseAsset{{Name: "Vinyl-Setup.exe"}, {Name: "SHA256SUMS"}, {Name: AssetMac}, {Name: AssetWindows}}
	for req, want := range map[updater.CheckRequest]string{
		{Platform: "darwin", Arch: "arm64"}:  AssetMac,
		{Platform: "windows", Arch: "amd64"}: AssetWindows,
	} {
		i := github.DefaultAssetMatcher(req, assets)
		if i < 0 || assets[i].Name != want {
			t.Errorf("%s/%s : fichier choisi %d", req.Platform, req.Arch, i)
		}
	}
}

// Les deux sources GitHub se construisent (dépôt et fichier d'empreintes valides).
func TestNewGitHub(t *testing.T) {
	if _, err := NewGitHub(func() bool { return false }); err != nil {
		t.Fatal(err)
	}
}

// Release sans empreinte (SHA256SUMS absent ou pas encore envoyé) : pas proposée, rien d'invérifié.
func TestUnverifiedReleaseIsIgnored(t *testing.T) {
	p := &fakeProvider{name: "s", version: "1.0.3", unverified: true}
	c := NewChannel(p, p, func() bool { return false })
	if r, err := c.Check(context.Background(), updater.CheckRequest{}); r != nil || err != nil {
		t.Errorf("r=%+v err=%v", r, err)
	}
}
