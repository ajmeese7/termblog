// Package version provides build-time version information.
package version

// These variables are set at build time using ldflags.
// Example: go build -ldflags "-X github.com/ajmeese7/termblog/internal/version.Version=v0.1.0"
var (
	// Version is the semantic version (e.g., "0.1.0")
	Version = "dev"

	// Commit is the git commit SHA
	Commit = "unknown"

	// Date is the build date
	Date = "unknown"
)

// Info returns a formatted version string.
func Info() string {
	return Version
}

// Full returns detailed version information.
func Full() string {
	return "termblog " + Version + " (commit: " + Commit + ", built: " + Date + ")"
}
