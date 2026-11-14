---
id: sql.search.fts
title: "Full-text search: tsvector, tsquery, ranking, a stored tsvector with GIN"
track: sql
module: search
order: 1
level: advanced
minutes: 15
since: pg12
prerequisites: [sql.indexes.gin-brgin, sql.constraints.generated]
compare_with: [sql.search.turkish, sql.search.trigram, sql.querying.patterns, fullstack.tracelite.step-12-search]
tags: [full-text-search, tsvector, tsquery, ts_rank, ts_headline, gin, generated-columns]
summary: >
  Full-text search reduces documents and queries to normalized lexemes and matches
  them with @@. Parse what users type with websearch_to_tsquery, order hits with
  ts_rank, and keep a STORED tsvector column under a GIN index.
examples:
  - id: tsvector-tsquery
    file: examples/01_tsvector_tsquery.sql
    title: "Documents, queries and @@: four query parsers and prefix matching"
    runtime: sql
    check: exact
    annotations:
      - lines: "2-3"
        note: >
          The english configuration lowercases, drops stop words and stems:
          exports becomes export, archived becomes archiv, every becomes everi.
          Lexemes are index keys, not words. Dropped stop words still count as
          positions (are was 10, so archiv is 11), which keeps phrase distances
          honest.
      - lines: "6-9"
        note: >
          Only to_tsquery reads operators from its input (&, |, !, <-> and
          parentheses). plainto_tsquery ANDs the words, phraseto_tsquery chains
          them with <-> (followed by), and websearch_to_tsquery understands
          quotes, or and a leading dash. All four run the words through the
          configuration first, so signed is looked up as sign.
      - lines: "12-16"
        note: >
          exported cases matches because both sides were stemmed to export and
          case. A quoted phrase has to appear in that order, so "test linked"
          fails, and -linked excludes the document.
      - lines: "19-24"
        note: >
          A prefix (:*) is stemmed like any other query word. The stemmer leaves
          migrati as it is, and the stored lexeme migrat doesn't start with
          migrati, so the search fails mid-word although migr and migration
          succeed. The simple configuration doesn't stem: every prefix of
          migrations matches.
      - lines: "27"
        note: >
          Two words without an operator are a syntax error for to_tsquery
          (SQLSTATE 42601). A search box fed into it fails on the first space,
          quote or stray parenthesis. websearch_to_tsquery never raises a
          syntax error, which, in the docs' words, makes it possible to use raw
          user-supplied input.
  - id: ranking
    file: examples/02_ranking.sql
    title: "Weights, ts_rank and ts_rank_cd, and ts_headline snippets"
    runtime: sql
    check: exact
    annotations:
      - lines: "14-16"
        note: >
          Both functions return float4. Rounding keeps the display stable. A
          rank has no unit and no fixed scale: compare ranks within one query,
          never across queries.
      - lines: "17"
        note: >
          ts_rank_cd (cover density) scores the shortest stretches of text that
          contain all the query terms, so it needs positions and returns 0 for a
          stripped vector. Both functions put REQ-2, where the words share a
          sentence, above REQ-3, where they are a sentence apart.
      - lines: "20-22"
        note: >
          setweight labels every title lexeme A and every body lexeme B, and ||
          appends the body's lexemes after the title's. The flat vector has no
          labels, so all its lexemes count as D. With the default weights
          (D 0.1, C 0.2, B 0.4, A 1.0) REQ-1's rank grows exactly tenfold and
          the others fourfold: the title match moves REQ-1 far ahead.
      - lines: "23"
        note: >
          REQ-4 mentions the audit log but not export. websearch_to_tsquery
          ANDs the words, so REQ-4 doesn't match and is never ranked.
      - lines: "28-29"
        note: >
          ts_headline parses the original text again, which costs more than the
          match itself: call it only for the rows you display. MaxFragments=2
          lets REQ-3's snippet show both distant matches, joined by "...". The
          default markers are <b> and </b>, and the output is not safe HTML.
  - id: stored-gin
    file: examples/03_stored_gin.sql
    title: "A stored tsvector column with GIN, and what ranking costs"
    runtime: sql
    check: exact
    annotations:
      - lines: "6-8"
        note: >
          The vector is computed on every INSERT and UPDATE and kept in the row.
          The expression must be immutable: to_tsvector('english', ...) is, the
          one-argument form (which depends on a setting) is only stable. Write
          STORED: in PG 18 a generated column without it is virtual, and
          virtual columns can't be indexed.
      - lines: "19"
        note: >
          A stored vector shows the labels and the concatenation: title lexemes
          at positions 1 to 4 with A, body lexemes from 5 on with B. can and the
          are stop words, so positions 6 and 8 are missing.
      - lines: "20"
        note: >
          For each lexeme, GIN keeps the rows that contain it. It stores no
          weight labels, so a query that filters on them (export:A) rechecks
          every candidate row.
      - lines: "24-29"
        note: >
          The planner folds websearch_to_tsquery into a constant. The Bitmap
          Index Scan finds all 4,000 matches, the Sort needs ts_rank for every
          one of them, and only then does Limit keep 5. A GIN index can't hand
          rows back in rank order, so ranking costs grow with the number of
          matches, not with LIMIT.
      - lines: "35-39"
        note: >
          Every fifth row has both words in its title. Rows 5, 25, 45... also
          say export in the body, which puts them first. Equal ranks are common
          in real data too: the id tiebreaker keeps pages from reshuffling.
