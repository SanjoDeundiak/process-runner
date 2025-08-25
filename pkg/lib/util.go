package lib

import (
	"github.com/google/uuid"
)

// NewID generates a UUID version 4 string (RFC 4122)
func NewID() string {
	return uuid.NewString()
}
