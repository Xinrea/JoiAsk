package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"joiask-backend/internal/adminpassword"
	"joiask-backend/internal/database"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestStartupRejectsUnsafeSessionSecretBeforeDatabaseInit(t *testing.T) {
	if os.Getenv("JOIASK_TEST_STARTUP") == "1" {
		os.Args = os.Args[:1]
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

func TestReadPassword(t *testing.T) {
	for _, input := range []string{"password", "  password  \n", "password\r\n", strings.Repeat("x", 4096) + "\r\n"} {
		got, err := readPassword(strings.NewReader(input))
		expected := strings.TrimSuffix(strings.TrimSuffix(input, "\n"), "\r")
		if err != nil || got != expected {
			t.Fatalf("password bytes changed or valid input rejected: %v", err)
		}
	}
	for _, input := range []string{"", "\n", "\r\n", "one\ntwo\n", "one\rtwo", strings.Repeat("x", 4097)} {
		if _, err := readPassword(strings.NewReader(input)); err == nil {
			t.Fatal("invalid password input accepted")
		}
	}
	if _, err := readPassword(failedPasswordReader{}); err == nil || strings.Contains(err.Error(), "private") {
		t.Fatal("read error must be sanitized")
	}
}

type failedPasswordReader struct{}

func (failedPasswordReader) Read([]byte) (int, error) {
	return 0, errors.New("private input error")
}

func TestAdminCommand(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	dir := t.TempDir()
	databasePath := filepath.Join(dir, "admin.db")
	configPath := filepath.Join(dir, "config.json")
	// CLI recovery deliberately does not need a valid HTTP session secret.
	config, err := json.Marshal(map[string]string{"db_type": "sqlite", "sqlite": databasePath})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, config, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JOIASK_CONFIG", configPath)
	for _, args := range [][]string{
		{"admin"}, {"admin", "delete", "operator"}, {"admin", "create", ""},
		{"admin", "create", "operator", "password-must-not-be-an-argument"}, {"unknown"},
	} {
		if err := run(args, strings.NewReader("password"), io.Discard); err == nil || strings.Contains(err.Error(), "password-must-not-be-an-argument") {
			t.Fatal("invalid command must fail without echoing arguments")
		}
	}
	if err := run([]string{"admin", "create", "operator"}, strings.NewReader(""), io.Discard); err == nil {
		t.Fatal("empty password accepted")
	}
	if _, err := os.Stat(databasePath); !os.IsNotExist(err) {
		t.Fatal("invalid command/password must not initialize the database")
	}
	var output bytes.Buffer
	if err := run([]string{"admin", "create", "operator"}, strings.NewReader("initial-password\n"), &output); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "initial-password") {
		t.Fatal("command printed password")
	}
	if err := run([]string{"admin", "create", "operator"}, strings.NewReader("replacement-password"), io.Discard); !errors.Is(err, database.ErrAdminExists) {
		t.Fatalf("duplicate create returned %v", err)
	}
	if err := run([]string{"admin", "reset-password", "missing"}, strings.NewReader("replacement-password"), io.Discard); !errors.Is(err, database.ErrAdminMissing) {
		t.Fatalf("missing reset returned %v", err)
	}
	if err := run([]string{"admin", "reset-password", "operator"}, strings.NewReader("replacement-password\r\n"), io.Discard); err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	var admin database.Admin
	if err := db.Where("username = ?", "operator").First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if ok, _ := adminpassword.Verify(admin.Password, "replacement-password"); !ok || admin.SessionVersion != 1 {
		t.Fatal("command failed to reset password and invalidate sessions")
	}
	if admin.Role != database.AdminRoleSuperAdmin {
		t.Fatal("CLI create must explicitly create a super administrator")
	}
	if err := db.Model(&admin).UpdateColumn("role", database.AdminRoleAdmin).Error; err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"admin", "promote", "operator"}, failedPasswordReader{}, io.Discard); err != nil {
		t.Fatal("promote must not read a password:", err)
	}
	if err := db.First(&admin, admin.ID).Error; err != nil || admin.Role != database.AdminRoleSuperAdmin || admin.SessionVersion != 2 {
		t.Fatal("explicit promote did not update role and revoke old sessions")
	}
	if err := run([]string{"admin", "promote", "missing"}, failedPasswordReader{}, io.Discard); !errors.Is(err, database.ErrAdminMissing) {
		t.Fatalf("promote missing returned %v", err)
	}
	if db.Migrator().HasTable(&database.Config{}) {
		t.Fatal("CLI must not initialize unrelated site tables")
	}
}
