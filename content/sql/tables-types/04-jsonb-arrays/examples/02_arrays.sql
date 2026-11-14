CREATE TABLE articles (
  id    int    PRIMARY KEY,
  title text   NOT NULL,
  tags  text[] NOT NULL DEFAULT '{}'
);

INSERT INTO articles VALUES
  (1, 'Goroutines in practice', ARRAY['go', 'concurrency']),
  (2, 'timestamptz explained',  ARRAY['sql', 'time']),
  (3, 'pgx and transactions',   ARRAY['go', 'sql']),
  (4, 'Draft',                  '{}');

-- Arrays are 1-based, and reading outside the bounds yields NULL, not an
-- error. For an empty array, array_length() is NULL and cardinality() is 0.
SELECT id, tags[1] AS first, tags[0] AS index_zero, tags[3] AS past_end,
       array_length(tags, 1) AS array_length, cardinality(tags) AS cardinality
FROM articles
ORDER BY id;

-- Membership: = ANY for one value, @> for "contains all of these".
SELECT id, title FROM articles WHERE 'sql' = ANY (tags) ORDER BY id;
SELECT id, title FROM articles WHERE tags @> ARRAY['go', 'sql'] ORDER BY id;

-- unnest turns an array back into rows: articles per tag.
SELECT tag, count(*) AS articles
FROM articles, unnest(tags) AS tag
GROUP BY tag
ORDER BY tag;

-- No foreign key reaches inside an array, so a typo is stored happily.
UPDATE articles SET tags = array_append(tags, 'golang') WHERE id = 1
RETURNING id, tags;
