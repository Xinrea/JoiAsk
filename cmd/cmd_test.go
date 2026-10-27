package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestStartupRejectsUnsafeSessionSecretBeforeDatabaseInit(t *testing.T) {
	if os.Getenv("JOIASK_TEST_STARTUP") == "1" {
		main()
		return
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for name, security := range map[string]map[string]string{
		"missing":     nil,
		"short":       {"session_secret": "short"},
		"legacy":      {"session_secret": "WhyJoiIsSoCute"},
		"sample":      {"session_secret": "replace-with-a-long-random-secret"},
		"development": {"session_secret": "dev-only-session-secret-change-in-production"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			databasePath := filepath.Join(dir, "ask.db")
			configPath := filepath.Join(dir, "config.json")
			config := map[string]interface{}{
				"db_type": "sqlite",
				"sqlite":  databasePath,
			}
			if security != nil {
				config["security"] = security
			}
			data, err := json.Marshal(config)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(configPath, data, 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("JOIASK_CONFIG", configPath)
			t.Setenv("JOIASK_TEST_STARTUP", "1")
			command := exec.Command(executable, "-test.run=^TestStartupRejectsUnsafeSessionSecretBeforeDatabaseInit$")
			output, err := command.CombinedOutput()
			if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
				t.Fatalf("expected fatal exit 1; error=%v output=%s", err, output)
			}
			if !strings.Contains(string(output), "security.session_secret") || !strings.Contains(string(output), "openssl rand -hex 32") {
				t.Fatalf("missing configuration recovery instructions: %s", output)
			}
			if strings.Contains(string(output), "Using sqlite") || strings.Contains(string(output), "Starting server") {
				t.Fatalf("startup continued after invalid configuration: %s", output)
			}
			if _, err := os.Stat(databasePath); !os.IsNotExist(err) {
				t.Fatalf("database must not be created before secret validation: %v", err)
			}
		})
	}
}
