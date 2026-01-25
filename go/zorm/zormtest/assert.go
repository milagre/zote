package zormtest

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertAccountFields(t *testing.T, accountID string, account *Account, fields []string) {
	assert.Equal(t, accountID, account.ID)
	assert.NotNil(t, account.Created)

	var expectedFields map[string]string
	switch accountID {
	case "1":
		expectedFields = map[string]string{
			"Company":      "Acme, Inc.",
			"ContactEmail": "contact@acme.example",
		}
	case "2":
		expectedFields = map[string]string{
			"Company":      "Dunder Mifflin",
			"ContactEmail": "contact@dundermifflin.example",
		}
	case "3":
		expectedFields = map[string]string{
			"Company":      "Explorers, LLC",
			"ContactEmail": "dora@explorers.test",
		}
	default:
		assert.Failf(t, "Unspecified account for account ID %s - no expectations defined", accountID)
	}

	for _, field := range fields {
		assert.Equal(t, expectedFields[field], reflect.ValueOf(account).Elem().FieldByName(field).Interface())
	}
}

func assertAccount(t *testing.T, accountID string, account *Account) {
	assertAccountFields(t, accountID, account, []string{"Company", "ContactEmail"})
}

func assertAddress(t *testing.T, address *UserAddress) {
	assert.NotNil(t, address.Created)

	switch address.ID {
	case "1":
		assert.Equal(t, "123 Loony Lane", address.Street)
		assert.Equal(t, "Acmeton", address.City)
		assert.Equal(t, "RI", address.State)

	case "2":
		assert.Equal(t, "1725 Slough Avenue", address.Street)
		assert.Equal(t, "Scranton", address.City)
		assert.Equal(t, "PA", address.State)

	default:
		assert.Failf(t, "Unspecified address for address ID %s - no expectations defined", address.ID)
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
