-- LEARN(sql.modeling.normalization): one fact, one place; the keys say which place.
-- A test log kept in a spreadsheet, imported as-is: one row per test run.
CREATE TABLE sheet (
  test_code    text,
  run_on       date,
  req_code     text,
  req_title    text,
  tester       text,
  tester_email text,
  result       text,
  tags         text
);
INSERT INTO sheet VALUES
  ('TC-1', '2026-11-02', 'REQ-1', 'Pay by card',    'Ayşe',   'ayse@example.com',   'passed', 'smoke, payment'),
  ('TC-2', '2026-11-02', 'REQ-1', 'Pay by card',    'Mehmet', 'mehmet@example.com', 'failed', 'payment'),
  ('TC-1', '2026-11-09', 'REQ-1', 'Card payment',   'Ayşe',   'ayse@example.com',   'passed', 'smoke, payment'),
  ('TC-3', '2026-11-09', 'REQ-2', 'Invoice e-mail', 'Mehmet', 'mehmet@example.com', 'passed', 'email');

-- Update anomaly: REQ-1 was renamed in one row only. A dependency
-- req_code -> req_title holds when each code has exactly one title.
SELECT req_code, string_agg(DISTINCT req_title, ' | ' ORDER BY req_title) AS titles
FROM sheet
GROUP BY req_code
HAVING count(DISTINCT req_title) > 1;

-- Insertion anomaly: a requirement with no tests yet only fits as a
-- row of NULLs, and now "rows" no longer means "test runs".
INSERT INTO sheet (req_code, req_title) VALUES ('REQ-3', 'Refund to card');
SELECT count(*) AS rows, count(run_on) AS runs FROM sheet;

-- Deletion anomaly: TC-3's only run was logged by mistake. Deleting it
-- also deletes the only record that REQ-2 exists.
DELETE FROM sheet WHERE test_code = 'TC-3';
SELECT DISTINCT req_code, req_title FROM sheet ORDER BY req_code, req_title;
