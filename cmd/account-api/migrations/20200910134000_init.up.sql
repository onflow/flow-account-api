-- EXTENSIONS
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
--
-- TRIGGERS
CREATE OR REPLACE FUNCTION update_row_modified_function_()
RETURNS TRIGGER
AS
$$
BEGIN
    -- ASSUMES the table has a column named exactly "updated_at".
    -- Fetch date-time of actual current moment from clock, rather than start of statement or start of transaction.
    NEW.updated_at = now();
    RETURN NEW;
END;
$$
language 'plpgsql';
--

CREATE TABLE accounts
(
    address TEXT PRIMARY KEY
);

CREATE TABLE public_keys
(
    account_address TEXT,
    public_key TEXT PRIMARY KEY,
    sig_algo TEXT NOT NULL,
    hash_algo TEXT NOT NULL,
    CONSTRAINT fk_account
      FOREIGN KEY(account_address)
	    REFERENCES accounts(address)
);
