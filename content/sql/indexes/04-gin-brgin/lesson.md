---
id: sql.indexes.gin-brgin
title: "GIN (jsonb, arrays, full-text, trigram), BRIN for time series, skip scan (PG 18)"
track: sql
module: indexes
order: 4
level: advanced
minutes: 15
since: pg18
prerequisites: [sql.indexes.composite, sql.tables-types.jsonb-arrays]
compare_with: [sql.json.indexing-design, sql.search.fts, sql.search.trigram, fullstack.tracelite.step-12-search]
tags: [indexes, gin, jsonb, arrays, full-text-search, pg_trgm, brin, skip-scan]
summary: >
  GIN indexes the parts inside a value (jsonb keys, array elements, words,
  trigrams); BRIN stores one min/max per block range, tiny and only useful when
  physical order follows the column; PG 18's skip scan lets a B-tree serve
  queries that omit its leading column.
examples:
  - id: gin-jsonb-arrays
    file: examples/01_gin_jsonb_arrays.sql
    title: "GIN on jsonb and arrays: which operators it serves, and which it doesn't"
    runtime: sql
    check: exact
    annotations:
      - lines: "7-14"
        note: >
          200 brands with 100 products each, four colors, and two tags per
          product: a label and one of 100 categories. Every marka-7 product
          happens to be siyah, so the containment query below matches 100 rows.
      - lines: "17-20"
        note: >
          jsonb_ops makes an entry for every key and every value. @> looks up
          the entries for brand, marka-7, color and siyah and intersects their
          row lists. GIN is always read through a bitmap scan: it yields row
          locations in no useful order. The recheck matters here, because
          keys and values are indexed separately: a row holding "siyah" under
          some other key would also be a candidate.
      - lines: "22-24"
        note: >
          Key existence (?, ?| and ?&) is served by jsonb_ops only. No product
          has a discount key, and the index says so without touching the table.
      - lines: "26-28"
        note: >
          ->> returns text, and text = text belongs to B-tree, not to the GIN
          operator class. Write the condition as attrs @> '{"brand": "marka-7"}',
          or give the key a B-tree expression index on (attrs ->> 'brand'), or,
          if every row has it, a real column (the JSON module's indexing lesson).
      - lines: "31-33"
        note: >
          jsonb_path_ops stores one hash per value together with the keys
          leading to it, instead of separate key and value entries: 24 pages
          instead of 37, and more specific lookups. The price: only @>, @? and
          @@, no key-existence operators.
      - lines: "36-40"
        note: >
          The array operator class serves @>, <@, && and =. 'kategori-7' =
          ANY (tags) asks the same question as tags @> ARRAY['kategori-7'], but
          the planner only matches operators, not meanings: Seq Scan.
  - id: gin-text
    file: examples/02_gin_text.sql
    title: "GIN for words (turkish full-text search) and for substrings (pg_trgm)"
    runtime: sql
    check: exact
    annotations:
      - lines: "1-3"
        note: >
          The turkish configuration lower-cases, drops stop words and stems:
          kitapları becomes kitap, kitapçılar (booksellers) becomes kitapçı, a
          different stem. Stemming is heuristic: yeni (new) became ye, which
          is also the verb "eat". The query kitaplar stems to kitap, so it
          matches kitapları but not kitapçılar.
      - lines: "15-17"
        note: >
          An expression index: to_tsvector('turkish', body) is computed on every
          insert and update, and GIN keeps, for each stem, the list of rows
          containing it.
      - lines: "18-22"
        note: >
          kitap & dün intersects two of those row lists: 1,000 articles (every
          20th one) mention books and yesterday.
      - lines: "24-27"
        note: >
          The one-argument to_tsvector uses the default_text_search_config
          setting, so its result can change with a SET; it can't be indexed and
          doesn't match the indexed expression. The docs require the
          configuration name in both places; a STORED generated tsvector
          column avoids repeating it in every query.
      - lines: "29-31"
        note: >
          pg_trgm is a trusted extension, so the non-superuser playground role
          may create it; the rollback removes it again. Each word is padded
          with two spaces in front and one behind, then cut into three-character
          pieces.
      - lines: "32-37"
        note: >
          A leading % defeats any B-tree, hence the Seq Scan. With the trigram
          index the pattern's own trigrams are looked up, and the candidates
          are rechecked with ILIKE: 223 titles. A pattern too short to contain
          a trigram can't narrow anything down and becomes a full index scan.
      - lines: "39-42"
        note: >
          pg_trgm asks the database's LC_CTYPE, fixed at CREATE DATABASE, which
          characters are letters. The playground's is a real locale, so ı and ğ
          are letters and ığdır has six trigrams, printed in hex because a
          trigram holding non-ASCII letters is stored as a 3-byte hash. Under
          LC_CTYPE 'C' every non-ASCII letter is a separator: ığdır shrinks to
          the trigrams of d and r, exactly those of dür, and vs_dur becomes 1.
      - lines: "43"
        note: >
          pg_trgm lower-cases each word with the database's default collation,
          C.UTF-8 here, which maps I to i, not ı. IĞDIR becomes iğdir and shares no
          trigram with ığdır, so a similarity search for the capitalised name
          scores the real one 0.
  - id: brin-skip-scan
    file: examples/03_brin_skip_scan.sql
    title: "BRIN on an append-only table, and a PG 18 skip scan"
    runtime: sql
    check: exact
    annotations:
      - lines: "1-2"
        note: >
          EXPLAIN prints timestamptz constants in the session's time zone.
          Pinning it keeps the output identical on servers configured for UTC.
      - lines: "8-12"
        note: >
          Rows arrive in time order, four sensors per minute, as in a log or
          an IoT feed: the physical position of a row follows its timestamp.
      - lines: "15-19"
        note: >
          pg_stats.correlation compares the column's order with the physical
          row order: 1.00 for at, 0.25 for the cycling sensor_id, 0.00 for the
          scrambled value. BRIN only helps a column like at. On large tables
          ANALYZE works from a random sample and such numbers shift slightly
          between runs; on this small table they come out the same every time.
      - lines: "21-27"
        note: >
          3 pages of BRIN against 32 of B-tree for a 128-page table: BRIN keeps
          one min/max per range of 8 pages, a B-tree one entry per row. The
          default range is 128 pages (1 MB of table); 8 suits this small table.
      - lines: "29-32"
        note: >
          BRIN can only say "this range may contain matches", so the scan is
          always lossy: Heap Blocks: lossy=8 is one whole range, and Rows
          Removed by Index Recheck: 1016 are its rows outside the hour. 13
          buffers in all (8 table pages, 5 index pages). The Bitmap Index Scan's
          actual rows=80.00 is not a row count; BRIN knows pages, not rows.
          The Planning section is pages the planner itself read.
      - lines: "34-41"
        note: >
          The index leads with sensor_id and the query doesn't mention it.
          Before PG 18 that condition couldn't narrow the scan: all 79 pages, or
          a Seq Scan. PG 18's skip scan repositions once per distinct sensor_id
          (Index Searches: 6 for 4 sensors) and reads 19 index pages. The plan
          never says "skip scan"; Index Searches above 1 for a single range
          condition is the sign.
use_cases:
  - "GIN, jsonb_path_ops: filtering semi-structured attributes with @> (product specs, event payloads, settings)"
  - "GIN on arrays: tags, labels, permission lists queried with @> (has all) and && (has any)"
  - "GIN on tsvector: full-text search in Turkish or English, the engine behind a site search box"
  - "GIN with pg_trgm: LIKE/ILIKE '%fragment%' and fuzzy similarity search on names and titles"
  - "BRIN: huge append-only tables queried by time range (logs, audit trails, sensor readings), where a B-tree would cost gigabytes"
  - "Skip scan: an existing (tenant_id, created_at) index serving an occasional query on created_at alone, when there are few tenants"
strengths:
  - "GIN answers 'contains this element/key/word' questions that no B-tree can, like Lucene's inverted index but inside your transactions"
  - "One GIN index on a jsonb column serves containment queries on any key, without an index per attribute"
  - "BRIN is orders of magnitude smaller than a B-tree and nearly free to maintain on append-only data"
  - "Skip scan (PG 18) can save you a second index for queries that omit the leading column, with no change to the SQL"
weaknesses:
  - "GIN only serves its own operators (@>, ?, &&, @@, LIKE via trigrams): ->> equality, = ANY and ranges fall back to other indexes or a Seq Scan"
  - "GIN is always a bitmap scan: no ordered output, no index-only scans, and inserts are expensive because one row adds many entries"
  - "BRIN is lossy and useless once physical order stops following the column (updates, out-of-order loads); newly appended ranges stay unsummarized until VACUUM or brin_summarize_new_values()"
  - "Skip scan pays off only when the skipped column has few distinct values; with many, the planner still prefers a Seq Scan or another index"
pitfalls:
  - title: "Writing = ANY where @> would use the index"
    detail: "Both ask whether the array contains a value, but only the operator form matches the GIN operator class."
    bad: "SELECT id FROM products WHERE 'kategori-7' = ANY (tags);"
    good: "SELECT id FROM products WHERE tags @> ARRAY['kategori-7'];"
  - title: "->> equality against a GIN-indexed jsonb column"
    detail: "attrs ->> 'brand' = 'x' compares text, which GIN can't serve. Rewrite it as containment, or add a B-tree expression index on (attrs ->> 'brand') for a key you always filter on."
    bad: "WHERE attrs ->> 'brand' = 'marka-7'"
    good: "WHERE attrs @> '{\"brand\": \"marka-7\"}'"
  - title: "Full-text queries without the configuration"
    detail: "An index on to_tsvector('turkish', body) serves only queries that spell the same two-argument call. Store the tsvector in a generated column, or wrap the expression in one SQL function used everywhere."
  - title: "BRIN on a column that doesn't follow the physical order"
    detail: "If rows are updated, deleted and refilled, or loaded out of order, every range's min/max spans nearly everything and BRIN excludes nothing. Check pg_stats.correlation first; near 1 or -1 is what BRIN needs."
  - title: "Trusting pg_trgm with Turkish letters without checking"
    detail: "pg_trgm skips characters it doesn't consider alphanumeric, and the database's LC_CTYPE decides which those are. Under LC_CTYPE 'C' every non-ASCII letter (ı, ğ, ş, ü...) is a separator: show_trgm('ığdır') returns only the trigrams of d and r, similarity('ığdır', 'dür') is 1, and similarity() and trigram index lookups on Turkish words get much weaker (LIKE results stay correct, since candidates are rechecked). The playground has a real LC_CTYPE. On another server, check datctype in pg_database and run show_trgm('ığdır'): six entries is right. LC_CTYPE can't be changed later; fixing it means a new database."
quiz:
  - q: "There is a GIN index on `tags text[]`. Which condition can use it?"
    options: ["'kategori-7' = ANY (tags)", "tags @> ARRAY['kategori-7']", "tags[1] = 'kategori-7'"]
    answer: 1
    why: "Only the array operators of the GIN operator class (@>, <@, &&, =) are indexable; = ANY and subscripts are not."
  - q: "Which question can a `jsonb_path_ops` GIN index NOT answer?"
    options: ["attrs @> '{\"color\": \"mavi\"}'", "attrs ? 'discount'", "attrs @? '$.stock ? (@ > 10)'"]
    answer: 1
    why: "jsonb_path_ops hashes values together with their paths, so there is no entry for a key on its own; key existence needs jsonb_ops."
  - q: "Why is BRIN a good fit for `created_at` on a 500 GB append-only events table?"
    options: ["It finds single rows faster than a B-tree", "Rows arrive in time order, so each block range covers a narrow time span, and one summary per range keeps the index tiny", "It supports ORDER BY created_at LIMIT 10"]
    answer: 1
    why: "BRIN works through correlation with the physical order; it is lossy and gives no ordering, but costs almost nothing to store and maintain."
  - q: "PG 18, index on `(tenant_id, created_at)`, query `WHERE created_at >= now() - interval '1 hour'`. When does skip scan make this efficient?"
    options: ["Always, regardless of data", "When tenant_id has few distinct values", "Only if the query also uses ORDER BY tenant_id"]
    answer: 1
    why: "Skip scan does one index search per distinct tenant_id; the docs note it's chosen when there are so few distinct leading values that most of the index can be skipped."
further_reading:
  - title: "PostgreSQL docs: Index Types (GIN, BRIN)"
    url: "https://www.postgresql.org/docs/current/indexes-types.html"
  - title: "PostgreSQL docs: GIN Indexes (built-in operator classes, fast update)"
    url: "https://www.postgresql.org/docs/current/gin.html"
  - title: "PostgreSQL docs: jsonb Indexing (jsonb_ops vs jsonb_path_ops)"
    url: "https://www.postgresql.org/docs/current/datatype-json.html#JSON-INDEXING"
  - title: "PostgreSQL docs: Tables and Indexes for text search"
    url: "https://www.postgresql.org/docs/current/textsearch-tables.html"
  - title: "PostgreSQL docs: pg_trgm"
    url: "https://www.postgresql.org/docs/current/pgtrgm.html"
  - title: "PostgreSQL docs: BRIN Indexes"
    url: "https://www.postgresql.org/docs/current/brin.html"
  - title: "PostgreSQL docs: Multicolumn Indexes (skip scan)"
    url: "https://www.postgresql.org/docs/current/indexes-multicolumn.html"
---

## Concept

A B-tree sorts whole values. Three other tools cover what sorting can't:

- **GIN** (Generalized Inverted Index) breaks each value into parts and indexes the parts:
  the keys and values of a `jsonb` document, the elements of an array, the stems of a
  `tsvector`, the trigrams of a string (with `pg_trgm`). For each part it keeps the list
  of rows containing it, so "which rows contain X and Y?" is a lookup and an
  intersection. It serves only its operator class's operators (`@>`, `?`, `&&`, `@@`,
  trigram `LIKE`), and is always read through a bitmap scan.
