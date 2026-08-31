// Package weather fetches the current weather for a city name the user
// enters, rather than guessing a location automatically. The chosen city is
// remembered on disk between runs.
package weather

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ErrNoCity means no city has been configured yet.
var ErrNoCity = errors.New("weather: no city configured")

// Snapshot is the latest known weather reading.
type Snapshot struct {
	City    string
	TempC   float64
	Desc    string
	Updated time.Time
	Err     error
}

// Ready reports whether a successful reading has ever been taken.
func (s Snapshot) Ready() bool {
	return !s.Updated.IsZero() && s.Err == nil
}

const refreshInterval = 20 * time.Minute

var httpClient = &http.Client{Timeout: 6 * time.Second}

// Service periodically refreshes the weather for a configured city in the
// background so callers never block waiting on the network.
type Service struct {
	mu   sync.RWMutex
	snap Snapshot
	city string
}

// Start launches a Service. It loads a previously-saved city (if any) and
// fetches its weather immediately; otherwise it sits idle until SetCity is
// called.
func Start() *Service {
	s := &Service{}
	if city, err := loadCity(); err == nil && city != "" {
		s.city = city
		go s.refresh()
	}
	go s.loop()
	return s
}

func (s *Service) loop() {
	t := time.NewTicker(refreshInterval)
	for range t.C {
		if s.City() != "" {
			s.refresh()
		}
	}
}

// City returns the currently configured city name, or "" if none is set.
func (s *Service) City() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.city
}

// SetCity changes the configured city, persists it, and kicks off an
// immediate refresh in the background.
func (s *Service) SetCity(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	s.mu.Lock()
	s.city = name
	s.mu.Unlock()
	_ = saveCity(name)
	go s.refresh()
}

func (s *Service) refresh() {
	city := s.City()
	if city == "" {
		return
	}
	snap := fetch(city)
	s.mu.Lock()
	s.snap = snap
	s.mu.Unlock()
}

// Snapshot returns the most recently fetched weather reading.
func (s *Service) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snap
}

func fetch(city string) Snapshot {
	loc, err := geocode(city)
	if err != nil {
		return Snapshot{Err: err, Updated: time.Now()}
	}
	temp, code, err := currentWeather(loc.Lat, loc.Lon)
	if err != nil {
		return Snapshot{Err: err, Updated: time.Now()}
	}
	return Snapshot{
		City:    loc.Name,
		TempC:   temp,
		Desc:    describe(code),
		Updated: time.Now(),
	}
}

type location struct {
	Name string
	Lat  float64
	Lon  float64
}

func geocode(city string) (location, error) {
	var out struct {
		Results []struct {
			Name      string  `json:"name"`
			Admin1    string  `json:"admin1"`
			Country   string  `json:"country"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"results"`
	}
	u := "https://geocoding-api.open-meteo.com/v1/search?name=" + url.QueryEscape(city) + "&count=1&language=ru"
	if err := getJSON(u, &out); err != nil {
		return location{}, err
	}
	if len(out.Results) == 0 {
		return location{}, fmt.Errorf("weather: город %q не найден", city)
	}
	r := out.Results[0]
	return location{Name: r.Name, Lat: r.Latitude, Lon: r.Longitude}, nil
}

func currentWeather(lat, lon float64) (tempC float64, code int, err error) {
	var out struct {
		CurrentWeather struct {
			Temperature float64 `json:"temperature"`
			WeatherCode int     `json:"weathercode"`
		} `json:"current_weather"`
	}
	u := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&current_weather=true", lat, lon)
	if err := getJSON(u, &out); err != nil {
		return 0, 0, err
	}
	return out.CurrentWeather.Temperature, out.CurrentWeather.WeatherCode, nil
}

func getJSON(url string, out any) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("weather: %s: status %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// describe maps a WMO weather code (used by Open-Meteo) to a short Russian
// description.
func describe(code int) string {
	switch {
	case code == 0:
		return "ясно"
	case code <= 3:
		return "облачно"
	case code == 45 || code == 48:
		return "туман"
	case code >= 51 && code <= 57:
		return "морось"
	case code >= 61 && code <= 67:
		return "дождь"
	case code >= 71 && code <= 77:
		return "снег"
	case code >= 80 && code <= 82:
		return "ливень"
	case code >= 85 && code <= 86:
		return "снегопад"
	case code >= 95:
		return "гроза"
	default:
		return "погода переменчива"
	}
}

func cityConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "tamagotchi")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "city.txt"), nil
}

func loadCity() (string, error) {
	path, err := cityConfigPath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func saveCity(name string) error {
	path, err := cityConfigPath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(name), 0o644)
}
