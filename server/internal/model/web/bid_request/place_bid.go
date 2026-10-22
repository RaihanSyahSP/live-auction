package bid_request

import "github.com/google/uuid"

type PlaceBidRequest struct {
	Amount         int64     `json:"amount" validate:"required,gt=0"`
	IdempotencyKey uuid.UUID `json:"idempotency_key" validate:"required"`
}
