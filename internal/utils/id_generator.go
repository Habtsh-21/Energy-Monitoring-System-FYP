package utils

import (

	"github.com/google/uuid"
)

// UUID generator
func IdGenerator() uuid.UUID {
	uuid := uuid.New()
	return uuid
}
