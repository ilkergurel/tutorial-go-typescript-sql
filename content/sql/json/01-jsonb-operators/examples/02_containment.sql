CREATE TABLE ci_events (
  id      int   PRIMARY KEY,
  payload jsonb NOT NULL
);
INSERT INTO ci_events VALUES
  (1, '{"build": {"branch": "main"}, "tags": ["nightly", "smoke"],
        "results": [{"test": "Giriş", "status": "passed", "ms": 120},
                    {"test": "Çıkış", "status": "failed", "ms": 85}]}'),
  (2, '{"build": {"branch": "feature/kvkk"}, "tags": ["smoke"],
        "results": [{"test": "Giriş", "status": "passed", "ms": 130}]}'),
  (3, '{"build": {"branch": "main"}, "tags": [], "results": [], "retry_of": 1}');

-- @> asks: does the left document contain this shape? Nested objects match
-- at the same path; array elements match in any order; extra keys are ignored.
SELECT id,
       payload @> '{"build": {"branch": "main"}}'         AS on_main,
       payload @> '{"branch": "main"}'                     AS wrong_level,
       payload @> '{"tags": ["smoke", "nightly"]}'         AS both_tags,
       payload @> '{"tags": "smoke"}'                      AS scalar_tag,
       payload -> 'tags' @> '"smoke"'                      AS array_has_smoke,
       payload @> '{"results": [{"status": "failed"}]}'    AS any_failed
FROM ci_events
ORDER BY id;

-- ? and friends test strings: top-level keys, or string elements of an array.
SELECT id,
       payload ? 'retry_of'                          AS has_retry,
       payload ? 'branch'                            AS nested_key,
       payload -> 'tags' ? 'smoke'                   AS smoke,
       payload -> 'tags' ?| ARRAY['nightly', 'perf'] AS any_of,
       payload -> 'tags' ?& ARRAY['nightly', 'smoke'] AS all_of
FROM ci_events
ORDER BY id;

-- jsonb equality is structural and numeric: 120.0 = 120, key order is irrelevant.
SELECT '{"ms": 120.0, "test": "Giriş"}'::jsonb
         = '{"test": "Giriş", "ms": 120}'::jsonb AS equal,
       '{"ms": 120.0}'::jsonb                    AS kept_as_written;

-- The plain json type stores text and has none of these operators.
SELECT payload::json @> '{"tags": ["smoke"]}' FROM ci_events;
