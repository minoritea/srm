package testdata

import (
	"github.com/minoritea/srm/testdata/pkg"
	"time"
)

type EmailAddress string

//go:generate go run ../cmd/srm/main.go -type User
type User struct {
	ID, Name string `srm:"name=name"`
	Age      int
	Emailer
	CreatedAt    time.Time
	privateField string
	pkg2.Writer
	FamillyName  string `srm:"name=last_name"`
	IgnoredField string `srm:"-"`
	Programmer
}

type Emailer struct {
	ID    string
	EMail EmailAddress
}
