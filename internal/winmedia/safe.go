package winmedia

import "fmt"

// safeRun exécute f en transformant une panique (appel système qui échoue mal) en erreur : un défaut
// dans la couche Windows ne doit jamais arrêter toute l'app.
func safeRun(f func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("winmedia: %v", r)
		}
	}()
	return f()
}
