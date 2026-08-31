// Package spotify reports the track currently playing on Spotify, so
// Elysia can react to it.
//
// The primary source is spotifylocal, which reads the local desktop app
// directly — no setup, and it works on Free accounts. There's also an
// optional Web API OAuth path (needs a one-time Spotify Developer "app"
// Client ID, see README) used as a fallback when spotifylocal finds
// nothing; note the Web API's player-state endpoints are Premium-only, so
// that path is only useful to Premium users who specifically want it.
package spotify

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"elygochi/internal/spotifylocal"
)

const (
	redirectPort  = "8888"
	redirectURI   = "http://127.0.0.1:" + redirectPort + "/callback"
	authURL       = "https://accounts.spotify.com/authorize"
	tokenURL      = "https://accounts.spotify.com/api/token"
	nowPlayingURL = "https://api.spotify.com/v1/me/player/currently-playing"
	scopes        = "user-read-currently-playing user-read-playback-state"

	pollInterval      = 5 * time.Second
	localPollInterval = 2 * time.Second
)

// Snapshot is the latest known playback state.
type Snapshot struct {
	Track   string
	Artist  string
	Playing bool
	Ready   bool // false until the service has authenticated and polled once
}

// Service polls Spotify in the background. Zero value is safe but inert;
// use Start.
type Service struct {
	mu       sync.RWMutex
	snap     Snapshot
	clientID string

	accessToken string
	expiresAt   time.Time
}

// Start launches the background poll loop(s). It always polls the local
// Spotify desktop app directly (spotifylocal) — no setup, and it works on
// Free accounts, unlike the Web API's player-state endpoints which are
// Premium-only. If a Client ID has also been configured (see SaveClientID),
// it additionally runs the Web API OAuth flow as a secondary source, used
// whenever the local read comes up empty (e.g. Spotify running on a
// different device on the same network isn't visible locally, but might
// still be reachable through the account's Web API state).
func Start() *Service {
	s := &Service{}
	go s.runLocal()

	clientID, err := loadClientID()
	if err == nil && clientID != "" {
		s.clientID = clientID
		go s.run()
	}
	return s
}

// runLocal polls the local Spotify desktop app. Whenever it gets a
// definitive local read (Spotify is running on this machine), that's
// authoritative — it overwrites whatever the Web API loop last reported,
// since it's simpler, needs no auth, and reflects this machine's Spotify
// specifically.
func (s *Service) runLocal() {
	poll := func() {
		local := spotifylocal.Get()
		if !local.OK {
			return
		}
		s.mu.Lock()
		s.snap = Snapshot{Track: local.Track, Artist: local.Artist, Playing: local.Playing, Ready: true}
		s.mu.Unlock()
	}
	poll()
	t := time.NewTicker(localPollInterval)
	for range t.C {
		poll()
	}
}

// Snapshot returns the most recently polled playback state.
func (s *Service) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snap
}

func (s *Service) run() {
	refreshToken, _ := loadRefreshToken()
	if refreshToken == "" {
		var err error
		refreshToken, err = s.authorize()
		if err != nil {
			fmt.Fprintf(os.Stderr, "spotify: authorization failed: %v\n", err)
			return
		}
		_ = saveRefreshToken(refreshToken)
	}

	if err := s.refreshAccessToken(refreshToken); err != nil {
		fmt.Fprintf(os.Stderr, "spotify: token refresh failed: %v\n", err)
		return
	}

	t := time.NewTicker(pollInterval)
	for range t.C {
		if time.Now().After(s.expiresAt) {
			if err := s.refreshAccessToken(refreshToken); err != nil {
				fmt.Fprintf(os.Stderr, "spotify: token refresh failed: %v\n", err)
				continue
			}
		}
		s.poll()
	}
}

