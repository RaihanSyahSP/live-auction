package refresh_token_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"live-auction/server/internal/exception"
	"live-auction/server/internal/helper"
	"live-auction/server/internal/model/domain"
	refreshrepo "live-auction/server/internal/repository/refresh_token"
)

func newRefreshRepo(t *testing.T) (*refreshrepo.RefreshTokenRepositoryImpl, pgx.Tx) {
	t.Helper()

	pool := helper.OpenTestDB(t)
	tx := helper.BeginTestTx(t, pool)
	return refreshrepo.NewRefreshTokenRepositoryImpl(), tx
}

func createRefreshUser(t *testing.T, tx pgx.Tx) uuid.UUID {
	t.Helper()

	var id uuid.UUID
	err := tx.QueryRow(
		context.Background(),
		`insert into users (email, password_hash, display_name, role)
		 values ($1, $2, $3, 'user')
		 returning id`,
		uuid.NewString()+"@example.com",
		"hash",
		"Refresh User",
	).Scan(&id)
	if err != nil {
		t.Fatalf("create user fixture: %v", err)
	}
	return id
}

func hashToken(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}

func createRefreshToken(
	t *testing.T,
	repo *refreshrepo.RefreshTokenRepositoryImpl,
	tx pgx.Tx,
	userID uuid.UUID,
	familyID uuid.UUID,
	raw string,
	expiresAt time.Time,
) domain.RefreshToken {
	t.Helper()

	created, err := repo.Create(context.Background(), tx, domain.RefreshToken{
		UserID:    userID,
		FamilyID:  familyID,
		TokenHash: hashToken(raw),
		ExpiresAt: expiresAt,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	return created
}

func TestRefreshTokenRepositoryCreateAndFind(t *testing.T) {
	repo, tx := newRefreshRepo(t)
	ctx := context.Background()
	userID := createRefreshUser(t, tx)
	familyID := uuid.New()
	raw := uuid.NewString()
	expiresAt := time.Now().Add(time.Hour).UTC()

	created := createRefreshToken(t, repo, tx, userID, familyID, raw, expiresAt)
	if created.ID == uuid.Nil {
		t.Error("Create() returned nil ID")
	}
	if created.UserID != userID {
		t.Errorf("UserID = %v, want %v", created.UserID, userID)
	}
	if created.FamilyID != familyID {
		t.Errorf("FamilyID = %v, want %v", created.FamilyID, familyID)
	}
	if !bytes.Equal(created.TokenHash, hashToken(raw)) {
		t.Error("TokenHash does not match")
	}
	if created.CreatedAt.IsZero() {
		t.Error("Create() returned zero CreatedAt")
	}

	found, err := repo.FindByHashForUpdate(ctx, tx, hashToken(raw))
	if err != nil {
		t.Fatalf("FindByHashForUpdate() error = %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("FindByHashForUpdate() ID = %v, want %v", found.ID, created.ID)
	}
	if found.RotatedAt != nil {
		t.Errorf("RotatedAt = %v, want nil", found.RotatedAt)
	}
	if found.ReplacedByID != nil {
		t.Errorf("ReplacedByID = %v, want nil", found.ReplacedByID)
	}
	if found.RevokedAt != nil {
		t.Errorf("RevokedAt = %v, want nil", found.RevokedAt)
	}
}

func TestRefreshTokenRepositoryRotate(t *testing.T) {
	repo, tx := newRefreshRepo(t)
	ctx := context.Background()
	userID := createRefreshUser(t, tx)
	familyID := uuid.New()
	oldRaw := uuid.NewString()
	newRaw := uuid.NewString()

	oldToken := createRefreshToken(t, repo, tx, userID, familyID, oldRaw, time.Now().Add(time.Hour))
	newToken := createRefreshToken(t, repo, tx, userID, familyID, newRaw, time.Now().Add(time.Hour))
	rotatedAt := time.Now().UTC()

	rotated, err := repo.Rotate(ctx, tx, oldToken.ID, newToken.ID, rotatedAt)
	if err != nil {
		t.Fatalf("Rotate() error = %v", err)
	}
	if rotated.RotatedAt == nil {
		t.Error("Rotate() returned nil RotatedAt")
	}
	if rotated.ReplacedByID == nil || *rotated.ReplacedByID != newToken.ID {
		t.Errorf("ReplacedByID = %v, want %v", rotated.ReplacedByID, newToken.ID)
	}

	foundOld, err := repo.FindByHashForUpdate(ctx, tx, hashToken(oldRaw))
	if err != nil {
		t.Fatalf("FindByHashForUpdate(old) error = %v", err)
	}
	if foundOld.RotatedAt == nil {
		t.Error("old token RotatedAt = nil, want value")
	}
	if foundOld.ReplacedByID == nil || *foundOld.ReplacedByID != newToken.ID {
		t.Errorf("old token ReplacedByID = %v, want %v", foundOld.ReplacedByID, newToken.ID)
	}

	foundNew, err := repo.FindByHashForUpdate(ctx, tx, hashToken(newRaw))
	if err != nil {
		t.Fatalf("FindByHashForUpdate(new) error = %v", err)
	}
	if foundNew.RotatedAt != nil {
		t.Errorf("new token RotatedAt = %v, want nil", foundNew.RotatedAt)
	}
}

func TestRefreshTokenRepositoryRevokeFamily(t *testing.T) {
	repo, tx := newRefreshRepo(t)
	ctx := context.Background()
	userID := createRefreshUser(t, tx)
	familyID := uuid.New()
	otherFamilyID := uuid.New()
	firstRaw := uuid.NewString()
	secondRaw := uuid.NewString()
	otherRaw := uuid.NewString()

	createRefreshToken(t, repo, tx, userID, familyID, firstRaw, time.Now().Add(time.Hour))
	createRefreshToken(t, repo, tx, userID, familyID, secondRaw, time.Now().Add(time.Hour))
	createRefreshToken(t, repo, tx, userID, otherFamilyID, otherRaw, time.Now().Add(time.Hour))
	revokedAt := time.Now().UTC()

	if err := repo.RevokeFamily(ctx, tx, familyID, revokedAt); err != nil {
		t.Fatalf("RevokeFamily() error = %v", err)
	}

	for _, raw := range []string{firstRaw, secondRaw} {
		found, err := repo.FindByHashForUpdate(ctx, tx, hashToken(raw))
		if err != nil {
			t.Fatalf("FindByHashForUpdate() error = %v", err)
		}
		if found.RevokedAt == nil {
			t.Errorf("token %s RevokedAt = nil, want value", raw)
		}
	}

	other, err := repo.FindByHashForUpdate(ctx, tx, hashToken(otherRaw))
	if err != nil {
		t.Fatalf("FindByHashForUpdate(other) error = %v", err)
	}
	if other.RevokedAt != nil {
		t.Errorf("other family RevokedAt = %v, want nil", other.RevokedAt)
	}
}

func TestRefreshTokenRepositoryDeleteExpired(t *testing.T) {
	repo, tx := newRefreshRepo(t)
	ctx := context.Background()
	userID := createRefreshUser(t, tx)
	expiredRaw := uuid.NewString()
	activeRaw := uuid.NewString()

	createRefreshToken(t, repo, tx, userID, uuid.New(), expiredRaw, time.Now().Add(-time.Hour))
	createRefreshToken(t, repo, tx, userID, uuid.New(), activeRaw, time.Now().Add(time.Hour))

	if err := repo.DeleteExpired(ctx, tx, time.Now().UTC()); err != nil {
		t.Fatalf("DeleteExpired() error = %v", err)
	}

	_, err := repo.FindByHashForUpdate(ctx, tx, hashToken(expiredRaw))
	if !errors.Is(err, exception.ErrRefreshTokenNotFound) {
		t.Fatalf("expired token error = %v, want %v", err, exception.ErrRefreshTokenNotFound)
	}

	if _, err := repo.FindByHashForUpdate(ctx, tx, hashToken(activeRaw)); err != nil {
		t.Fatalf("active token error = %v, want nil", err)
	}
}

func TestRefreshTokenRepositoryFindByHashNotFound(t *testing.T) {
	repo, tx := newRefreshRepo(t)

	_, err := repo.FindByHashForUpdate(context.Background(), tx, hashToken(uuid.NewString()))
	if !errors.Is(err, exception.ErrRefreshTokenNotFound) {
		t.Fatalf("FindByHashForUpdate() error = %v, want %v", err, exception.ErrRefreshTokenNotFound)
	}
}
