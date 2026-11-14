-- LEARN(sql.search.fts): a document becomes lexemes, a query becomes lexemes and operators.
SELECT to_tsvector('english',
  'Exports must include every linked test case; signed exports are archived.') AS document;

-- Four ways to build a query. Only to_tsquery reads operators from its input.
SELECT to_tsquery('english', 'export & (sign | archive)')            AS to_tsquery,
       plainto_tsquery('english', 'signed exports')                  AS plainto,
       phraseto_tsquery('english', 'linked test case')               AS phraseto,
       websearch_to_tsquery('english', '"test case" export -draft')  AS websearch;

-- @@ asks: does this document satisfy this query?
SELECT q AS typed,
       to_tsvector('english', 'Exports must include every linked test case')
         @@ websearch_to_tsquery('english', q) AS matches
FROM (VALUES ('exported cases'), ('"linked test"'), ('"test linked"'),
             ('export -linked')) AS t(q);

-- Type-ahead needs prefixes (:*), and a prefix goes through the stemmer too.
SELECT q AS typed,
       to_tsvector('english', 'Database migrations run at startup')
         @@ to_tsquery('english', q) AS english,
       to_tsvector('simple', 'Database migrations run at startup')
         @@ to_tsquery('simple', q) AS simple
FROM (VALUES ('migr:*'), ('migrati:*'), ('migration:*')) AS t(q);

-- to_tsquery is a query language, not a search box.
SELECT to_tsquery('english', 'signed exports');
