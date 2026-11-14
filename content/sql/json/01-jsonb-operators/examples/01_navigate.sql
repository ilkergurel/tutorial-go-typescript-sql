-- LEARN(sql.json.jsonb-operators): -> keeps jsonb, ->> gives text; JSON null is not SQL NULL.
CREATE TABLE ci_events (
  id      int   PRIMARY KEY,
  payload jsonb NOT NULL          -- a CI webhook body, stored as received
);
INSERT INTO ci_events VALUES
  (1, '{"build": {"number": 108, "branch": "main", "agent": {"os": "linux"}},
        "results": [{"test": "Giriş", "status": "passed", "ms": 120},
                    {"test": "Çıkış", "status": "failed", "ms": 85}],
        "tags": ["nightly", "smoke"], "retry_of": null}'),
  (2, '{"build": {"number": 9, "branch": "feature/kvkk"},
        "results": [{"test": "Giriş", "status": "passed", "ms": 130}],
        "tags": []}');

-- Chain -> to walk down, or give #> / #>> the whole path. JSON arrays
-- are 0-based, and negative indexes count from the end.
SELECT id,
       payload -> 'build' -> 'number'          AS number_jsonb,
       payload -> 'build' ->> 'branch'         AS branch_text,
       payload #>> '{build,agent,os}'          AS agent_os,
       payload -> 'results' -> 0 ->> 'test'    AS first_test,
       payload -> 'results' -> -1 ->> 'status' AS last_status,
       payload -> 'tags' ->> 0                 AS first_tag
FROM ci_events
ORDER BY id;

-- A key holding JSON null and a missing key look alike through ->>.
SELECT id,
       payload -> 'retry_of'               AS arrow,
       payload -> 'retry_of' IS NULL       AS arrow_is_null,
       payload ->> 'retry_of' IS NULL      AS text_is_null,
       payload ? 'retry_of'                AS has_key,
       jsonb_typeof(payload -> 'retry_of') AS json_type
FROM ci_events
ORDER BY id;

-- ->> returns text, and text compares character by character.
SELECT id,
       payload -> 'build' ->> 'number' > '100'      AS as_text,
       (payload -> 'build' ->> 'number')::int > 100 AS as_int,
       payload -> 'build' -> 'number' > '100'       AS as_jsonb
FROM ci_events
ORDER BY id;

-- Text on the left, a number on the right: no operator matches.
SELECT id FROM ci_events WHERE payload -> 'build' ->> 'number' = 108;
