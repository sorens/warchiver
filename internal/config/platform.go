package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const appID = "com.warchiver"

// ConfigDir returns the platform-specific application config directory.
// macOS:   ~/Library/Application Support/com.warchiver/
// Windows: %APPDATA%\com.warchiver\
// Linux:   ~/.config/com.warchiver/
func ConfigDir() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("getting home dir: %w", err)
		}
		return filepath.Join(home, "Library", "Application Support", appID), nil
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		return filepath.Join(appdata, appID), nil
	default: // linux and others
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("getting home dir: %w", err)
		}
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig != "" {
			return filepath.Join(xdgConfig, appID), nil
		}
		return filepath.Join(home, ".config", appID), nil
	}
}

// ArchiveDir returns the directory where warchive files are stored.
func ArchiveDir() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "archives"), nil
}

// ConfigFilePath returns the path to the main config JSON file.
func ConfigFilePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// EnsureDirs creates the config and archive directories if they don't exist.
func EnsureDirs() error {
	archiveDir, err := ArchiveDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(archiveDir, 0700)
}
