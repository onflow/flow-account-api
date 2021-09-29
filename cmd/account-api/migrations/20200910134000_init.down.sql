DROP TABLE accounts;
DROP TABLE public_keys;

-- TRIGGERS
DROP FUNCTION IF EXISTS update_row_modified_function_();
--
-- EXTENSIONS
DROP EXTENSION IF EXISTS "uuid-ossp";