use_cases:
  - "Search boxes over prose: requirement titles and descriptions in TraceLite, articles, tickets, product descriptions, help pages"
  - "A Ctrl+K palette over a few thousand records (lessons, documents, commands), with prefix matching for the word being typed"
  - "Filtering by words inside SQL, combined with ordinary WHERE conditions, joins and permissions in the same query"
strengths:
  - "Unlike a separate Elasticsearch cluster, the index is updated in the same transaction as the row: a committed requirement is searchable at once, a rolled-back one never was, and there is no sync job"
  - "Unlike ILIKE '%export%', it matches words after normalization: exported finds exports, and stop words don't count"
  - "websearch_to_tsquery accepts whatever a user types (quotes, or, -) without raising errors"
  - "One STORED tsvector column with a GIN index serves every query; queries name the column, so they can't drift from the indexed expression"
weaknesses:
  - "ts_rank uses only the document and the query: unlike BM25 in Lucene or Elasticsearch, it doesn't know how rare a word is across the table (no IDF)"
  - "GIN finds matches but can't return them in rank order; every match is ranked before LIMIT applies"
  - "No typo tolerance: exprot finds nothing (pg_trgm, two lessons on, covers typos)"
  - "Stemmers are rule-based per language; prefixes go through them too, so type-ahead needs an unstemmed vector"
pitfalls:
  - title: "Passing raw user input to to_tsquery"
    detail: "to_tsquery parses its own query language. Two words without an operator are a syntax error (SQLSTATE 42601), and parentheses, quotes or ! in the input change the query."
    bad: "SELECT id FROM requirements WHERE search @@ to_tsquery('english', $1);"
    good: "SELECT id FROM requirements WHERE search @@ websearch_to_tsquery('english', $1);"
  - title: "Prefix search through a stemmer"
    detail: "A prefix is stemmed first: migrati:* stays migrati and misses the stored lexeme migrat. For type-ahead, match the unfinished last word against a simple (unstemmed) vector, built from a word your code has split and quoted."
    bad: "to_tsvector('english', title) @@ to_tsquery('english', 'migrati:*')"
    good: "to_tsvector('simple', title) @@ to_tsquery('simple', 'migrati:*')"
  - title: "A generated tsvector without STORED (PG 18)"
    detail: "Since PG 18 a generated column without a keyword is virtual, and CREATE INDEX on it fails with 'indexes on virtual generated columns are not supported'. PG 12 to 17 required STORED, so old snippets work as written; new ones that drop the keyword don't."
    bad: "search tsvector GENERATED ALWAYS AS (to_tsvector('english', body))"
    good: "search tsvector GENERATED ALWAYS AS (to_tsvector('english', body)) STORED"
  - title: "Rendering ts_headline output as HTML"
    detail: "The docs warn that ts_headline output is not guaranteed safe for web pages: it removes only some tags from the document, so it is no defence against XSS. Pick markers that can't occur in the text, escape the snippet, then turn the markers into <mark> elements (in React, render text nodes)."
  - title: "Searching identifiers with full-text search"
    detail: "The parser reads REQ-104 as the word req and the signed integer -104, so a search for 104 finds nothing. Match codes on their own column with = or pg_trgm, and keep FTS for prose."
