package core

import (
	"os"
	"path/filepath"
)

type Paths struct {
	HomeDir           string
	DataDir           string
	LogFile           string
	HistoryFile       string
	AnalyticsFile     string
	LatestVersionFile string
}

var defaultPaths *Paths

func ensureDefaultPaths() {
	if defaultPaths == nil {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}

		defaultPaths = &Paths{
			HomeDir:           homeDir,
			DataDir:           filepath.Join(homeDir, ".local", "share", "bish"),
			LogFile:           filepath.Join(homeDir, ".local", "share", "bish", "gsh.log"),
			HistoryFile:       filepath.Join(homeDir, ".local", "share", "bish", "history.db"),
			AnalyticsFile:     filepath.Join(homeDir, ".local", "share", "bish", "analytics.db"),
			LatestVersionFile: filepath.Join(homeDir, ".local", "share", "bish", "latest_version.txt"),
		}

		err = os.MkdirAll(defaultPaths.DataDir, 0755)
		if err != nil {
			panic(err)
		}
	}
}

func HomeDir() string {
	ensureDefaultPaths()
	return defaultPaths.HomeDir
}

func DataDir() string {
	ensureDefaultPaths()
	return defaultPaths.DataDir
}

func LogFile() string {
	ensureDefaultPaths()
	return defaultPaths.LogFile
}

func HistoryFile() string {
	ensureDefaultPaths()
	return defaultPaths.HistoryFile
}

func AnalyticsFile() string {
	ensureDefaultPaths()
	return defaultPaths.AnalyticsFile
}

func LatestVersionFile() string {
	ensureDefaultPaths()
	return defaultPaths.LatestVersionFile
}
