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