quiz:
  - q: "A user types into a search box. Which function should receive that text?"
    options: ["to_tsquery", "plainto_tsquery", "websearch_to_tsquery", "to_tsvector"]
    answer: 2
    why: "websearch_to_tsquery never raises a syntax error and understands quotes, or and a leading dash; to_tsquery fails on plain multi-word input."
  - q: "The title is labeled A, the body B, default weights. export appears once in REQ-1's title and once in REQ-2's body; the rows are otherwise alike. Which ranks higher for export?"
    options: ["REQ-1", "REQ-2", "Neither: ts_rank ignores labels"]
    answer: 0
    why: "Default weights are D 0.1, C 0.2, B 0.4, A 1.0: the title occurrence is weighted 1.0, the body occurrence 0.4."
  - q: "There is a GIN index on search. What does it do for ORDER BY ts_rank(search, q) DESC LIMIT 10?"
    options: ["Returns the 10 best rows directly, in rank order", "Finds all matching rows; each is ranked and sorted before LIMIT applies", "Nothing: GIN can't be used together with ORDER BY"]
    answer: 1
    why: "GIN answers which rows match. It keeps no ranks, so the executor ranks every match and sorts before the limit."
  - q: "The user has typed migrati. Why does to_tsquery('english', 'migrati:*') miss a document about migrations?"
    options: ["Prefix matching needs a GiST index", "The fragment stays migrati after stemming, and the stored lexeme migrat doesn't start with it", "The :* syntax only works in websearch_to_tsquery"]
    answer: 1
    why: "Query words, prefixes included, go through the configuration's stemmer; use an unstemmed simple vector for type-ahead."
further_reading:
  - title: "PostgreSQL docs: Full Text Search, Introduction"
    url: "https://www.postgresql.org/docs/current/textsearch-intro.html"
  - title: "PostgreSQL docs: Controlling Text Search (parsing queries, ranking, headlines)"
    url: "https://www.postgresql.org/docs/current/textsearch-controls.html"
  - title: "PostgreSQL docs: Tables and Indexes (a generated tsvector column)"
    url: "https://www.postgresql.org/docs/current/textsearch-tables.html"
  - title: "PostgreSQL docs: Text Search Functions and Operators"
    url: "https://www.postgresql.org/docs/current/functions-textsearch.html"
  - title: "PostgreSQL docs: Preferred Index Types for Text Search"
    url: "https://www.postgresql.org/docs/current/textsearch-indexes.html"
---

## Concept

Full-text search is two data types and an operator. A `tsvector` is a document reduced
to its **lexemes**, normalized words with their positions. A `tsquery` is a query reduced
the same way, plus operators: `&`, `|`, `!` and `<->` (followed by). `vector @@ query`
says whether the document satisfies the query.

A **text search configuration** does the reducing. Its parser splits text into typed
tokens (words, numbers, e-mail addresses...), and for each token type it lists
dictionaries that lowercase the token, drop it as a stop word or stem it. `english`
turns *exports*, *exported* and *exporting* into `export`. The next lesson compares
`english`, `turkish` and `simple`.

On top of matching:

