--
-- accounts
--

CREATE TABLE accounts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	created DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	modified DATETIME DEFAULT NULL,
	company TEXT NOT NULL
);


CREATE TRIGGER accounts_modified_timestamp
AFTER UPDATE ON accounts
BEGIN
   UPDATE accounts SET modified = datetime('now') WHERE id = NEW.id;
END;

INSERT INTO accounts (company) VALUES
("Acme"),
("Dunder Mifflin");

UPDATE accounts SET company="Acme, Inc." where company="Acme";

--
-- users
--

CREATE TABLE users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	created DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	modified DATETIME DEFAULT NULL,
    account_id INTEGER NOT NULL,
    first_name TEXT NOT NULL,

    FOREIGN KEY (account_id) REFERENCES accounts(id)
);


CREATE TRIGGER users_modified_timestamp
AFTER UPDATE ON users
BEGIN
   UPDATE users SET modified = datetime('now') WHERE id = NEW.id;
END;

INSERT INTO users (account_id, first_name) SELECT id, "Daffy" FROM accounts WHERE company="Acme, Inc.";
INSERT INTO users (account_id, first_name) SELECT id, "Dwight" FROM accounts WHERE company="Dunder Mifflin";

--
-- user auths
--

CREATE TABLE user_auths (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	created DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	modified DATETIME DEFAULT NULL,
    user_id INTEGER NOT NULL,
    provider TEXT NOT NULL,
	data TEXT NOT NULL,

    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TRIGGER user_auths_modified_timestamp
AFTER UPDATE ON user_auths
BEGIN
   UPDATE user_auths SET modified = datetime('now') WHERE id = NEW.id;
END;

INSERT INTO user_auths (user_id, provider, data) SELECT id, 'password', 'P@ssw0rd!' FROM users WHERE first_name='Daffy';
INSERT INTO user_auths (user_id, provider, data) SELECT id, 'oauth2', '{"token":"1234"}' FROM users WHERE first_name='Daffy';

INSERT INTO user_auths (user_id, provider, data) SELECT id, 'password', 't0tally_S3CURE!' FROM users WHERE first_name='Dwight';
INSERT INTO user_auths (user_id, provider, data) SELECT id, 'passkey', '{"secret":"5678"}' FROM users WHERE first_name='Dwight';

--
-- user addresses
--

CREATE TABLE user_addresses (
	created DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	modified DATETIME DEFAULT NULL,
    user_id INTEGER NOT NULL PRIMARY KEY,
    street TEXT NOT NULL,
	city TEXT NOT NULL,
	state TEXT NOT NULL,

    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TRIGGER user_addresses_modified_timestamp
AFTER UPDATE ON user_addresses
BEGIN
   UPDATE user_addresses SET modified = datetime('now') WHERE id = NEW.id;
END;

INSERT INTO user_addresses (user_id, street, city, state) SELECT id, '1725 Slough Avenue', 'Scranton', 'PA' FROM users WHERE first_name='Dwight';
INSERT INTO user_addresses (user_id, street, city, state) SELECT id, '123 Loony Lane', 'Acmeton', 'RI' FROM users WHERE first_name='Daffy';

