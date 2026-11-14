-- Reconcile the test plan with what CI actually reported.
CREATE TABLE planned  (test text PRIMARY KEY);
CREATE TABLE reported (test text PRIMARY KEY, passed boolean NOT NULL);
INSERT INTO planned  VALUES ('login'), ('logout'), ('reset');
INSERT INTO reported VALUES ('login', true), ('reset', false), ('signup', true);

-- FULL: rows from both sides; NULL marks the side without a match.
SELECT p.test AS planned, r.test AS reported, r.passed
FROM planned p
FULL JOIN reported r ON r.test = p.test
ORDER BY coalesce(p.test, r.test);

-- CROSS: every combination, here the matrix a release must cover.
CREATE TABLE browsers (name text PRIMARY KEY);
CREATE TABLE locales  (code text PRIMARY KEY);
CREATE TABLE runs (
  browser text,
  locale  text,
  passed  boolean NOT NULL,
  PRIMARY KEY (browser, locale)
);
INSERT INTO browsers VALUES ('chrome'), ('firefox');
INSERT INTO locales  VALUES ('en-US'), ('tr-TR');
INSERT INTO runs VALUES
  ('chrome', 'en-US', true), ('chrome', 'tr-TR', false), ('firefox', 'en-US', true);

-- Generate what should exist, then LEFT JOIN what does: NULL = never run.
SELECT b.name AS browser, l.code AS locale, r.passed
FROM browsers b
CROSS JOIN locales l
LEFT JOIN runs r ON r.browser = b.name AND r.locale = l.code
ORDER BY b.name, l.code;
