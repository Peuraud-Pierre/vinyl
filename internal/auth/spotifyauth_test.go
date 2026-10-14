package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type memStore struct {
	tok Token
	has bool
}

func (m *memStore) Load() (Token, error) {
	if !m.has {
		return Token{}, errors.New("absent")
	}
	return m.tok, nil
}
func (m *memStore) Save(t Token) error { m.tok, m.has = t, true; return nil }
func (m *memStore) Delete() error      { m.has = false; return nil }

func TestAccessTokenNotLoggedIn(t *testing.T) {
	a := NewSpotifyAuth(func() string { return "cid" }, &memStore{})
	if _, err := a.AccessToken(context.Background()); !errors.Is(err, ErrNotLoggedIn) {
		t.Errorf("got %v", err)
	}
}

func TestAccessTokenValidIsReturned(t *testing.T) {
	now := time.Unix(1000, 0)
	st := &memStore{tok: Token{AccessToken: "A", RefreshToken: "R", Expiry: now.Add(time.Hour)}, has: true}
	a := NewSpotifyAuth(func() string { return "cid" }, st)
	a.Now = func() time.Time { return now }
	tok, err := a.AccessToken(context.Background())
	if err != nil || tok != "A" {
		t.Errorf("got %q %v", tok, err)
	}
}

func TestAccessTokenRefreshesWhenExpired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "R" || r.Form.Get("client_id") != "cid" {
			http.Error(w, "bad", 400)
			return
		}
		w.Write([]byte(`{"access_token":"B","expires_in":3600}`))
	}))
	defer srv.Close()
	now := time.Unix(1000, 0)
	st := &memStore{tok: Token{AccessToken: "A", RefreshToken: "R", Expiry: now.Add(-time.Minute)}, has: true}
	a := NewSpotifyAuth(func() string { return "cid" }, st)
	a.Now, a.TokenURL, a.HTTP = func() time.Time { return now }, srv.URL, srv.Client()
	tok, err := a.AccessToken(context.Background())
	if err != nil || tok != "B" {
		t.Fatalf("got %q %v", tok, err)
	}
	if st.tok.RefreshToken != "R" {
		t.Errorf("le refresh token doit être conservé quand Spotify n'en renvoie pas de nouveau")
	}
}

func TestRefreshRejectedLogsOut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, `{"error":"invalid_grant"}`, 400) }))
	defer srv.Close()
	st := &memStore{tok: Token{AccessToken: "A", RefreshToken: "R"}, has: true}
	a := NewSpotifyAuth(func() string { return "cid" }, st)
	a.TokenURL, a.HTTP = srv.URL, srv.Client()
	if _, err := a.AccessToken(context.Background()); !errors.Is(err, ErrNotLoggedIn) {
		t.Errorf("got %v", err)
	}
	if st.has {
		t.Error("un refresh refusé doit effacer le jeton")
	}
}

func TestLoginRequiresClientID(t *testing.T) {
	a := NewSpotifyAuth(func() string { return "" }, &memStore{})
	if err := a.Login(context.Background()); err == nil {
		t.Error("Client ID vide → erreur attendue")
	}
}

func TestLoginFlow(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("code") != "CODE" || r.Form.Get("code_verifier") == "" || r.Form.Get("redirect_uri") != RedirectURI {
			http.Error(w, "bad", 400)
			return
		}
		w.Write([]byte(`{"access_token":"A","refresh_token":"R","expires_in":3600}`))
	}))
	defer tokenSrv.Close()
	st := &memStore{}
	a := NewSpotifyAuth(func() string { return "cid" }, st)
	a.TokenURL, a.HTTP = tokenSrv.URL, tokenSrv.Client()
	a.OpenBrowser = func(authURL string) error {
		// Simule Spotify qui redirige vers notre callback avec le même state.
		go func() {
			u, _ := http.NewRequest("GET", authURL, nil)
			state := u.URL.Query().Get("state")
			http.Get(RedirectURI + "?code=CODE&state=" + state)
		}()
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.Login(ctx); err != nil {
		t.Fatal(err)
	}
	if !st.has || st.tok.AccessToken != "A" {
		t.Errorf("jeton non enregistré : %+v", st.tok)
	}
}

func TestRefreshServerErrorKeepsToken(t *testing.T) {
	for _, code := range []int{429, 500, 503} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "indisponible", code) }))
		st := &memStore{tok: Token{AccessToken: "A", RefreshToken: "R"}, has: true}
		a := NewSpotifyAuth(func() string { return "cid" }, st)
		a.TokenURL, a.HTTP = srv.URL, srv.Client()
		_, err := a.AccessToken(context.Background())
		srv.Close()
		if err == nil || errors.Is(err, ErrNotLoggedIn) {
			t.Errorf("HTTP %d : erreur temporaire attendue, obtenu %v", code, err)
		}
		if !st.has {
			t.Errorf("HTTP %d : le jeton ne doit pas être effacé", code)
		}
	}
}
