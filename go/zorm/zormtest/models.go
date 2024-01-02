package zormtest

import (
	"time"
)

type Account struct {
	ID       string
	Created  time.Time
	Modified *time.Time

	Company string
}

type User struct {
	ID       string
	Created  time.Time
	Modified *time.Time

	AccountID string
	Account   *Account

	FirstName string
}
