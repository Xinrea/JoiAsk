package database

import (
	"bytes"
	"errors"
	"joiask-backend/internal/adminpassword"
	"joiask-backend/pkg/util"
	"log"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func bootstrapTestDB(t *testing.T) (*gorm.DB, *bytes.Buffer) {
	t.Helper()
	var logs bytes.Buffer
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "bootstrap.db")), &gorm.Config{
		Logger: logger.New(log.New(&logs, "", 0), logger.Config{LogLevel: logger.Info, SlowThreshold: time.Second}),
	})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	if err := db.AutoMigrate(&Admin{}); err != nil {
		t.Fatal(err)
	}
	logs.Reset()
	return db, &logs
}

func TestExplicitAdminProvisioningAndRecovery(t *testing.T) {
	db, logs := bootstrapTestDB(t)
	password := "  " + strings.Repeat("long-password", 8) + "  "
	if err := SetAdminPassword(db, "operator", password, false); err != nil {
		t.Fatal(err)
	}
	if logs.Len() != 0 {
		t.Fatal("credential provisioning must not log SQL")
	}
	var original Admin
	if err := db.Where("username = ?", "operator").First(&original).Error; err != nil {
		t.Fatal(err)
	}
	if ok, legacy := adminpassword.Verify(original.Password, password); !ok || legacy {
		t.Fatal("new credentials must use adminpassword.Hash without trimming")
	}
	if original.Role != AdminRoleSuperAdmin {
		t.Fatal("explicit CLI create must provision a super administrator")
	}
	if err := SetAdminPassword(db, "operator", "different-password", false); !errors.Is(err, ErrAdminExists) {
		t.Fatalf("duplicate create returned %v", err)
	}
	if err := SetAdminPassword(db, "missing", "different-password", true); !errors.Is(err, ErrAdminMissing) {
		t.Fatalf("missing reset returned %v", err)
	}
	var preserved Admin
	if err := db.First(&preserved, original.ID).Error; err != nil || preserved.Password != original.Password || preserved.SessionVersion != 0 {
		t.Fatal("failed commands changed existing credentials")
	}
	if err := db.Model(&original).UpdateColumn("role", AdminRoleAdmin).Error; err != nil {
		t.Fatal(err)
	}
	for version := 1; version <= 2; version++ {
		logs.Reset()
		if err := SetAdminPassword(db, "operator", "new-password", true); err != nil {
			t.Fatal(err)
		}
		if logs.Len() != 0 {
			t.Fatal("password reset must not log SQL")
		}
		var updated Admin
		if err := db.First(&updated, original.ID).Error; err != nil {
			t.Fatal(err)
		}
		if ok, _ := adminpassword.Verify(updated.Password, "new-password"); !ok || updated.SessionVersion != version {
			t.Fatal("reset must replace the password and increment the session version, even for the same password")
		}
		if updated.Role != AdminRoleAdmin {
			t.Fatal("password recovery must not promote an existing administrator")
		}
	}
	var count int64
	if err := db.Model(&Admin{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatal("reset of missing account must not create an administrator")
	}
}

func TestAdminCredentialFailuresDoNotExposeSQLOrSecrets(t *testing.T) {
	db, logs := bootstrapTestDB(t)
	if err := SetAdminPassword(db, "operator", "old-password", false); err != nil {
		t.Fatal(err)
	}
	for name, operation := range map[string]string{"insert": "INSERT", "update": "UPDATE OF password"} {
		if err := db.Exec("CREATE TRIGGER reject_" + name + " BEFORE " + operation +
			" ON admins BEGIN SELECT RAISE(ABORT, 'private-database-error'); END").Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, reset := range []bool{false, true} {
		logs.Reset()
		username := "sensitive-username"
		if reset {
			username = "operator"
		}
		err := SetAdminPassword(db, username, "sensitive-password", reset)
		if !errors.Is(err, errAdminDatabase) {
			t.Fatalf("expected sanitized database error, got %v", err)
		}
		if logs.Len() != 0 || strings.Contains(err.Error(), "private-database-error") {
			t.Fatal("database failure exposed SQL or driver error")
		}
	}
	var stored Admin
	if err := db.First(&stored).Error; err != nil || stored.SessionVersion != 0 {
		t.Fatal("failed reset changed session version")
	}
	if ok, _ := adminpassword.Verify(stored.Password, "old-password"); !ok {
		t.Fatal("failed reset changed password")
	}
}

func TestAdminProvisioningRejectsEmptyCredentials(t *testing.T) {
	db, _ := bootstrapTestDB(t)
	for _, input := range [][2]string{{"", "password"}, {" \t ", "password"}, {"operator", ""}} {
		if err := SetAdminPassword(db, input[0], input[1], false); err == nil {
			t.Fatal("empty credential accepted")
		}
	}
	var count int64
	if err := db.Model(&Admin{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatal("invalid input created an administrator")
	}
}

func TestStartupNeverSeedsAdministrators(t *testing.T) {
	db, _ := bootstrapTestDB(t)
	previous := DB
	DB = db
	t.Cleanup(func() { DB = previous })
	initializeDB()
	var count int64
	if err := db.Model(&Admin{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatal("empty database must not receive default credentials")
	}
	hashed, err := adminpassword.Hash("existing-password")
	if err != nil {
		t.Fatal(err)
	}
	accounts := []Admin{
		{Username: "admin", Password: util.Md5v("legacy-password"), SessionVersion: 3},
		{Username: "another-operator", Password: hashed, SessionVersion: 8},
	}
	for i := range accounts {
		if err := db.Create(&accounts[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Model(&accounts[0]).Update("username", "renamed-operator").Error; err != nil {
		t.Fatal(err)
	}
	initializeDB()
	initializeDB()
	for _, expected := range accounts {
		var actual Admin
		if err := db.First(&actual, expected.ID).Error; err != nil {
			t.Fatal(err)
		}
		if actual.Username != expected.Username || actual.Password != expected.Password || actual.SessionVersion != expected.SessionVersion {
			t.Fatal("startup changed existing administrator credentials")
		}
	}
	if err := db.Model(&Admin{}).Count(&count).Error; err != nil || count != 2 {
		t.Fatal("startup recreated renamed administrator")
	}
	if err := db.Where("1 = 1").Delete(&Admin{}).Error; err != nil {
		t.Fatal(err)
	}
	initializeDB()
	if err := db.Model(&Admin{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatal("startup recreated deleted administrators")
	}
}

func TestLegacyAdminRoleMigrationRunsOncePerAccount(t *testing.T) {
	db, _ := bootstrapTestDB(t)
	accounts := []Admin{
		{Username: "admin", Password: "legacy-secret", SessionVersion: 4},
		{Username: "Admin", Password: "another-secret", SessionVersion: 5},
		{Username: "operator", Password: "third-secret", SessionVersion: 6},
		{Username: "existing-super", Password: "fourth-secret", SessionVersion: 7, Role: AdminRoleSuperAdmin},
	}
	if err := db.Create(&accounts).Error; err != nil {
		t.Fatal(err)
	}
	if err := initializeAdminRoles(db); err != nil {
		t.Fatal(err)
	}
	for i, expected := range accounts {
		var actual Admin
		if err := db.First(&actual, expected.ID).Error; err != nil {
			t.Fatal(err)
		}
		role := AdminRoleAdmin
		if i == 0 || i == 3 {
			role = AdminRoleSuperAdmin
		}
		if actual.Role != role || actual.Password != expected.Password || actual.SessionVersion != expected.SessionVersion {
			t.Fatal("migration assigned incorrect role or modified credentials/version")
		}
	}
	// A super administrator remains privileged after rename; a regular account
	// taking the historic username never gains privileges on later startups.
	if err := db.Model(&Admin{}).Where("id = ?", accounts[0].ID).Update("username", "renamed-super").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&Admin{}).Where("id = ?", accounts[2].ID).Update("username", "admin").Error; err != nil {
		t.Fatal(err)
	}
	if err := initializeAdminRoles(db); err != nil {
		t.Fatal(err)
	}
	var renamed, regular Admin
	if err := db.First(&renamed, accounts[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&regular, accounts[2].ID).Error; err != nil {
		t.Fatal(err)
	}
	if renamed.Role != AdminRoleSuperAdmin || regular.Role != AdminRoleAdmin {
		t.Fatal("username affected an already migrated role")
	}
}

func TestPromoteRecoversDeploymentWithoutLegacyAdmin(t *testing.T) {
	db, logs := bootstrapTestDB(t)
	account := Admin{Username: "owner", Password: "existing-hash", SessionVersion: 3}
	if err := db.Create(&account).Error; err != nil {
		t.Fatal(err)
	}
	if err := initializeAdminRoles(db); err != nil {
		t.Fatal(err)
	}
	var migrated Admin
	if err := db.First(&migrated, account.ID).Error; err != nil || migrated.Role != AdminRoleAdmin {
		t.Fatal("migration must not automatically choose a replacement super administrator")
	}
	logs.Reset()
	if err := PromoteAdmin(db, "owner"); err != nil {
		t.Fatal(err)
	}
	if err := PromoteAdmin(db, "owner"); err != nil {
		t.Fatal(err)
	}
	if err := PromoteAdmin(db, "missing"); !errors.Is(err, ErrAdminMissing) {
		t.Fatalf("missing promotion returned %v", err)
	}
	if logs.Len() != 0 {
		t.Fatal("promotion must not log SQL")
	}
	var promoted Admin
	if err := db.First(&promoted, account.ID).Error; err != nil {
		t.Fatal(err)
	}
	if promoted.Role != AdminRoleSuperAdmin || promoted.Password != account.Password || promoted.SessionVersion != 4 {
		t.Fatal("explicit promotion must preserve password and revoke old sessions exactly once")
	}
}

func TestAdminManagementLockFailureSkipsCallback(t *testing.T) {
	db, _ := bootstrapTestDB(t)
	if err := db.Migrator().DropTable(&Admin{}); err != nil {
		t.Fatal(err)
	}
	called := false
	err := WithAdminManagementLock(db, func(*gorm.DB) error {
		called = true
		return nil
	})
	if err == nil || called {
		t.Fatal("failed management lock must fail closed")
	}
}

func TestAdminManagementLockDoesNotMutateAccountsOrRunUpdateHooks(t *testing.T) {
	db, _ := bootstrapTestDB(t)
	account := Admin{Username: "owner", Password: "existing-hash", SessionVersion: 3, Role: AdminRoleAdmin}
	if err := db.Create(&account).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Update().Before("gorm:update").Register("reject_updates", func(tx *gorm.DB) {
		tx.AddError(errors.New("credential update hook"))
	}); err != nil {
		t.Fatal(err)
	}
	called := false
	if err := WithAdminManagementLock(db, func(*gorm.DB) error {
		called = true
		return nil
	}); err != nil || !called {
		t.Fatal("lock ran a model update callback or skipped its callback")
	}
	var stored Admin
	if err := db.First(&stored, account.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Password != account.Password || stored.SessionVersion != account.SessionVersion ||
		stored.Role != account.Role || !stored.UpdatedAt.Equal(account.UpdatedAt) {
		t.Fatal("management lock changed account data")
	}
}
