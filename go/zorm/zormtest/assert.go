package zormtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertAccount(t *testing.T, accountID string, account *Account) {
	assert.Equal(t, accountID, account.ID)
	assert.NotNil(t, account.Created)

	switch accountID {
	case "1":
		assert.Equal(t, "Acme, Inc.", account.Company)

	case "2":
		assert.Equal(t, "Dunder Mifflin", account.Company)
	}
}

func assertAddress(t *testing.T, userID string, address *UserAddress) {
	assert.Equal(t, userID, address.UserID)
	assert.NotNil(t, address.Created)

	switch userID {
	case "1":
		assert.Equal(t, "123 Loony Lane", address.Street)
		assert.Equal(t, "Acmeton", address.City)
		assert.Equal(t, "RI", address.State)

	case "2":
		assert.Equal(t, "1725 Slough Avenue", address.Street)
		assert.Equal(t, "Scranton", address.City)
		assert.Equal(t, "PA", address.State)

	default:
		assert.Failf(t, "Unspecified address for user ID %s - no expectations defined", userID)
	}
}