- **BRIN** (Block Range INdex) stores one small summary, usually min and max, per range of
  table pages. A query skips every range whose summary rules it out. It is tiny and
  cheap, and it only works when a column follows the physical row order, which
  append-only time series do by nature.
- **Skip scan** (PostgreSQL 18) isn't a new index type: a B-tree on `(a, b)` can serve a
  condition on `b` alone by searching once per distinct value of `a`. Before 18, such a
  query could only read the whole index or the whole table.

## Coming from C/C++/Java/Python/JS

- **Lucene / Elasticsearch:** GIN is the same idea, an inverted index from term to
  postings. With `tsvector` and the `turkish` configuration you get stemmed full-text
  search inside your transactions, with no second system to keep in sync.
- **Java/Go:** a GIN index is a `Map<String, Set<Long>>` (Go: `map[string][]int64`) from
  element to row ids, maintained on every write. A `jsonb_path_ops` index keys that map
  by a hash of path and value.
- **Parquet or columnar stores:** BRIN is the per-row-group min/max statistics that let a
  reader skip whole chunks, applied to a heap table.
- **Python:** skip scan is a loop of `bisect` calls over a list sorted by
  `(sensor_id, at)`: for each distinct sensor, jump to `(sensor, from)` and read until
  `to`, instead of reading the whole list.

