package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/updater"

	"vinyl/internal/beta"
	"vinyl/internal/settings"
)

// Updater : ce dont le service a besoin du système de mise à jour de Wails (*updater.Updater).
type Updater interface {
	Check(ctx context.Context) (*updater.Release, error)
	DownloadAndInstall(ctx context.Context) error
	Restart(ctx context.Context) error
	State() updater.State
}

// UpdateInfo : résultat d'une recherche de mise à jour.
type UpdateInfo struct {
	Available bool   `json:"available"`
	Version   string `json:"version"`
	Notes     string `json:"notes"`
	Dev       bool   `json:"dev"` // fabrication à la main : pas de mise à jour
}

func (s *VinylService) SetUpdater(u Updater) { s.upd = u }

// CheckForUpdate cherche une version plus récente (stable, ou bêta avec un code contributeur).
// Sans réseau : rien de nouveau, sans erreur.
func (s *VinylService) CheckForUpdate() (UpdateInfo, error) {
	if s.version == "dev" || s.upd == nil {
		return UpdateInfo{Dev: s.version == "dev"}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rel, err := s.upd.Check(ctx)
	if err != nil || rel == nil {
		return UpdateInfo{}, nil
	}
	return UpdateInfo{Available: true, Version: rel.Version, Notes: rel.Notes}, nil
}

var errReadOnlyApp = errors.New("Vinyl ne peut pas se remplacer à cet endroit (droits insuffisants)")

// InstallUpdate revérifie la version proposée (le code contributeur a pu être retiré entre-temps), la
// télécharge et la vérifie, la met en place et redémarre Vinyl. false : plus rien à installer.
func (s *VinylService) InstallUpdate() (bool, error) {
	if s.upd == nil {
		return false, errors.New("mise à jour indisponible")
	}
	if err := s.checkReplaceable(); err != nil {
		return false, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	switch st := s.upd.State(); st {
	case updater.StateDownloading, updater.StateVerifying, updater.StateInstalling:
		return true, nil // déjà en cours
	}
	rel, err := s.upd.Check(ctx)
	if err != nil {
		return false, err
	}
	if rel == nil {
		return false, nil
	}
	if err := s.upd.DownloadAndInstall(ctx); err != nil {
		if errors.Is(err, updater.ErrDownloadInProgress) {
			return true, nil // déjà en cours
		}
		return false, err
	}
	return true, s.upd.Restart(ctx)
}

// replaceTarget : ce que la mise à jour remplace (le .app sur Mac, l'exécutable ailleurs).
func replaceTarget(exe string) string {
	if i := strings.Index(exe, ".app"+string(os.PathSeparator)); i >= 0 {
		return exe[:i+len(".app")]
	}
	return exe
}

// checkReplaceable vérifie, avant de télécharger, que le dossier de l'app est modifiable : sinon le
// remplacement échouerait après la fermeture de Vinyl, sans message.
func (s *VinylService) checkReplaceable() error {
	exe, err := s.exePath()
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	f, err := os.CreateTemp(filepath.Dir(replaceTarget(exe)), ".vinyl-update-*")
	if err != nil {
		return errReadOnlyApp
	}
	f.Close()
	os.Remove(f.Name())
	return nil
}

// ReleaseURL : page GitHub d'une version (repli si l'installation échoue).
func (s *VinylService) ReleaseURL(version string) string {
	return "https://github.com/Peuraud-Pierre/vinyl/releases/tag/v" + strings.TrimPrefix(version, "v")
}

// BetaAccess : nom du contributeur si un code valide est enregistré, "" sinon.
func (s *VinylService) BetaAccess() string {
	if s.store == nil {
		return ""
	}
	name, _ := beta.Check(s.store.Get().BetaCode)
	return name
}

// SetBetaCode enregistre un code contributeur valide (vide : retire l'accès aux bêtas).
func (s *VinylService) SetBetaCode(code string) (string, error) {
	if s.store == nil {
		return "", errors.New("réglages indisponibles")
	}
	code = strings.TrimSpace(code)
	name := ""
	if code != "" {
		var ok bool
		if name, ok = beta.Check(code); !ok {
			return "", errors.New("code invalide")
		}
	}
	if _, err := s.store.Update(settings.Patch{BetaCode: &code}); err != nil {
		return "", err
	}
	return name, nil
}

// WatchUpdates vérifie régulièrement s'il existe une nouvelle version et l'annonce (sans la télécharger :
// l'installation attend le clic sur « Mettre à jour »). Rien en « dev » ni pendant une installation.
// La fonction renvoyée arrête la vérification.
func (s *VinylService) WatchUpdates(every time.Duration, announce func(UpdateInfo)) (stop func()) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		if s.version == "dev" || s.upd == nil {
			return
		}
		t := time.NewTicker(every)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				switch s.upd.State() {
				case updater.StateChecking, updater.StateDownloading, updater.StateVerifying,
					updater.StateInstalling, updater.StateReady:
					continue
				}
				if info, _ := s.CheckForUpdate(); info.Available {
					announce(info)
				}
			}
		}
	}()
	return func() { cancel(); <-done }
}
