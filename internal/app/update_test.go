package app

import (
	"context"
	"crypto/ed25519"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/updater"

	"vinyl/internal/beta"
	"vinyl/internal/player"
	"vinyl/internal/settings"
)

func newBare() *VinylService {
	return New(nil, player.NewWatcher(nil, func() string { return "auto" }, func(player.Snapshot) {}, time.Now), nil)
}

func TestVersion(t *testing.T) {
	s := newBare()
	if v := s.Version(); v != "dev" {
		t.Errorf("par défaut : %q", v)
	}
	s.SetVersion("1.0.3")
	if v := s.Version(); v != "1.0.3" {
		t.Errorf("injectée : %q", v)
	}
}

type fakeUpdater struct {
	rel      *updater.Release
	checkErr error
	installs int
	restarts int
	busy     bool
	state    updater.State
	checks   int
}

func (f *fakeUpdater) Check(context.Context) (*updater.Release, error) {
	f.checks++
	return f.rel, f.checkErr
}
func (f *fakeUpdater) DownloadAndInstall(context.Context) error {
	if f.busy {
		return updater.ErrDownloadInProgress
	}
	f.installs++
	return nil
}
func (f *fakeUpdater) Restart(context.Context) error { f.restarts++; return nil }
func (f *fakeUpdater) State() updater.State {
	if f.state == "" {
		return updater.StateIdle
	}
	return f.state
}

func TestDevBuildNeverChecks(t *testing.T) {
	s := newBare()
	s.SetUpdater(&fakeUpdater{rel: &updater.Release{Version: "9.9.9"}})
	info, err := s.CheckForUpdate()
	if err != nil || info.Available || !info.Dev {
		t.Errorf("dev : %+v %v", info, err)
	}
}

func TestUpdateAvailableThenInstalled(t *testing.T) {
	s := newBare()
	s.SetVersion("1.0.2")
	u := &fakeUpdater{rel: &updater.Release{Version: "1.0.3", Notes: "Nouveautés"}}
	s.SetUpdater(u)
	info, err := s.CheckForUpdate()
	if err != nil || !info.Available || info.Version != "1.0.3" || info.Notes != "Nouveautés" {
		t.Fatalf("%+v %v", info, err)
	}
	s.exePath = func() (string, error) { return filepath.Join(t.TempDir(), "vinyl"), nil }
	if ok, err := s.InstallUpdate(); err != nil || !ok || u.installs != 1 || u.restarts != 1 {
		t.Errorf("ok=%v installs=%d restarts=%d err=%v", ok, u.installs, u.restarts, err)
	}
}

// À jour : Wails renvoie (nil, nil).
func TestUpToDate(t *testing.T) {
	s := newBare()
	s.SetVersion("1.0.3")
	s.SetUpdater(&fakeUpdater{})
	if info, err := s.CheckForUpdate(); err != nil || info.Available {
		t.Errorf("%+v %v", info, err)
	}
}

// Pas de réseau : pas d'erreur montrée, simplement « rien de nouveau ».
func TestCheckWithoutNetworkIsQuiet(t *testing.T) {
	s := newBare()
	s.SetVersion("1.0.2")
	s.SetUpdater(&fakeUpdater{checkErr: errors.New("dial tcp: no route")})
	if info, err := s.CheckForUpdate(); err != nil || info.Available {
		t.Errorf("%+v %v", info, err)
	}
}

// Second clic pendant un téléchargement : ignoré, sans erreur.
func TestInstallWhileDownloadingIsIgnored(t *testing.T) {
	s := newBare()
	s.SetVersion("1.0.2")
	u := &fakeUpdater{busy: true, rel: &updater.Release{Version: "1.0.3"}}
	s.SetUpdater(u)
	s.exePath = func() (string, error) { return filepath.Join(t.TempDir(), "vinyl"), nil }
	if _, err := s.InstallUpdate(); err != nil || u.restarts != 0 {
		t.Errorf("err = %v, restarts = %d", err, u.restarts)
	}
}

func TestReleaseURL(t *testing.T) {
	if u := newBare().ReleaseURL("1.0.3"); u != "https://github.com/Peuraud-Pierre/vinyl/releases/tag/v1.0.3" {
		t.Errorf("%q", u)
	}
}

