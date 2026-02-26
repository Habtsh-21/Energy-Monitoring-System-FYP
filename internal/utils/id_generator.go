package utils

import (
	"math/rand"
	"time"
)

// IdGenerator returns a 10 digit random number as a uint
func IdGenerator() uint {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return uint(r.Int63n(10000000000))
}
