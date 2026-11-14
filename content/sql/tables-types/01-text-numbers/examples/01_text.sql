CREATE TABLE users (
  full_name text       NOT NULL,  -- no arbitrary limit: the usual choice
  handle    varchar(6) NOT NULL   -- the limit is checked on every write
);

INSERT INTO users VALUES
  ('Şükrü Işık',   'şükrü'),
  ('İpek Çağlar',  'ipekç'),
  ('Ada Lovelace', 'ada');

-- length() counts characters, octet_length() counts UTF-8 bytes.
-- varchar(n) counts characters too: 'şükrü' fits in 6 although it is 8 bytes.
SELECT handle, length(handle) AS chars, octet_length(handle) AS bytes
FROM users
ORDER BY handle;

-- Case mapping follows the collation. The database default maps i to I;
-- Turkish rules need an ICU collation.
SELECT upper(handle) AS default_rules,
       upper(handle COLLATE "tr-x-icu") AS turkish_rules
FROM users
ORDER BY handle;

-- An explicit cast to varchar(n) truncates silently (the SQL standard says so)...
SELECT 'kahramanmaraşlı'::varchar(6) AS cast_result;

-- ...but storing an over-long value is an error.
INSERT INTO users VALUES ('Kahramanmaraşlı Ayşe', 'kahramanmaraşlı');