func TestBetaCode(t *testing.T) {
	st := settings.NewStore(filepath.Join(t.TempDir(), "s.json"))
	st.Load()
	s := New(st, player.NewWatcher(nil, func() string { return "auto" }, func(player.Snapshot) {}, time.Now), nil)
	pub, priv, _ := ed25519.GenerateKey(nil)
	old := beta.PublicKey
	beta.PublicKey = pub
	defer func() { beta.PublicKey = old }()
	if _, err := s.SetBetaCode("faux"); err == nil {
		t.Error("code invalide accepté")
	}
	if st.Get().BetaCode != "" {
		t.Error("code invalide enregistré")
	}
	name, err := s.SetBetaCode(" " + beta.Sign(priv, "Léa") + "\n")
	if err != nil || name != "Léa" || s.BetaAccess() != "Léa" {
		t.Errorf("name=%q err=%v accès=%q", name, err, s.BetaAccess())
	}
	if _, err := s.SetBetaCode(""); err != nil || s.BetaAccess() != "" {
		t.Errorf("retrait : %v %q", err, s.BetaAccess())
	}
}

// Vérification régulière : signale la nouvelle version sans rien télécharger.
func TestPeriodicCheckOnlyAnnounces(t *testing.T) {
	s := newBare()
	s.SetVersion("1.0.2")
	u := &fakeUpdater{rel: &updater.Release{Version: "1.0.3"}}
	s.SetUpdater(u)
	found := make(chan UpdateInfo, 1)
	stop := s.WatchUpdates(10*time.Millisecond, func(i UpdateInfo) {
		select {
		case found <- i:
		default:
		}
	})
	defer stop()
	select {
	case i := <-found:
		if i.Version != "1.0.3" || u.installs != 0 {
			t.Errorf("%+v installs=%d", i, u.installs)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("aucune annonce")
	}
}

// Pendant un téléchargement, la vérification régulière ne fait rien ; en « dev », elle ne démarre pas.
func TestPeriodicCheckSkipsWhenBusyOrDev(t *testing.T) {
	s := newBare()
	u := &fakeUpdater{rel: &updater.Release{Version: "1.0.3"}}
	s.SetUpdater(u)
	stop := s.WatchUpdates(5*time.Millisecond, func(UpdateInfo) {})
	time.Sleep(40 * time.Millisecond)
	stop()
	if u.checks != 0 {
		t.Errorf("dev : %d vérifications", u.checks)
	}
	s.SetVersion("1.0.2")
	u.state = updater.StateDownloading
	stop = s.WatchUpdates(5*time.Millisecond, func(UpdateInfo) {})
	time.Sleep(40 * time.Millisecond)
	stop()
	if u.checks != 0 {
		t.Errorf("occupé : %d vérifications", u.checks)
	}
}

// Version proposée puis plus d'actualité (code contributeur retiré) : « Mettre à jour » n'installe rien.
func TestInstallRechecksFirst(t *testing.T) {
	s := newBare()
	s.SetVersion("1.0.3")
	u := &fakeUpdater{rel: &updater.Release{Version: "1.1.0-beta.1"}}
	s.SetUpdater(u)
	s.exePath = func() (string, error) { return filepath.Join(t.TempDir(), "vinyl"), nil }
	if info, _ := s.CheckForUpdate(); !info.Available {
		t.Fatal("bêta non proposée")
	}
	u.rel = nil // code retiré : plus rien de nouveau
	ok, err := s.InstallUpdate()
	if err != nil || ok || u.installs != 0 {
		t.Errorf("ok=%v err=%v installs=%d", ok, err, u.installs)
	}
}

// Dossier de l'app non modifiable (autre compte, app lancée depuis une image disque…) : erreur avant
// tout téléchargement, pour que le bandeau propose la page de la release.
func TestInstallRefusedWhenAppFolderReadOnly(t *testing.T) {
	dir := t.TempDir()
	app := filepath.Join(dir, "Vinyl.app", "Contents", "MacOS")
	if err := os.MkdirAll(app, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0o755)
	s := newBare()
	s.SetVersion("1.0.2")
	u := &fakeUpdater{rel: &updater.Release{Version: "1.0.3"}}
	s.SetUpdater(u)
	s.exePath = func() (string, error) { return filepath.Join(app, "vinyl"), nil }
	if _, err := s.InstallUpdate(); err == nil || u.installs != 0 {
		t.Errorf("err=%v installs=%d", err, u.installs)
	}
}

func TestReplaceTarget(t *testing.T) {
	for exe, want := range map[string]string{
		"/Applications/Vinyl.app/Contents/MacOS/vinyl":      "/Applications/Vinyl.app",
		`C:\Users\l\AppData\Local\Programs\Vinyl\vinyl.exe`: `C:\Users\l\AppData\Local\Programs\Vinyl\vinyl.exe`,
	} {
		if got := replaceTarget(exe); got != want {
			t.Errorf("%s → %s", exe, got)
		}
	}
}
