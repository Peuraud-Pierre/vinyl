// Package widgetupdate détecte quand le widget de bureau doit être réinstallé, et le réinstalle.
package widgetupdate

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math"
	"regexp"
	"time"
)

type Reason string

const (
	NotInstalled Reason = "notInstalled"
	Expired      Reason = "expired"
	ExpiresSoon  Reason = "expiresSoon"
	NewVersion   Reason = "newVersion"
)

// expirySoon : en dessous de ce délai, on propose de re-signer.
const expirySoon = 48 * time.Hour

// Observed décrit ce qu'on a constaté sur le disque. Valeur zéro d'un champ = inconnue.
type Observed struct {
	Installed     bool
	Expiry        time.Time // ExpirationDate du profil intégré
	SourceTree    string    // git rev-parse HEAD:widget
	InstalledTree string    // empreinte enregistrée à la dernière installation
	Now           time.Time
}

// Detect renvoie les raisons de mettre à jour, par ordre de priorité.
func Detect(o Observed) []Reason {
	if !o.Installed {
		return []Reason{NotInstalled}
	}
	var r []Reason
	if !o.Expiry.IsZero() {
		if !o.Now.Before(o.Expiry) {
			r = append(r, Expired)
		} else if o.Expiry.Sub(o.Now) < expirySoon {
			r = append(r, ExpiresSoon)
		}
	}
	if o.SourceTree != "" && o.SourceTree != o.InstalledTree {
		r = append(r, NewVersion)
	}
	return r
}

// Message est le texte de la bulle au survol de l'icône.
func Message(r Reason, expiry, now time.Time) string {
	switch r {
	case NotInstalled:
		return "Widget pas encore installé"
	case Expired:
		return "Signature du widget expirée"
	case ExpiresSoon:
		days := max(1, int(math.Ceil(expiry.Sub(now).Hours()/24)))
		s := ""
		if days > 1 {
			s = "s"
		}
		return fmt.Sprintf("Signature du widget : expire dans %d jour%s", days, s)
	case NewVersion:
		return "Nouvelle version du widget"
	}
	return ""
}

var expiryRe = regexp.MustCompile(`<key>ExpirationDate</key>\s*<date>([^<]+)</date>`)

// ParseExpiry lit ExpirationDate dans un profil de provisionnement décodé (security cms -D).
func ParseExpiry(plist string) (time.Time, bool) {
	m := expiryRe.FindStringSubmatch(plist)
	if m == nil {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, m[1])
	return t, err == nil
}

// TeamFromPEM renvoie l'équipe (champ OU) du premier certificat non expiré.
func TeamFromPEM(data []byte, now time.Time) (string, bool) {
	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			return "", false
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil || now.After(cert.NotAfter) || len(cert.Subject.OrganizationalUnit) == 0 {
			continue
		}
		return cert.Subject.OrganizationalUnit[0], true
	}
}
