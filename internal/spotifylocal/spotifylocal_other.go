//go:build !windows && !darwin

package spotifylocal

// Get is a no-op on platforms without a local-detection implementation;
// callers fall back to whatever else they have (e.g. the Web API).
func Get() Snapshot {
	return Snapshot{}
}