func (s *Service) poll() {
	// The local desktop-app read is authoritative when it's available;
	// don't let a slightly-stale Web API response clobber it.
	if spotifylocal.Get().OK {
		return
	}
	req, _ := http.NewRequest(http.MethodGet, nowPlayingURL, nil)
	req.Header.Set("Authorization", "Bearer "+s.accessTokenSnapshot())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		s.mu.Lock()
		s.snap = Snapshot{Ready: true}
		s.mu.Unlock()
		return
	}
	if resp.StatusCode != http.StatusOK {
		return
	}

	var out struct {
		IsPlaying bool `json:"is_playing"`
		Item      struct {
			Name    string `json:"name"`
			Artists []struct {
				Name string `json:"name"`
			} `json:"artists"`
		} `json:"item"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return
	}

	artist := ""
	if len(out.Item.Artists) > 0 {
		artist = out.Item.Artists[0].Name
	}

	s.mu.Lock()
	s.snap = Snapshot{Track: out.Item.Name, Artist: artist, Playing: out.IsPlaying, Ready: true}
	s.mu.Unlock()
}

func (s *Service) accessTokenSnapshot() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.accessToken
}

func (s *Service) refreshAccessToken(refreshToken string) error {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {s.clientID},
	}
	tok, err := postToken(form)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.accessToken = tok.AccessToken
	s.expiresAt = time.Now().Add(time.Duration(tok.ExpiresIn-30) * time.Second)
	s.mu.Unlock()
	return nil
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func postToken(form url.Values) (*tokenResponse, error) {
	resp, err := http.PostForm(tokenURL, form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify: token endpoint status %d", resp.StatusCode)
	}
	var tok tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, err
	}
	return &tok, nil
}

// authorize runs the one-time PKCE authorization-code flow: opens the
// user's browser to Spotify's consent page and waits for the local
// redirect carrying the auth code, then exchanges it for tokens.
func (s *Service) authorize() (refreshToken string, err error) {
	verifier := randomVerifier()
	challenge := codeChallenge(verifier)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if e := r.URL.Query().Get("error"); e != "" {
			errCh <- fmt.Errorf("spotify: authorization denied: %s", e)
			fmt.Fprint(w, "Отказано в доступе. Можно закрыть эту вкладку.")
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("spotify: no code in callback")
			return
		}
		codeCh <- code
		fmt.Fprint(w, "Готово! Элизия теперь видит, что вы слушаете. Можно закрыть эту вкладку.")
	})
	srv := &http.Server{Addr: "127.0.0.1:" + redirectPort, Handler: mux}
	go srv.ListenAndServe()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	authorizeURL := authURL + "?" + url.Values{
		"client_id":             {s.clientID},
		"response_type":         {"code"},
		"redirect_uri":          {redirectURI},
		"code_challenge_method": {"S256"},
		"code_challenge":        {challenge},
		"scope":                 {scopes},
	}.Encode()
	if err := openBrowser(authorizeURL); err != nil {
		fmt.Fprintf(os.Stderr, "spotify: couldn't open a browser automatically, open this URL yourself:\n%s\n", authorizeURL)
	}

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return "", err
	case <-time.After(3 * time.Minute):
		return "", fmt.Errorf("spotify: timed out waiting for browser authorization")
	}

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {s.clientID},
		"code_verifier": {verifier},
	}
	tok, err := postToken(form)
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	s.accessToken = tok.AccessToken
	s.expiresAt = time.Now().Add(time.Duration(tok.ExpiresIn-30) * time.Second)
	s.mu.Unlock()
	return tok.RefreshToken, nil
}

// openBrowser opens url in the user's default browser, on whichever OS
// this happens to be running on.
func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default: // linux and friends
		return exec.Command("xdg-open", url).Start()
	}
}

func randomVerifier() string {
	b := make([]byte, 64)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func codeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func configDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "elygochi")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func loadClientID() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(dir, "spotify_client_id.txt"))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// ClientID returns the currently configured Client ID, or "" if none is
// set, for display in settings UI.
func ClientID() string {
	id, _ := loadClientID()
	return id
}

// SaveClientID persists the given Client ID. Callers should re-run Start
// afterwards to (re)launch the auth/poll loop with it.
func SaveClientID(id string) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "spotify_client_id.txt"), []byte(strings.TrimSpace(id)), 0o644)
}

func loadRefreshToken() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(dir, "spotify_refresh_token.txt"))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func saveRefreshToken(token string) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "spotify_refresh_token.txt"), []byte(token), 0o600)
}
