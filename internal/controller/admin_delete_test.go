package controller

import (
	"errors"
	"joiask-backend/internal/database"
	"testing"

	"gorm.io/gorm"
)

func TestAdminDeletionRechecksActorAfterAuthentication(t *testing.T) {
	for _, mutation := range []string{"deleted", "password reset"} {
		t.Run(mutation, func(t *testing.T) {
			setupRecordIDTestDB(t)
			var actor database.Admin
			if err := database.DB.First(&actor, 1).Error; err != nil {
				t.Fatal(err)
			}
			if mutation == "deleted" {
				if err := database.DB.Delete(&actor).Error; err != nil {
					t.Fatal(err)
				}
			} else if err := database.DB.Model(&database.Admin{}).Where("id = ?", actor.ID).UpdateColumn("session_version", 1).Error; err != nil {
				t.Fatal(err)
			}
			if err := deleteAdminAccount(actor, 2); !errors.Is(err, errAdminMutationSessionExpired) {
				t.Fatalf("stale actor was accepted: %v", err)
			}
			var target database.Admin
			if err := database.DB.First(&target, 2).Error; err != nil {
				t.Fatalf("stale actor deleted the remaining account: %v", err)
			}
		})
	}
}

func TestConcurrentAdministratorsCannotDeleteEachOther(t *testing.T) {
	setupRecordIDTestDB(t)
	if err := database.DB.Model(&database.Admin{}).Where("id = ?", 2).UpdateColumn("role", database.AdminRoleSuperAdmin).Error; err != nil {
		t.Fatal(err)
	}
	var admins []database.Admin
	if err := database.DB.Order("id").Find(&admins).Error; err != nil || len(admins) != 2 {
		t.Fatalf("load administrators: %v", err)
	}
	// Both actors represent requests that have already passed authentication.
	start := make(chan struct{})
	results := make(chan error, 2)
	for i, actor := range admins {
		target := admins[1-i].ID
		go func(actor database.Admin, target uint) {
			<-start
			results <- deleteAdminAccount(actor, target)
		}(actor, target)
	}
	close(start)
	successes := 0
	for range admins {
		if err := <-results; err == nil {
			successes++
		}
	}
	var count int64
	if err := database.DB.Model(&database.Admin{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if successes != 1 || count != 1 {
		t.Fatalf("mutual deletion must leave one account: successes=%d remaining=%d", successes, count)
	}
}

func TestAdminDeletionDoesNotModifyActor(t *testing.T) {
	setupRecordIDTestDB(t)
	var before database.Admin
	if err := database.DB.First(&before, 1).Error; err != nil {
		t.Fatal(err)
	}
	if err := deleteAdminAccount(before, 2); err != nil {
		t.Fatal(err)
	}
	if err := deleteAdminAccount(before, 2); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("missing target: %v", err)
	}
	var after database.Admin
	if err := database.DB.First(&after, before.ID).Error; err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatalf("locking the actor changed account data: before=%+v after=%+v", before, after)
	}
}
