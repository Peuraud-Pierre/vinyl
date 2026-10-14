package widgetupdate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Runner exécute une commande et renvoie sa sortie combinée (stdout + stderr).
type Runner interface {
	Run(ctx context.Context, dir, name string, args ...string) (string, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	if ctx.Err() != nil {
		return out.String(), ctx.Err()
	}
	return out.String(), err
}

// UserError porte un message prêt à afficher.
type UserError struct{ Msg string }

func (e *UserError) Error() string { return e.Msg }

func userErr(format string, a ...any) error { return &UserError{Msg: fmt.Sprintf(format, a...)} }

const msgAccount = "Connecte ton compte Apple dans Xcode → Réglages → Comptes, puis installe une fois le widget depuis Xcode (voir widget/README.md)"

// ErrorMessage traduit une erreur de mise à jour en texte pour l'interface.
func ErrorMessage(err error) string {
	var ue *UserError
	switch {
	case err == nil:
		return ""
	case errors.As(err, &ue):
		return ue.Msg
	case errors.Is(err, context.DeadlineExceeded):
		return "La mise à jour a pris trop de temps"
	default:
		return err.Error()
	}
}

type Updater struct {
	ProjectDir    string   // racine du dépôt Vinyl (contient widget/)
	AppPath       string   // /Applications/Vinyl Widget.app
	BuildDir      string   // -derivedDataPath
	RecordPath    string   // widget-install.json
	XcodeGenPaths []string // emplacements possibles de xcodegen
	Run           Runner
	Now           func() time.Time
}

type record struct {
	Tree        string    `json:"tree"`
	InstalledAt time.Time `json:"installedAt"`
}

func (u *Updater) widgetDir() string { return filepath.Join(u.ProjectDir, "widget") }

func (u *Updater) sourceTree(ctx context.Context) string {
	out, err := u.Run.Run(ctx, "", "/usr/bin/git", "-C", u.ProjectDir, "rev-parse", "HEAD:widget")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// Observe relève l'état sur le disque, sans rien modifier.
func (u *Updater) Observe(ctx context.Context) Observed {
	o := Observed{Now: u.Now()}
	if _, err := os.Stat(u.AppPath); err == nil {
		o.Installed = true
		profile := filepath.Join(u.AppPath, "Contents", "embedded.provisionprofile")
		if out, err := u.Run.Run(ctx, "", "/usr/bin/security", "cms", "-D", "-i", profile); err == nil {
			if t, ok := ParseExpiry(out); ok {
				o.Expiry = t
			}
		}
	}
	o.SourceTree = u.sourceTree(ctx)
	var rec record
	if b, err := os.ReadFile(u.RecordPath); err == nil && json.Unmarshal(b, &rec) == nil {
		o.InstalledTree = rec.Tree
	}
	return o
}

// Update recompile le widget et remplace l'app installée. L'ancienne version reste en place en cas d'échec.
func (u *Updater) Update(ctx context.Context) error {
	if fi, err := os.Stat(filepath.Join(u.widgetDir(), "project.yml")); err != nil || fi.IsDir() {
		return userErr("Dossier du projet introuvable : %s. Corrige widgetProjectDir dans settings.json", u.ProjectDir)
	}
	xcodegen := ""
	for _, p := range u.XcodeGenPaths {
		if _, err := os.Stat(p); err == nil {
			xcodegen = p
			break
		}
	}
	if xcodegen == "" {
		return userErr("Installe XcodeGen : brew install xcodegen")
	}
	pems, _ := u.Run.Run(ctx, "", "/usr/bin/security", "find-certificate", "-a", "-c", "Apple Development", "-p")
	team, ok := TeamFromPEM([]byte(pems), u.Now())
	if !ok {
		return userErr(msgAccount)
	}
	if out, err := u.Run.Run(ctx, u.widgetDir(), xcodegen, "generate"); err != nil {
		return failure(err, "La génération du projet a échoué", out)
	}
	out, err := u.Run.Run(ctx, u.widgetDir(), "/usr/bin/xcodebuild",
		"-project", "VinylWidget.xcodeproj", "-scheme", "Vinyl Widget", "-configuration", "Release",
		"-derivedDataPath", u.BuildDir, "-allowProvisioningUpdates", "DEVELOPMENT_TEAM="+team,
		// Version croissante : si plusieurs copies de l'extension sont enregistrées, macOS prend la plus récente.
		fmt.Sprintf("CURRENT_PROJECT_VERSION=%d", u.Now().Unix()), "build")
	if err != nil {
		if isAccountProblem(out) {
			return userErr(msgAccount)
		}
		return failure(err, "La compilation a échoué", out)
	}
	built := filepath.Join(u.BuildDir, "Build", "Products", "Release", "Vinyl Widget.app")
	u.Run.Run(ctx, "", "/usr/bin/killall", "Vinyl Widget") // erreur ignorée : l'assistant peut être arrêté
	if err := u.install(ctx, built); err != nil {
		if _, statErr := os.Stat(u.AppPath); statErr == nil {
			u.Run.Run(ctx, "", "/usr/bin/open", "-a", u.AppPath) // l'ancienne version reprend du service
		}
		return err
	}
	// Une seule copie de l'app sur le disque : sinon macOS peut enregistrer l'extension de la copie de build.
	os.RemoveAll(built)
	if out, err := u.Run.Run(ctx, "", "/usr/bin/open", "-a", u.AppPath); err != nil {
		return failure(err, "Widget installé, mais impossible de relancer l'assistant", out)
	}
	return u.saveRecord(ctx)
}

// install copie l'app compilée à côté de l'ancienne puis échange les deux par renommage.
func (u *Updater) install(ctx context.Context, built string) error {
	dir, base := filepath.Dir(u.AppPath), filepath.Base(u.AppPath)
	tmp, old := filepath.Join(dir, "."+base+".new"), filepath.Join(dir, "."+base+".old")
	os.RemoveAll(tmp)
	os.RemoveAll(old)
	if out, err := u.Run.Run(ctx, "", "/usr/bin/ditto", built, tmp); err != nil {
		os.RemoveAll(tmp)
		return failure(err, "Copie impossible vers "+dir, out)
	}
	_, statErr := os.Stat(u.AppPath)
	hadOld := statErr == nil
	if hadOld {
		if err := os.Rename(u.AppPath, old); err != nil {
			os.RemoveAll(tmp)
			return userErr("Remplacement impossible de %s : %v", u.AppPath, err)
		}
	}
	if err := os.Rename(tmp, u.AppPath); err != nil {
		if hadOld {
			os.Rename(old, u.AppPath)
		}
		os.RemoveAll(tmp)
		return userErr("Remplacement impossible de %s : %v", u.AppPath, err)
	}
	os.RemoveAll(old)
	return nil
}

func (u *Updater) saveRecord(ctx context.Context) error {
	b, _ := json.Marshal(record{Tree: u.sourceTree(ctx), InstalledAt: u.Now()})
	if err := os.MkdirAll(filepath.Dir(u.RecordPath), 0o755); err != nil {
		return err
	}
	tmp := u.RecordPath + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, u.RecordPath)
}

func isAccountProblem(out string) bool {
	low := strings.ToLower(out)
	for _, k := range []string{"no account", "no accounts", "requires a development team", "no signing certificate", "not signed in"} {
		if strings.Contains(low, k) {
			return true
		}
	}
	return false
}

// failure garde l'erreur de délai telle quelle (traduite par ErrorMessage), sinon ajoute la fin de la sortie.
func failure(err error, what, out string) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return err
	}
	return userErr("%s :\n%s", what, lastLines(out, 20))
}

func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
