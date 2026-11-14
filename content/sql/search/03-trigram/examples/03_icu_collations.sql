-- Nondeterministic ICU collations: equal can mean "equal ignoring case or accents".
CREATE COLLATION tr_ci  (provider = icu, locale = 'tr-u-ks-level2',  deterministic = false);
CREATE COLLATION und_ci (provider = icu, locale = 'und-u-ks-level2', deterministic = false);
CREATE COLLATION tr_ai  (provider = icu, locale = 'tr-u-ks-level1',  deterministic = false);
CREATE COLLATION und_ai (provider = icu, locale = 'und-u-ks-level1', deterministic = false);

SELECT a, b,
       a = b COLLATE tr_ci  AS tr_ci,
       a = b COLLATE und_ci AS und_ci,
       a = b COLLATE tr_ai  AS tr_ai,
       a = b COLLATE und_ai AS und_ai
FROM (VALUES ('ISPARTA', 'ısparta'), ('İZMİR', 'izmir'), ('IZMIR', 'izmir'),
             ('Çeşme', 'cesme'), ('Iğdır', 'igdir')) AS p(a, b);

-- On a column, the collation governs =, LIKE (PG 18), UNIQUE and its index.
CREATE TABLE users (username text COLLATE tr_ci PRIMARY KEY);
INSERT INTO users VALUES ('ışık'), ('Deniz');

SELECT username FROM users WHERE username = 'IŞIK';
SELECT username FROM users WHERE username LIKE 'DEN%';

INSERT INTO users VALUES ('DENİZ');
