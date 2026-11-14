SET TimeZone = 'Europe/Istanbul';

CREATE TABLE ci_reports (
  id     int   PRIMARY KEY,
  report jsonb NOT NULL
);
INSERT INTO ci_reports VALUES
  (1, '{"build": "108", "finished": "2026-11-13T06:12:00Z",
        "suites": [{"name": "auth", "slowest_ms": 120},
                   {"name": "export", "slowest_ms": 900}],
        "owner": null}'),
  (2, '{"build": "109", "suites": [{"name": "auth", "slowest_ms": 130}]}');

-- JSON_VALUE extracts one scalar and casts it with RETURNING; JSON_QUERY
-- returns JSON; JSON_EXISTS tests. PASSING binds path variables.
SELECT id,
       JSON_VALUE(report, '$.build' RETURNING int)               AS build,
       JSON_VALUE(report, '$.finished' RETURNING timestamptz)    AS finished,
       JSON_VALUE(report, '$.owner' DEFAULT 'nobody' ON EMPTY)   AS owner,
       JSON_QUERY(report, '$.suites[*].name' WITH WRAPPER)       AS suites,
       JSON_EXISTS(report, '$.suites[*] ? (@.slowest_ms > $max)'
                   PASSING 500 AS max)                           AS has_slow
FROM ci_reports
ORDER BY id;

-- By default every problem becomes NULL: two values where one was expected,
-- an object instead of a scalar, a missing key.
SELECT JSON_VALUE(report, '$.suites[*].name') AS two_values,
       JSON_VALUE(report, '$.suites[0]')      AS an_object,
       JSON_VALUE(report, '$.nope')           AS missing
FROM ci_reports
WHERE id = 1;

-- Ask for errors where a NULL would hide a bug.
SELECT JSON_VALUE(report, '$.suites[*].name' ERROR ON ERROR)
FROM ci_reports
WHERE id = 1;
