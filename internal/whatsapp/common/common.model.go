package common

import "github.com/google/uuid"

type Contact struct {
	ID  uuid.UUID `json:"id" db:"id"`
	Iss string    `json:"iss" db:"iss"`
	Sub string    `json:"sub" db:"sub"`
}

func (contact *Contact) GetSub() string   { return contact.Sub }
func (contact *Contact) GetIss() string   { return contact.Iss }
func (contact *Contact) GetID() uuid.UUID { return contact.ID }
