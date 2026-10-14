package widgetupdate

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"slices"
	"testing"
	"time"
)

var testNow = time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

func TestDetect(t *testing.T) {
	cases := []struct {
		name string
		o    Observed
		want []Reason
	}{
		{"absent", Observed{SourceTree: "a", Now: testNow}, []Reason{NotInstalled}},
		{"à jour", Observed{Installed: true, Expiry: testNow.Add(72 * time.Hour), SourceTree: "a", InstalledTree: "a", Now: testNow}, nil},
		{"expirée", Observed{Installed: true, Expiry: testNow.Add(-time.Minute), SourceTree: "a", InstalledTree: "a", Now: testNow}, []Reason{Expired}},
		{"expire pile maintenant", Observed{Installed: true, Expiry: testNow, SourceTree: "a", InstalledTree: "a", Now: testNow}, []Reason{Expired}},
		{"expire bientôt", Observed{Installed: true, Expiry: testNow.Add(47 * time.Hour), SourceTree: "a", InstalledTree: "a", Now: testNow}, []Reason{ExpiresSoon}},
		{"nouvelle version", Observed{Installed: true, SourceTree: "b", InstalledTree: "a", Now: testNow}, []Reason{NewVersion}},
		{"expirée puis nouvelle version", Observed{Installed: true, Expiry: testNow.Add(-time.Hour), SourceTree: "b", InstalledTree: "a", Now: testNow}, []Reason{Expired, NewVersion}},
		{"expiration inconnue, dépôt introuvable", Observed{Installed: true, InstalledTree: "a", Now: testNow}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Detect(c.o); !slices.Equal(got, c.want) {
				t.Fatalf("Detect = %v, want %v", got, c.want)
			}
		})
	}
}

func TestMessage(t *testing.T) {
	cases := map[string]string{
		Message(NotInstalled, time.Time{}, testNow):              "Widget pas encore installé",
		Message(Expired, testNow, testNow):                       "Signature du widget expirée",
		Message(ExpiresSoon, testNow.Add(47*time.Hour), testNow): "Signature du widget : expire dans 2 jours",
		Message(ExpiresSoon, testNow.Add(5*time.Hour), testNow):  "Signature du widget : expire dans 1 jour",
		Message(NewVersion, time.Time{}, testNow):                "Nouvelle version du widget",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

func TestParseExpiry(t *testing.T) {
	plist := `<?xml version="1.0"?><plist version="1.0"><dict>
	<key>CreationDate</key>
	<date>2026-10-10T08:00:00Z</date>
	<key>ExpirationDate</key>
	<date>2026-10-17T08:00:00Z</date>
	</dict></plist>`
	got, ok := ParseExpiry(plist)
	if !ok || !got.Equal(time.Date(2026, 10, 17, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("ParseExpiry = %v, %v", got, ok)
	}
	if _, ok := ParseExpiry("pas un plist"); ok {
		t.Fatal("ParseExpiry doit échouer sur un texte quelconque")
	}
}

func makeCert(t *testing.T, ou string, notAfter time.Time) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "Apple Development: Test", OrganizationalUnit: []string{ou}},
		NotBefore:    notAfter.Add(-365 * 24 * time.Hour),
		NotAfter:     notAfter,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func TestTeamFromPEM(t *testing.T) {
	expired := makeCert(t, "OLDTEAM", testNow.Add(-time.Hour))
	valid := makeCert(t, "TEAM123", testNow.Add(300*24*time.Hour))
	if team, ok := TeamFromPEM(append(expired, valid...), testNow); !ok || team != "TEAM123" {
		t.Fatalf("TeamFromPEM = %q, %v", team, ok)
	}
	if _, ok := TeamFromPEM(expired, testNow); ok {
		t.Fatal("un certificat expiré ne doit pas compter")
	}
	if _, ok := TeamFromPEM(nil, testNow); ok {
		t.Fatal("aucun certificat : pas d'équipe")
	}
}
