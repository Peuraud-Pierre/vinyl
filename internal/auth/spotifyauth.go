package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/zalando/go-keyring"
)

const (
	RedirectURI = "http://127.0.0.1:43821/callback"
	scopes      = "user-library-read user-library-modify playlist-read-private user-follow-read user-read-playback-state user-modify-playback-state"
)

var ErrNotLoggedIn = errors.New("Spotify non connecté")

// DefaultClientID : l'app Vinyl enregistrée chez Spotify. Public par nature (connexion PKCE, sans
// secret) : l'utilisateur n'a rien à créer, il accepte juste l'accès dans la page Spotify.
const DefaultClientID = "9a0e3508a94e44658407da3a3cb7b59d"

// EffectiveClientID : le Client ID personnel s'il y en a un (Réglages > Avancé), sinon celui de Vinyl.
func EffectiveClientID(custom string) string {
	if c := strings.TrimSpace(custom); c != "" {
		return c
	}
	return DefaultClientID
}

// tokenHTTPError est une réponse non 200 du serveur de jetons Spotify.
type tokenHTTPError struct{ status int }

func (e *tokenHTTPError) Error() string { return fmt.Sprintf("spotify token: HTTP %d", e.status) }

type Token struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	Expiry       time.Time `json:"expiry"`
}

type TokenStore interface {
	Load() (Token, error)
	Save(Token) error
	Delete() error
}

// KeychainStore range le jeton dans le Trousseau macOS (service « Vinyl.Spotify »).
type KeychainStore struct{}

func (KeychainStore) Load() (Token, error) {
	s, err := keyring.Get("Vinyl.Spotify", "token")
	if err != nil {
		return Token{}, err
	}
	var t Token
	err = json.Unmarshal([]byte(s), &t)
	return t, err
}

func (KeychainStore) Save(t Token) error {
	b, _ := json.Marshal(t)
	return keyring.Set("Vinyl.Spotify", "token", string(b))
}

func (KeychainStore) Delete() error {
	err := keyring.Delete("Vinyl.Spotify", "token")
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

type SpotifyAuth struct {
	ClientID    func() string
	Store       TokenStore
	AuthURL     string
	TokenURL    string
	HTTP        *http.Client
	Now         func() time.Time
	OpenBrowser func(string) error
	mu          sync.Mutex
}

func NewSpotifyAuth(clientID func() string, store TokenStore) *SpotifyAuth {
	return &SpotifyAuth{
		ClientID:    clientID,
		Store:       store,
		AuthURL:     "https://accounts.spotify.com/authorize",
		TokenURL:    "https://accounts.spotify.com/api/token",
		HTTP:        &http.Client{Timeout: 10 * time.Second},
		Now:         time.Now,
		OpenBrowser: openBrowser, // propre à chaque système (openbrowser_*.go)
	}
}

func (a *SpotifyAuth) LoggedIn() bool {
	_, err := a.Store.Load()
	return err == nil
}

func (a *SpotifyAuth) Logout() error { return a.Store.Delete() }

// Login ouvre la page d'autorisation Spotify et attend le retour sur RedirectURI (OAuth PKCE).
func (a *SpotifyAuth) Login(ctx context.Context) error {
	clientID := EffectiveClientID(a.ClientID())
	verifier, err := NewVerifier()
	if err != nil {
		return err
	}
	state, err := NewVerifier()
	if err != nil {
		return err
	}
	ln, err := net.Listen("tcp", "127.0.0.1:43821")
	if err != nil {
		return fmt.Errorf("port 43821 occupé : %w", err)
	}
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/callback" {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		if q.Get("state") != state {
			http.Error(w, "state invalide", 400)
			select {
			case errCh <- errors.New("state OAuth invalide"):
			default:
			}
			return
		}
		if e := q.Get("error"); e != "" {
			fmt.Fprint(w, "Connexion annulée. Tu peux fermer cet onglet.")
			select {
			case errCh <- fmt.Errorf("Spotify : %s", e):
			default:
			}
			return
		}
		fmt.Fprint(w, "Connecté à Vinyl. Tu peux fermer cet onglet.")
		select {
		case codeCh <- q.Get("code"):
		default:
		}
	})}
	go srv.Serve(ln)
	defer srv.Close()

	q := url.Values{
		"client_id": {clientID}, "response_type": {"code"}, "redirect_uri": {RedirectURI},
		"code_challenge_method": {"S256"}, "code_challenge": {Challenge(verifier)},
		"state": {state}, "scope": {scopes},
	}
	if err := a.OpenBrowser(a.AuthURL + "?" + q.Encode()); err != nil {
		return err
	}
	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
	tok, err := a.exchange(ctx, url.Values{
		"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {RedirectURI},
		"client_id": {clientID}, "code_verifier": {verifier},
	}, "")
	if err != nil {
		return err
	}
	return a.Store.Save(tok)
}

func (a *SpotifyAuth) exchange(ctx context.Context, form url.Values, oldRefresh string) (Token, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", a.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return Token{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := a.HTTP.Do(req)
	if err != nil {
		return Token{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return Token{}, &tokenHTTPError{status: resp.StatusCode}
	}
	var body struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return Token{}, err
	}
	if body.RefreshToken == "" {
		body.RefreshToken = oldRefresh
	}
	return Token{AccessToken: body.AccessToken, RefreshToken: body.RefreshToken,
		Expiry: a.Now().Add(time.Duration(body.ExpiresIn)*time.Second - 30*time.Second)}, nil
}

// AccessToken renvoie un jeton valide, rafraîchi si besoin. Un refresh refusé déconnecte.
func (a *SpotifyAuth) AccessToken(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	tok, err := a.Store.Load()
	if err != nil {
		return "", ErrNotLoggedIn
	}
	if a.Now().Before(tok.Expiry) {
		return tok.AccessToken, nil
	}
	fresh, err := a.exchange(ctx, url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {tok.RefreshToken}, "client_id": {EffectiveClientID(a.ClientID())},
	}, tok.RefreshToken)
	if err != nil {
		// Seul un refus explicite (400/401 : refresh token révoqué ou invalide) déconnecte.
		// Réseau, 429 ou 5xx : erreur temporaire, on garde le jeton.
		var httpErr *tokenHTTPError
		if errors.As(err, &httpErr) && (httpErr.status == 400 || httpErr.status == 401) {
			a.Store.Delete()
			return "", ErrNotLoggedIn
		}
		return "", fmt.Errorf("Spotify temporairement indisponible : %w", err)
	}
	if err := a.Store.Save(fresh); err != nil {
		return "", err
	}
	return fresh.AccessToken, nil
}
