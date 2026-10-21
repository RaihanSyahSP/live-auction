package user_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"live-auction/server/internal/exception"
	"live-auction/server/internal/helper"
	"live-auction/server/internal/model/domain"
	userrepo "live-auction/server/internal/repository/user"
)

func newUserRepo(t *testing.T) (*userrepo.UserRepositoryImpl, pgx.Tx) {
	t.Helper()

	pool := helper.OpenTestDB(t)
	tx := helper.BeginTestTx(t, pool)
	return userrepo.NewUserRepositoryImpl(), tx
}

func uniqueEmail() string {
	return uuid.NewString() + "@example.com"
}

func TestUserRepositoryCreateAndFindByEmail(t *testing.T) {
	repo, tx := newUserRepo(t)
	ctx := context.Background()
	email := uniqueEmail()

	created, err := repo.Create(ctx, tx, domain.User{
		Email:        email,
		PasswordHash: "hash",
		DisplayName:  "Raihan",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == uuid.Nil {
		t.Error("Create() returned nil ID")
	}
	if created.CreatedAt.IsZero() {
		t.Error("Create() returned zero CreatedAt")
	}

	found, err := repo.FindByEmail(ctx, tx, strings.ToUpper(email))
	if err != nil {
		t.Fatalf("FindByEmail() error = %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("FindByEmail() ID = %v, want %v", found.ID, created.ID)
	}
	if found.DisplayName != created.DisplayName {
		t.Errorf("FindByEmail() DisplayName = %q, want %q", found.DisplayName, created.DisplayName)
	}
}

func TestUserRepositoryFindByID(t *testing.T) {
	repo, tx := newUserRepo(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, tx, domain.User{
		Email:        uniqueEmail(),
		PasswordHash: "hash",
		DisplayName:  "Dewi",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	found, err := repo.FindByID(ctx, tx, created.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("FindByID() ID = %v, want %v", found.ID, created.ID)
	}
}

func TestUserRepositoryCreatesUserRoleByDefault(t *testing.T) {
	repo, tx := newUserRepo(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, tx, domain.User{
		Email:        uniqueEmail(),
		PasswordHash: "hash",
		DisplayName:  "Default Role",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Role != domain.RoleUser {
		t.Errorf("Create() Role = %q, want %q", created.Role, domain.RoleUser)
	}
}

func TestUserRepositoryCreatesAdminRole(t *testing.T) {
	repo, tx := newUserRepo(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, tx, domain.User{
		Email:        uniqueEmail(),
		PasswordHash: "hash",
		DisplayName:  "Admin",
		Role:         domain.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Role != domain.RoleAdmin {
		t.Errorf("Create() Role = %q, want %q", created.Role, domain.RoleAdmin)
	}
}

func TestUserRepositoryRejectsDuplicateEmail(t *testing.T) {
	repo, tx := newUserRepo(t)
	ctx := context.Background()
	email := uniqueEmail()

	if _, err := repo.Create(ctx, tx, domain.User{
		Email:        email,
		PasswordHash: "hash",
		DisplayName:  "First",
	}); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	_, err := repo.Create(ctx, tx, domain.User{
		Email:        strings.ToUpper(email),
		PasswordHash: "hash",
		DisplayName:  "Second",
	})
	if !errors.Is(err, exception.ErrEmailTaken) {
		t.Fatalf("second Create() error = %v, want %v", err, exception.ErrEmailTaken)
	}
}

func TestUserRepositoryFindByIDNotFound(t *testing.T) {
	repo, tx := newUserRepo(t)

	_, err := repo.FindByID(context.Background(), tx, uuid.New())
	if !errors.Is(err, exception.ErrUserNotFound) {
		t.Fatalf("FindByID() error = %v, want %v", err, exception.ErrUserNotFound)
	}
}

func TestUserRepositoryFindByEmailNotFound(t *testing.T) {
	repo, tx := newUserRepo(t)

	_, err := repo.FindByEmail(context.Background(), tx, uniqueEmail())
	if !errors.Is(err, exception.ErrUserNotFound) {
		t.Fatalf("FindByEmail() error = %v, want %v", err, exception.ErrUserNotFound)
	}
}
