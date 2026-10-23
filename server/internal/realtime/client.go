package realtime

import "github.com/google/uuid"

type Client struct {
	UserID *uuid.UUID
	Send   chan []byte
}
