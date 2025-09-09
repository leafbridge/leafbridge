package lbrand

import "crypto/rand"

// NewID returns a random identifier of type T.
func NewID[T ~string]() T {
	return T(rand.Text())
}
