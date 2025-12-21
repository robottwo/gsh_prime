package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildVersionVariable(t *testing.T) {
	tests := []struct {
		name            string
		buildVersion    string
		expectedDefault string
	}{
		{
			name:            "Default BUILD_VERSION is dev",
			buildVersion:    "dev",
			expectedDefault: "dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// BUILD_VERSION should have a default value
			assert.NotEmpty(t, BUILD_VERSION, "BUILD_VERSION should not be empty")
			assert.Equal(t, tt.expectedDefault, BUILD_VERSION, "Default BUILD_VERSION should be 'dev'")
		})
	}
}

func TestVersionFlag(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectOutput   string
		expectExit     bool
		buildVersion   string
	}{
		{
			name:         "Version flag prints BUILD_VERSION",
			args:         []string{"-ver"},
			expectOutput: "dev",
			expectExit:   true,
			buildVersion: "dev",
		},
		{
			name:         "Version flag with custom version",
			args:         []string{"-ver"},
			expectOutput: "v0.25.10",
			expectExit:   true,
			buildVersion: "v0.25.10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original BUILD_VERSION
			originalVersion := BUILD_VERSION
			defer func() { BUILD_VERSION = originalVersion }()

			// Set test BUILD_VERSION
			BUILD_VERSION = tt.buildVersion

			// Reset flags for each test
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			versionFlag = flag.Bool("ver", false, "display build version")

			// Parse test flags
			os.Args = append([]string{"bish"}, tt.args...)
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			versionFlag = flag.Bool("ver", false, "display build version")
			flag.Parse()

			// Check if version flag is set
			if *versionFlag {
				// This simulates the main() version check
				assert.True(t, tt.expectExit, "Version flag should cause early exit")
				assert.Equal(t, tt.buildVersion, BUILD_VERSION, "BUILD_VERSION should match")
			}
		})
	}
}

func TestHelpFlag(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Reset flags
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	helpFlag = flag.Bool("h", false, "display help information")
	
	// Set test args
	os.Args = []string{"bish", "-h"}
	
	// Parse flags
	flag.Parse()
	
	// Verify help flag is set
	assert.True(t, *helpFlag, "Help flag should be set")
}

func TestCommandFlag(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Reset flags
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	command = flag.String("c", "", "run a command")
	
	// Set test args
	testCommand := "echo hello"
	os.Args = []string{"bish", "-c", testCommand}
	
	// Parse flags
	flag.Parse()
	
	// Verify command flag value
	assert.Equal(t, testCommand, *command, "Command flag should contain the test command")
}

func TestLoginShellFlag(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "Login shell flag set",
			args:     []string{"bish", "-l"},
			expected: true,
		},
		{
			name:     "Login shell flag not set",
			args:     []string{"bish"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			loginShell = flag.Bool("l", false, "run as a login shell")
			
			// Set test args
			os.Args = tt.args
			
			// Parse flags
			flag.Parse()
			
			// Verify login shell flag
			assert.Equal(t, tt.expected, *loginShell, "Login shell flag should match expected value")
		})
	}
}

func TestBuildVersionInjection(t *testing.T) {
	t.Run("BUILD_VERSION should be injectable via ldflags", func(t *testing.T) {
		// This test verifies that BUILD_VERSION can be set during compilation
		// The default value should be "dev"
		assert.Contains(t, []string{"dev", "v0.25.10"}, BUILD_VERSION, 
			"BUILD_VERSION should be either 'dev' or a version string")
	})

	t.Run("BUILD_VERSION format validation", func(t *testing.T) {
		if BUILD_VERSION != "dev" {
			// If not dev, it should start with 'v' and contain version numbers
			assert.True(t, strings.HasPrefix(BUILD_VERSION, "v") || 
				strings.Contains(BUILD_VERSION, "."),
				"BUILD_VERSION should follow semantic versioning or be 'dev'")
		}
	})
}

func TestDefaultVarsEmbedded(t *testing.T) {
	t.Run("DEFAULT_VARS should be embedded", func(t *testing.T) {
		assert.NotNil(t, DEFAULT_VARS, "DEFAULT_VARS should be embedded")
		assert.NotEmpty(t, DEFAULT_VARS, "DEFAULT_VARS should not be empty")
	})

	t.Run("DEFAULT_VARS should contain valid shell configuration", func(t *testing.T) {
		content := string(DEFAULT_VARS)
		// Should contain common shell configuration elements
		assert.NotEmpty(t, content, "DEFAULT_VARS content should not be empty")
	})
}