- **Parsing user input:** `websearch_to_tsquery` reads quotes, `or` and `-` and never
  raises a syntax error. `to_tsquery` is the full query language, for queries your code
  builds, including prefixes (`migr:*`).
- **Ranking:** `ts_rank` and `ts_rank_cd` score a match. `setweight` labels lexemes
  A to D so that a title counts more than a body.
- **Snippets:** `ts_headline` marks the matching words in the original text.
- **Speed:** a `tsvector` column, `GENERATED ALWAYS AS (...) STORED`, under a GIN index.

**StackLab has no search yet.** `docs/ARCHITECTURE.md` plans a
`search tsvector GENERATED ALWAYS AS (...) STORED` column on a `lessons` table, the
engine for a Ctrl+K palette. Today the lessons are bundled into the frontend at build
time, and the app database has neither that table nor a search endpoint. The third
example builds the planned design on TraceLite requirements instead.

## Coming from C/C++/Java/Python/JS

- **Lucene / Elasticsearch:** a configuration is an analyzer (tokenizer plus filter
  chain), a `tsvector` is the analyzed field, GIN is the inverted index, and
  `websearch_to_tsquery` is a simple query-string parser. Two differences matter. The
  index lives in your transaction, so there is no refresh interval and no second system
  to keep in sync. And `ts_rank` looks at one document and the query only: unlike BM25
  it has no IDF, so a word that appears in every row weighs as much as a rare one.
- **Java/Go:** don't build `to_tsquery` input by concatenating user text. It's the same
  mistake as concatenating SQL, except it yields syntax errors and wrong queries instead
  of injection. Pass the text as a parameter to `websearch_to_tsquery`.
- **Python/JS:** `ILIKE '%export%'` is `s.includes('export')`: it finds *exporter* but
  not *exported* when the user typed *exports*. Full-text search matches words after
  normalization, which is what people expect from a search box. Substrings and typos are
  `pg_trgm`'s job, two lessons on.

## How it works

1. The parser splits the text into typed tokens; `ts_debug` shows them. A requirement
   code like `REQ-104` becomes the word `req` and the signed integer `-104`.
2. For each token, the configuration's dictionaries run in order until one recognizes it.
   It returns lexemes, or an empty list for a stop word. Snowball stemmers recognize
   every word, so they come last.
3. `to_tsvector` collects the lexemes, sorted and de-duplicated, with positions (and
   weights after `setweight`). Stop words leave gaps in the positions.
4. A GIN index on the column maps each lexeme to the rows that contain it. `&` intersects
   those row lists and `|` unions them. Ranking needs positions and weights, which the
   index doesn't keep, so it reads the stored vector of every matching row.

## Design notes

- **Store the vector.** A STORED generated column parses the text once per write. The
  docs list two advantages over an expression index on `to_tsvector('english', body)`:
  queries name the column instead of repeating the expression (and its configuration),
  and rechecks read the stored vector instead of parsing the text again. The expression
  index is smaller. Write `STORED` explicitly (PG 18 defaults to virtual).
- **Name the configuration in every call**, queries included. Without it,
  `default_text_search_config` decides, and the next lesson shows that silently finding
  nothing.
- **Search box text** goes to `websearch_to_tsquery`. A Ctrl+K palette that searches as
  you type also needs a prefix for the unfinished last word, and prefixes don't survive
  stemming: keep a `simple` vector for that, and build the prefix query from a word your
  code has split and quoted.
- **Rank what you display.** Order by `rank DESC, id` so that equal ranks don't reshuffle
  between pages. When thousands of rows match, ranking them is the cost; narrow the
  candidates with ordinary filters first.
- **Identifiers aren't prose.** Requirement codes, ticket numbers and version strings go
  in their own column, matched with `=` or `pg_trgm`.
- **Size matters less than you think, both ways.** For a few hundred lessons the planner
  will scan the stored vectors sequentially and that is fine; the GIN index pays off as
  the table grows. Past millions of documents, or when you need relevance tuning,
  synonyms per tenant or faceting, a dedicated search engine starts to earn its upkeep.
