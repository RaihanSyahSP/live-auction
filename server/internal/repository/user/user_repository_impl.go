package user

import (
	"context"
	"errors"
	"fmt"
	"live-auction/server/internal/exception"
	"live-auction/server/internal/model/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type UserRepositoryImpl struct{}

func NewUserRepositoryImpl() *UserRepositoryImpl {
	return &UserRepositoryImpl{}
}

func (u *UserRepositoryImpl) Create(ctx context.Context, tx pgx.Tx, user domain.User) (domain.User, error) {
	role := user.Role
	if role == "" {
		role = domain.RoleUser
	}

	const query = `
		insert into users (email, password_hash, display_name, role)
		values ($1, $2, $3, $4)
		returning id, email, password_hash, display_name, role, created_at
	`

	created, err := scanUser(tx.QueryRow(
		ctx,
		query,
		user.Email,
		user.PasswordHash,
		user.DisplayName,
		role,
	))
	if err != nil {
		if isUniqueViolation(err, "users_email_lower_idx") {
			return domain.User{}, exception.ErrEmailTaken
		}
		return domain.User{}, fmt.Errorf("create user: %w", err)
	}

	return created, nil
}

func (u *UserRepositoryImpl) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (domain.User, error) {
	const query = `
			select id, email, password_hash, display_name, role, created_at
			from users
			where id = $1
		`

	user, err := scanUser(tx.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, exception.ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("find user by id: %w", err)
	}

	return user, nil
}

func (u *UserRepositoryImpl) FindByEmail(ctx context.Context, tx pgx.Tx, email string) (domain.User, error) {
	const query = `
			select id, email, password_hash, display_name, role, created_at
			from users
			where lower(email) = lower($1)
		`

	user, err := scanUser(tx.QueryRow(ctx, query, email))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, exception.ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("find user by email: %w", err)
	}

	return user, nil
}

func scanUser(row pgx.Row) (domain.User, error) {
	var user domain.User
	var role string

	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&role,
		&user.CreatedAt,
	)
	if err != nil {
		return domain.User{}, err
	}

	user.Role = domain.Role(role)
	return user, nil
}

func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == "23505" &&
		pgErr.ConstraintName == constraint
}
