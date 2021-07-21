package testdata

import (
	"time"
)

type EmailAddress string

//go:generate go run ../main.go -type User
type User struct {
	ID, Name string
	Age      int
	Emailer
	CreatedAt    time.Time
	privateField string
}

type Emailer struct {
	ID    string
	Email EmailAddress
}
