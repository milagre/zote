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

func assertUserAuth(t *testing.T, userID string, auth *UserAuth) {
	assert.Equal(t, userID, auth.UserID)
	assert.NotNil(t, auth.Created)

	switch userID {
	case "1":
		switch auth.ID {
		case "1":
			assert.Equal(t, "password", auth.Provider)
			assert.Equal(t, "P@ssw0rd!", auth.Data)

		case "2":
			assert.Equal(t, "oauth2", auth.Provider)
			assert.Equal(t, "{\"token\":\"1234\"}", auth.Data)

		default:
			assert.Failf(t, "Unspecified auth ID %s for user ID %s - no expectations defined", auth.ID, userID)
		}

	default:
		assert.Failf(t, "Unspecified auth for user ID %s - no expectations defined", userID)
	}
}
