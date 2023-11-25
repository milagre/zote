package zormsqlite3_test_structs

type Account struct {
	ID      string
	Company string
}

type User struct {
	ID      string
	Name    string
	Account *Account
}
