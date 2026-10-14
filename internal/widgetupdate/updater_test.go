package widgetupdate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

type call struct {
	dir, name string
	args      []string
}

type fakeRunner struct {
	calls   []call
	respond func(c call) (string, error)
}

func (f *fakeRunner) Run(_ context.Context, dir, name string, args ...string) (string, error) {
	c := call{dir, name, args}
	f.calls = append(f.calls, c)
	return f.respond(c)
}

func (f *fakeRunner) names() []string {
	var n []string
	for _, c := range f.calls {
		n = append(n, filepath.Base(c.name))
	}
	return n
}

// defaultRespond simule une installation réussie ; certPEM = sortie de security find-certificate.
func defaultRespond(certPEM []byte) func(c call) (string, error) {
	return func(c call) (string, error) {
		switch filepath.Base(c.name) {
		case "security":
			if c.args[0] == "find-certificate" {
				return string(certPEM), nil
			}
			return "", errors.New("pas de profil")
		case "ditto":
			dst := c.args[1]
			os.MkdirAll(filepath.Join(dst, "Contents"), 0o755)
			return "", os.WriteFile(filepath.Join(dst, "Contents", "Info.plist"), []byte("new"), 0o644)
		case "git":
			return "tree123\n", nil
		}
		return "", nil
	}
}

func newTestUpdater(t *testing.T, r *fakeRunner) *Updater {
	t.Helper()
	root := t.TempDir()
	project := filepath.Join(root, "Document on Mac", "vinyl") // espace dans le chemin, comme chez Pierre
	os.MkdirAll(filepath.Join(project, "widget"), 0o755)
	os.WriteFile(filepath.Join(project, "widget", "project.yml"), []byte("name: VinylWidget\n"), 0o644)
	os.MkdirAll(filepath.Join(root, "bin"), 0o755)
	os.WriteFile(filepath.Join(root, "bin", "xcodegen"), nil, 0o755)
	os.MkdirAll(filepath.Join(root, "Applications"), 0o755)
	return &Updater{
		ProjectDir:    project,
		AppPath:       filepath.Join(root, "Applications", "Vinyl Widget.app"),
		BuildDir:      filepath.Join(root, "build"),
		RecordPath:    filepath.Join(root, "widget-install.json"),
		XcodeGenPaths: []string{filepath.Join(root, "absent", "xcodegen"), filepath.Join(root, "bin", "xcodegen")},
		Run:           r,
		Now:           func() time.Time { return testNow },
	}
}

func installOld(t *testing.T, u *Updater) {
	t.Helper()
	os.MkdirAll(filepath.Join(u.AppPath, "Contents"), 0o755)
	os.WriteFile(filepath.Join(u.AppPath, "Contents", "Info.plist"), []byte("old"), 0o644)
}

