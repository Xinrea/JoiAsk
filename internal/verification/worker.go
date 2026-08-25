package verification

import (
	"context"
	"joiask-backend/internal/bilibili"
	"joiask-backend/internal/database"
	"joiask-backend/internal/secret"
	"time"

	log "github.com/sirupsen/logrus"
)

type Worker struct {
	client *bilibili.Client
	now    func() time.Time
}

func NewWorker(client *bilibili.Client) *Worker {
	return &Worker{client: client, now: time.Now}
}

func (w *Worker) Start(ctx context.Context) {
	go func() {
		w.RunOnce(ctx)
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.RunOnce(ctx)
			}
		}
	}()
}

func (w *Worker) RunOnce(ctx context.Context) {
	var account database.BilibiliVerificationAccount
	if err := database.DB.First(&account).Error; err != nil {
		return
	}
	now := w.now().UTC()
	cookieValue, err := secret.DecryptCookie(account.EncryptedCookie)
	if err != nil {
		w.recordCheck(&account, now, err, false)
		return
	}
	followers, err := w.client.Followers(ctx, account.BilibiliUID, cookieValue, 50)
	if err != nil {
		w.recordCheck(&account, now, err, false)
		return
	}
	w.recordCheck(&account, now, nil, true)

	var requests []database.BilibiliVerificationRequest
	if err := database.DB.Where("status = ?", database.VerificationPending).Find(&requests).Error; err != nil {
		log.Errorf("failed to load Bilibili verification requests: %v", err)
		return
	}
	followerByUID := make(map[int64]bilibili.Follower, len(followers))
	for _, follower := range followers {
		followerByUID[follower.MID] = follower
	}
	for _, request := range requests {
		follower, ok := followerByUID[request.BilibiliUID]
		if !ok || follower.MTime < request.RequestedAt.Unix() || follower.MTime > request.ExpiresAt.Unix() {
			continue
		}
		followedAt := time.Unix(follower.MTime, 0).UTC()
		confirmationUntil := now.Add(10 * time.Minute)
		updates := map[string]any{
			"status":             database.VerificationVerified,
			"followed_at":        followedAt,
			"bilibili_name":      follower.Name,
			"bilibili_avatar":    follower.Face,
			"verified_at":        now,
			"confirmation_until": confirmationUntil,
		}
		database.DB.Model(&database.BilibiliVerificationRequest{}).
			Where("id = ? AND status = ?", request.ID, database.VerificationPending).
			Updates(updates)
	}
	database.DB.Model(&database.BilibiliVerificationRequest{}).
		Where("status = ? AND expires_at < ?", database.VerificationPending, now).
		Update("status", database.VerificationExpired)
}

func (w *Worker) recordCheck(account *database.BilibiliVerificationAccount, now time.Time, err error, success bool) {
	updates := map[string]any{"last_checked_at": now}
	if success {
		updates["last_successful_at"] = now
		updates["last_error"] = ""
	} else if err != nil {
		updates["last_error"] = err.Error()
		log.Warnf("Bilibili verification check failed: %v", err)
	}
	database.DB.Model(account).Updates(updates)
}
