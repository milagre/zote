package zormtest

import (
	"time"
)

type Account struct {
	ID       string
	Created  time.Time
	Modified *time.Time

	Company string

	Users []*User
}

type User struct {
	ID       string
	Created  time.Time
	Modified *time.Time

	AccountID string
	Account   *Account

	Address *UserAddress

	Auths []*UserAuth

	FirstName string
}

type UserAuth struct {
	ID       string
	Created  time.Time
	Modified *time.Time

	UserID string
	User   *User

	Provider string
	Data     string
}

type UserAddress struct {
	Created  time.Time
	Modified *time.Time

	UserID string

	Street string
	City   string
	State  string
}