## How it works

GIN is a B-tree of keys whose leaves hold row lists (posting lists, or posting trees
when a key is very common). A single row can add dozens of entries, which makes writes
expensive; by default (`fastupdate`) new entries go to a pending list first and are
merged in bulk later, so searches also scan that list until it's flushed. GIN can't
return rows in any order and can't do index-only scans.

BRIN has a range map and one summary tuple per range of `pages_per_range` pages. A scan
returns every page of every range whose summary might match, and the executor rechecks
each row: always lossy. Pages appended after the index was built belong to unsummarized
ranges until `VACUUM` (or autovacuum, or `brin_summarize_new_values()`) summarizes them;
the `autosummarize` storage parameter is off by default. Besides `minmax`, there are
`minmax-multi`, `bloom` and `inclusion` operator classes for data with outliers or for
equality-heavy queries.

Skip scan, in the docs' words, generates "a dynamic equality constraint internally,
that matches every possible value in an index column". The planner chooses it when it
expects so few distinct leading values that most leaf pages can be skipped. It is
visible only as `Index Searches` greater than 1 in `EXPLAIN ANALYZE`.

## Design notes

- **jsonb:** GIN with `jsonb_path_ops` for `@>` filters on varied keys. A key that every
  query filters on deserves a B-tree expression index or, better, its own column (the
  JSON module's indexing lesson); GIN can't do ranges or ordering on it.
- **Arrays:** GIN, and write the query with `@>` or `&&`.
- **Text:** GIN on a stored, generated `tsvector` column for words; GIN with
  `gin_trgm_ops` for substrings and typos. GiST trigram indexes also support
  nearest-neighbour ordering (`ORDER BY name <-> 'query' LIMIT 10`), which GIN can't.
- **Time series:** BRIN on the timestamp of an append-only table, with `pages_per_range`
  tuned to how narrow your typical query window is. Keep a B-tree for point lookups.
- **Skip scan** relaxes the column-order rule from the previous lesson for low-cardinality
  leading columns; it doesn't remove it. Check `Index Searches` and buffers before you
  drop the second index you thought you needed.

Turkish text needs one check per tool: the `turkish` FTS configuration stems Turkish
suffixes but lower-cases with the database's rules (here C.UTF-8, which maps I to i).
`pg_trgm` lower-cases the same way and takes its idea of a letter from the database's
`LC_CTYPE`, as the last query of the second example and the pitfall show. The search
module covers both in depth.
