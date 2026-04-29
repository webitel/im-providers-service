package model

import (
	"crypto/sha256"
	"fmt"
)

type ExternalUser struct {
	ID        string
	FirstName string
	LastName  string
}

// FullName returns the combined name for logging and upserting
func (u *ExternalUser) FullName() string {
	return fmt.Sprintf("%s %s", u.FirstName, u.LastName)
}

// Hash generates a unique checksum of the user's current identity.
// If the name changes, the hash changes, forcing a cache miss and a DB update.
func (u *ExternalUser) Hash() string {
	return fmt.Sprintf("%x", sha256.Sum256(
		fmt.Appendf(
			nil,
			"%s:%s:%s",
			u.ID,
			u.FirstName,
			u.LastName,
		)))
}
