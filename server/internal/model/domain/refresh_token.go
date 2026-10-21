package domain

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	FamilyID     uuid.UUID
	TokenHash    []byte
	ExpiresAt    time.Time
	CreatedAt    time.Time
	RotatedAt    *time.Time
	ReplacedByID *uuid.UUID
	RevokedAt    *time.Time
}
