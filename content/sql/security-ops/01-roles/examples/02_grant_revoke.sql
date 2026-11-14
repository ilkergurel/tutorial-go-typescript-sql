CREATE TABLE customers (
  id   int PRIMARY KEY,
  name text NOT NULL,
  tckn char(11) NOT NULL -- national ID number: personal data under KVKK
);
INSERT INTO customers VALUES (1, 'Ayşe Yılmaz', '10000000146');

-- NULL means "default privileges": everything for the owner, nothing for others.
SELECT relacl FROM pg_class WHERE relname = 'customers';

-- PUBLIC is every role, present and future. Here it gets two columns.
GRANT SELECT (id, name) ON customers TO PUBLIC;
SELECT has_table_privilege('public', 'customers', 'SELECT') AS whole_table,
       has_column_privilege('public', 'customers', 'name', 'SELECT') AS name_col,
       has_column_privilege('public', 'customers', 'tckn', 'SELECT') AS tckn_col;

-- The owner plays its own app role: owners may revoke their own privileges.
REVOKE ALL ON customers FROM playground;
GRANT INSERT, UPDATE (name) ON customers TO playground;

-- Table-level grants live in pg_class.relacl, column grants in pg_attribute.
SELECT relacl FROM pg_class WHERE relname = 'customers';
SELECT attname, attacl FROM pg_attribute
WHERE attrelid = 'customers'::regclass AND attnum > 0
ORDER BY attnum;

SELECT id, name FROM customers;
UPDATE customers SET name = 'Ayşe Kaya' WHERE id = 1;

-- Ownership is not a privilege: REVOKE ALL left ALTER and DROP untouched.
ALTER TABLE customers ADD COLUMN email text;

SELECT * FROM customers;