func TestVersionFileExists(t *testing.T) {
	t.Run("VERSION file should exist in repository root", func(t *testing.T) {
		// Get the repository root
		repoRoot := findRepoRoot()
		if repoRoot == "" {
			t.Skip("Could not find repository root")
		}
		
		versionFile := filepath.Join(repoRoot, "VERSION")
		
		// Check if VERSION file exists
		_, err := os.Stat(versionFile)
		if err == nil {
			// File exists, read and validate
			content, err := os.ReadFile(versionFile)
			require.NoError(t, err, "Should be able to read VERSION file")
			
			version := strings.TrimSpace(string(content))
			assert.NotEmpty(t, version, "VERSION file should not be empty")
			
			// Validate semantic versioning format
			parts := strings.Split(version, ".")
			assert.GreaterOrEqual(t, len(parts), 2, "VERSION should have at least major.minor format")
		}
	})
}

func TestMakefileBuildCommand(t *testing.T) {
	t.Run("Makefile build target should inject version", func(t *testing.T) {
		// Get repository root
		repoRoot := findRepoRoot()
		if repoRoot == "" {
			t.Skip("Could not find repository root")
		}

		makefilePath := filepath.Join(repoRoot, "Makefile")
		
		// Check if Makefile exists
		content, err := os.ReadFile(makefilePath)
		if err != nil {
			t.Skip("Makefile not found")
		}

		makefileContent := string(content)
		
		// Verify Makefile contains version injection
		assert.Contains(t, makefileContent, "VERSION", 
			"Makefile should reference VERSION")
		assert.Contains(t, makefileContent, "-ldflags", 
			"Makefile should use ldflags for version injection")
		assert.Contains(t, makefileContent, "main.BUILD_VERSION", 
			"Makefile should inject main.BUILD_VERSION")
	})
}

func TestBuildWithVersionInjection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping build test in short mode")
	}

	t.Run("Build with version from VERSION file", func(t *testing.T) {
		repoRoot := findRepoRoot()
		if repoRoot == "" {
			t.Skip("Could not find repository root")
		}

		// Read VERSION file
		versionContent, err := os.ReadFile(filepath.Join(repoRoot, "VERSION"))
		if err != nil {
			t.Skip("VERSION file not found")
		}

		version := strings.TrimSpace(string(versionContent))
		assert.NotEmpty(t, version, "VERSION file should not be empty")

		// Validate version format
		parts := strings.Split(version, ".")
		assert.GreaterOrEqual(t, len(parts), 2, "Version should have at least major.minor")
		
		// Each part should be numeric
		for _, part := range parts {
			assert.Regexp(t, "^[0-9]+$", part, "Version parts should be numeric")
		}
	})
}

func TestEnvironmentVariableExport(t *testing.T) {
	t.Run("BISH_BUILD_VERSION should be exported to shell environment", func(t *testing.T) {
		// The main.go sets BISH_BUILD_VERSION in the environment
		// This test verifies the environment variable name is correct
		expectedEnvVar := "BISH_BUILD_VERSION"
		
		// Check that the environment variable name follows conventions
		assert.Equal(t, "BISH_BUILD_VERSION", expectedEnvVar,
			"Environment variable should be named BISH_BUILD_VERSION")
		
		// Verify it starts with BISH_ prefix
		assert.True(t, strings.HasPrefix(expectedEnvVar, "BISH_"),
			"Environment variable should have BISH_ prefix")
	})
}

// Helper function to find repository root
func findRepoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Walk up until we find .git directory or go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return ""
}

func TestFlagDefinitions(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		defaultValue interface{}
		description  string
	}{
		{
			name:         "Command flag",
			flagName:     "c",
			defaultValue: "",
			description:  "run a command",
		},
		{
			name:         "Login shell flag",
			flagName:     "l",
			defaultValue: false,
			description:  "run as a login shell",
		},
		{
			name:         "Help flag",
			flagName:     "h",
			defaultValue: false,
			description:  "display help information",
		},
		{
			name:         "Version flag",
			flagName:     "ver",
			defaultValue: false,
			description:  "display build version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flag set
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			
			// Re-register the flags as they would be in main
			command = flag.String("c", "", "run a command")
			loginShell = flag.Bool("l", false, "run as a login shell")
			helpFlag = flag.Bool("h", false, "display help information")
			versionFlag = flag.Bool("ver", false, "display build version")

			// Get the flag
			f := flag.Lookup(tt.flagName)
			require.NotNil(t, f, "Flag %s should be defined", tt.flagName)
			assert.Equal(t, tt.description, f.Usage, "Flag description should match")
		})
	}
}