func readInfo(t *testing.T, u *Updater) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(u.AppPath, "Contents", "Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestUpdateSuccess(t *testing.T) {
	r := &fakeRunner{respond: defaultRespond(makeCert(t, "TEAM123", testNow.Add(time.Hour*24*300)))}
	u := newTestUpdater(t, r)
	installOld(t, u)

	if err := u.Update(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := readInfo(t, u); got != "new" {
		t.Fatalf("app installée = %q, want new", got)
	}
	entries, _ := os.ReadDir(filepath.Dir(u.AppPath))
	if len(entries) != 1 {
		t.Fatalf("fichiers temporaires restants : %v", entries)
	}
	var rec struct{ Tree string }
	b, _ := os.ReadFile(u.RecordPath)
	json.Unmarshal(b, &rec)
	if rec.Tree != "tree123" {
		t.Fatalf("empreinte enregistrée = %q", rec.Tree)
	}
	want := []string{"security", "xcodegen", "xcodebuild", "killall", "ditto", "open", "git"}
	if !slices.Equal(r.names(), want) {
		t.Fatalf("commandes = %v, want %v", r.names(), want)
	}
	gen, build := r.calls[1], r.calls[2]
	if gen.name != u.XcodeGenPaths[1] || gen.dir != filepath.Join(u.ProjectDir, "widget") {
		t.Fatalf("xcodegen = %+v", gen)
	}
	for _, a := range []string{"DEVELOPMENT_TEAM=TEAM123", "-allowProvisioningUpdates", "Vinyl Widget", "Release", u.BuildDir} {
		if !slices.Contains(build.args, a) {
			t.Errorf("xcodebuild sans %q : %v", a, build.args)
		}
	}
	if build.dir != filepath.Join(u.ProjectDir, "widget") {
		t.Errorf("xcodebuild lancé dans %q", build.dir)
	}
}

func TestUpdateMissingProject(t *testing.T) {
	r := &fakeRunner{respond: defaultRespond(nil)}
	u := newTestUpdater(t, r)
	u.ProjectDir = filepath.Join(u.ProjectDir, "ailleurs")
	err := u.Update(context.Background())
	if !strings.HasPrefix(ErrorMessage(err), "Dossier du projet introuvable : ") {
		t.Fatalf("message = %q", ErrorMessage(err))
	}
	if len(r.calls) != 0 {
		t.Fatalf("aucune commande attendue : %v", r.names())
	}
}

func TestUpdateMissingXcodeGen(t *testing.T) {
	r := &fakeRunner{respond: defaultRespond(nil)}
	u := newTestUpdater(t, r)
	u.XcodeGenPaths = u.XcodeGenPaths[:1]
	if got := ErrorMessage(u.Update(context.Background())); got != "Installe XcodeGen : brew install xcodegen" {
		t.Fatalf("message = %q", got)
	}
}

func TestUpdateNoCertificate(t *testing.T) {
	r := &fakeRunner{respond: defaultRespond(nil)}
	u := newTestUpdater(t, r)
	if got := ErrorMessage(u.Update(context.Background())); got != msgAccount {
		t.Fatalf("message = %q", got)
	}
	if slices.Contains(r.names(), "xcodebuild") {
		t.Fatal("xcodebuild ne doit pas être lancé sans équipe")
	}
}

func TestUpdateBuildFails(t *testing.T) {
	cert := makeCert(t, "TEAM123", testNow.Add(time.Hour*24*300))
	cases := []struct {
		name, out string
		check     func(msg string) bool
	}{
		{"compte", `error: No Account for Team "TEAM123"`, func(m string) bool { return m == msgAccount }},
		{"autre", strings.Repeat("ligne\n", 10) + func() string {
			var b strings.Builder
			for i := 1; i <= 30; i++ {
				fmt.Fprintf(&b, "erreur %d\n", i)
			}
			return b.String()
		}(), func(m string) bool {
			return strings.HasPrefix(m, "La compilation a échoué") && strings.Contains(m, "erreur 30") && !strings.Contains(m, "erreur 10\n")
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			base := defaultRespond(cert)
			r := &fakeRunner{respond: func(cl call) (string, error) {
				if filepath.Base(cl.name) == "xcodebuild" {
					return c.out, errors.New("exit status 65")
				}
				return base(cl)
			}}
			u := newTestUpdater(t, r)
			installOld(t, u)
			msg := ErrorMessage(u.Update(context.Background()))
			if !c.check(msg) {
				t.Fatalf("message = %q", msg)
			}
			if readInfo(t, u) != "old" {
				t.Fatal("l'ancienne version doit rester en place")
			}
		})
	}
}

func TestUpdateTimeout(t *testing.T) {
	base := defaultRespond(makeCert(t, "TEAM123", testNow.Add(time.Hour*24*300)))
	r := &fakeRunner{respond: func(c call) (string, error) {
		if filepath.Base(c.name) == "xcodebuild" {
			return "", context.DeadlineExceeded
		}
		return base(c)
	}}
	u := newTestUpdater(t, r)
	if got := ErrorMessage(u.Update(context.Background())); got != "La mise à jour a pris trop de temps" {
		t.Fatalf("message = %q", got)
	}
}

func TestUpdateCopyFailsKeepsOldApp(t *testing.T) {
	base := defaultRespond(makeCert(t, "TEAM123", testNow.Add(time.Hour*24*300)))
	r := &fakeRunner{respond: func(c call) (string, error) {
		if filepath.Base(c.name) == "ditto" {
			return "ditto: Permission denied", errors.New("exit status 1")
		}
		return base(c)
	}}
	u := newTestUpdater(t, r)
	installOld(t, u)
	if err := u.Update(context.Background()); err == nil {
		t.Fatal("erreur attendue")
	}
	if readInfo(t, u) != "old" {
		t.Fatal("l'ancienne version doit rester en place")
	}
}

func TestObserve(t *testing.T) {
	r := &fakeRunner{respond: func(c call) (string, error) {
		switch filepath.Base(c.name) {
		case "security":
			return "<key>ExpirationDate</key>\n<date>2026-10-17T08:00:00Z</date>", nil
		case "git":
			return "tree456\n", nil
		}
		return "", nil
	}}
	u := newTestUpdater(t, r)
	installOld(t, u)
	os.WriteFile(u.RecordPath, []byte(`{"tree":"tree123","installedAt":"2026-10-10T08:00:00Z"}`), 0o644)

	o := u.Observe(context.Background())
	want := Observed{Installed: true, Expiry: time.Date(2026, 10, 17, 8, 0, 0, 0, time.UTC),
		SourceTree: "tree456", InstalledTree: "tree123", Now: testNow}
	if o != want {
		t.Fatalf("Observe = %+v, want %+v", o, want)
	}
	if got := r.calls[0].args[len(r.calls[0].args)-1]; got != filepath.Join(u.AppPath, "Contents", "embedded.provisionprofile") {
		t.Fatalf("profil lu = %q", got)
	}
}

func TestObserveNotInstalled(t *testing.T) {
	r := &fakeRunner{respond: func(c call) (string, error) { return "", errors.New("introuvable") }}
	u := newTestUpdater(t, r)
	if o := u.Observe(context.Background()); o.Installed || o.SourceTree != "" || !o.Expiry.IsZero() {
		t.Fatalf("Observe = %+v", o)
	}
}
