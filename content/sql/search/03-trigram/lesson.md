---
id: sql.search.trigram
title: "pg_trgm similarity, ICU collations and case-insensitive matching, unaccent"
track: sql
module: search
order: 3
level: advanced
minutes: 15
since: pg18
prerequisites: [sql.search.turkish, sql.indexes.gin-brgin]
compare_with: [sql.search.fts, sql.querying.patterns, sql.constraints.keys, go.basics.strings-runes, fullstack.tracelite.step-12-search]
tags: [pg_trgm, similarity, unaccent, icu, nondeterministic-collation, case-insensitive, turkish]
summary: >
  pg_trgm scores strings by shared three-letter pieces: it forgives typos, not ş
  typed as s. An unaccent search key in an IMMUTABLE wrapper folds those away
  (merging ı and i), and nondeterministic ICU collations make =, LIKE and UNIQUE
  case-insensitive with Turkish rules.
examples:
  - id: similarity
    file: examples/01_similarity.sql
    title: "Trigrams, similarity, % and <->, on Turkish city names"
    runtime: sql
    check: exact
    annotations:
      - lines: "2"
        note: >
          pg_trgm is a trusted extension, so the non-superuser playground role
          may create it; the rollback removes it again.
      - lines: "3"
        note: >
          Each word is lowercased, padded with two spaces in front and one
          behind, and cut into three-character pieces. The six entries for
          ığdır are its six trigrams; those containing non-ASCII letters print
          as hex codes. Which characters count as letters depends on the
          database's LC_CTYPE (see the pitfalls): with LC_CTYPE C, this
          database used to return only four trigrams, around d and r.
      - lines: "11-13"
        note: >
          similarity is shared trigrams divided by all distinct trigrams of
          both strings: Antlya and Antalya share 5 of 10. % compares it with
          pg_trgm.similarity_threshold (default 0.3), and a trigram index can
          serve %.
      - lines: "16-20"
        note: >
          <-> is the distance, 1 - similarity. ORDER BY distance LIMIT 1 is a
          "did you mean" lookup; name breaks ties. A GiST trigram index can
          return rows in this order directly, GIN can't.
      - lines: "18-19"
        note: >
          IZMIR finds İzmir at 1.00, because pg_trgm lowercases with the
          database's rules (C.UTF-8 maps I and İ to i). Turkish letters are
          another matter: sanliurfa reaches only 0.25, below the % threshold;
          igdir lands on İzmir (0.20) rather than Iğdır; ızmır, which is what
          Turkish lowercasing makes of IZMIR, shares almost nothing with İzmir.
  - id: unaccent-index
    file: examples/02_unaccent_index.sql
    title: "A folded search key: unaccent in an IMMUTABLE wrapper, under a trigram index"
    runtime: sql
    check: exact
    annotations:
      - lines: "3"
        note: >
          unaccent replaces characters by a rules table (unaccent.rules): ş to
          s, ı to i, İ to I, ğ to g. In Turkish these are letters of their own,
          not accented ones; the table doesn't care, which is exactly what
          input typed on an English keyboard needs.
      - lines: "7-9"
        note: >
          unaccent is STABLE and index expressions need IMMUTABLE, hence the
          wrapper; IMMUTABLE is our promise to reindex if unaccent.rules ever
          changes. The RETURN body (SQL-standard, PG 14) is parsed now and
          records its dependencies. A quoted body would be parsed at run time,
          and since PG 17 CREATE INDEX, REINDEX and ANALYZE run with a safe
          search_path: tested here, such a wrapper fails with 'text search
          dictionary "unaccent" does not exist'.
      - lines: "15-17"
        note: >
          10,000 generated village names give the planner a reason to use the
          index. The index stores the trigrams of search_key(name): every I
          variant (I, ı, İ, i) becomes i, and ş, ğ, ç, ö, ü lose their marks.
      - lines: "19-25"
        note: >
          Every ASCII spelling now finds its city at 1.00, and ızmır meets
          İzmir. Istambul still scores 0.50: folding fixes letters, trigrams
          fix typos, and the two combine.
      - lines: "28-29"
        note: >
          search_key('SANLIURFA') is computed once at plan time, and
          search_key(name) is exactly the indexed expression, so the GIN index
          answers %. A query spelled lower(unaccent(name)) wouldn't match it.
      - lines: "32-33"
        note: >
          The price: kır (countryside) and kir (dirt), ılık (lukewarm) and ilik
          (bone marrow) share a key. Use the key to find candidates; display,
          rank and compare with the original text.
      - lines: "36"
        note: >
          Indexing unaccent directly fails: functions in an index expression
          must be IMMUTABLE (SQLSTATE 42P17).
  - id: icu-collations
    file: examples/03_icu_collations.sql
    title: "Nondeterministic ICU collations: Turkish case-insensitive =, LIKE and UNIQUE"
    runtime: sql
    check: exact
    annotations:
      - lines: "2-5"
        note: >
          ks-level2 compares letters and accents but ignores case; ks-level1
          ignores accents too. tr uses the Turkish case pairs (I/ı, İ/i) and
          treats ç, ğ, ı, ö, ş and ü as letters of their own; und, the root
          locale, uses Unicode's defaults. deterministic = false is what allows
          different strings to compare equal.
      - lines: "7-13"
        note: >
          tr_ci pairs ISPARTA with ısparta and İZMİR with izmir, but not IZMIR
          with izmir. und_ci does the opposite: I pairs with i, and İ counts as
          I with a dot accent, equal to i only at level 1. Çeşme equals cesme
          only under und_ai, because Turkish treats ç and ş as letters at every
          level. Iğdır never equals igdir: even the root collation keeps ı
          apart. No collation is the folded key of the previous example.
      - lines: "16"
        note: >
          On the column, the collation makes every comparison Turkish and
          case-insensitive, with no lower() in the queries, and the primary key
          index follows it. citext would apply the database's lower() instead,
          which maps I to i.
      - lines: "19-20"
        note: >
          IŞIK finds ışık. LIKE with a nondeterministic collation is new in PG
          18; ILIKE, SIMILAR TO and regular expressions still refuse such
          collations (ILIKE fails with SQLSTATE 0A000).
      - lines: "22"
        note: >
          The primary key compares with the column's collation, so DENİZ is a
          duplicate of Deniz: case variants can't register as separate users.
use_cases:
  - "\"Did you mean\": the nearest city, product or customer name to a misspelled query (ORDER BY name <-> $1 LIMIT 5)"
  - "Search boxes used from English keyboards or phones without Turkish letters: igdir, sanliurfa, IZMIR"
  - "Case-insensitive identities with Turkish rules: usernames, tags, city names in a UNIQUE column"
  - "Substring search (LIKE '%fragment%') on titles and codes, with a trigram index"
strengths:
  - "Typo tolerance with no external engine: similarity, % and <-> work per row and are indexable with GIN or GiST"
  - "unaccent folds Turkish letters that Unicode normalization keeps: ı has no decomposition, so Python's NFKD trick leaves it alone; unaccent maps it to i"
  - "A nondeterministic collation moves case-insensitivity into the schema: =, LIKE (PG 18), UNIQUE and the index agree, and no query can forget a lower()"
weaknesses:
  - "Trigram similarity knows nothing about language: ş and s, ı and i are simply different characters, so ASCII-typed Turkish scores low without folding"
  - "unaccent is STABLE: indexing it needs an IMMUTABLE wrapper, a promise Postgres doesn't check"
  - "Folding merges distinct Turkish words (kır and kir)"
  - "Nondeterministic collations cost performance, rule out B-tree deduplication, and don't support ILIKE, SIMILAR TO or regular expressions"
pitfalls:
  - title: "pg_trgm on a database with LC_CTYPE C"
    detail: "pg_trgm decides which characters are letters through the database's LC_CTYPE, fixed at CREATE DATABASE. Before StackLab's playground database was recreated, it had LC_CTYPE C, and show_trgm('ığdır') returned only {\"  d\",\"  r\",\" d \",\" r \"}: ı and ğ counted as separators, weakening similarity and trigram index lookups on Turkish words. Run show_trgm('ığdır') on a new server: six entries is right. Fixing it means a new database; scripts/fix-playground-ctype.sql does that for the playground."
  - title: "Indexing unaccent() directly"
    detail: "unaccent is STABLE, so CREATE INDEX fails with 'functions in index expression must be marked IMMUTABLE'. Wrap it, and use the wrapper in both the index and the queries."
    bad: "CREATE INDEX ON places USING gin (lower(unaccent(name)) gin_trgm_ops);"
    good: "CREATE INDEX ON places USING gin (search_key(name) gin_trgm_ops);"
  - title: "A wrapper whose body is a string"
    detail: "Since PG 17, CREATE INDEX, REINDEX, ANALYZE and VACUUM run with a safe search_path, so a quoted body can't find an unaccent installed in another schema: 'text search dictionary \"unaccent\" does not exist'. Use a RETURN body, which is parsed when the function is created, or schema-qualify the function and the dictionary."
    bad: "CREATE FUNCTION search_key(text) RETURNS text LANGUAGE sql IMMUTABLE AS 'SELECT lower(unaccent(''unaccent''::regdictionary, $1))';"
    good: "CREATE FUNCTION search_key(text) RETURNS text LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT RETURN lower(unaccent('unaccent'::regdictionary, $1));"
  - title: "Expecting an accent-insensitive collation to fold Turkish letters"
    detail: "und-u-ks-level1 equates Çeşme with cesme but keeps ı apart from i; tr-u-ks-level1 equates neither, because in Turkish they are letters. For users typing without Turkish letters, compare folded keys instead."
  - title: "ILIKE on a nondeterministic column"
    detail: "ILIKE doesn't support nondeterministic collations and fails with SQLSTATE 0A000. Since PG 18, plain LIKE does, and it is already case-insensitive under such a collation."
    bad: "SELECT username FROM users WHERE username ILIKE 'den%';"
    good: "SELECT username FROM users WHERE username LIKE 'den%';"
quiz:
  - q: "Why does sanliurfa score only 0.25 against Şanlıurfa?"
    options: ["pg_trgm is case-sensitive", "ş and ı are different characters from s and i, so most trigrams differ", "The similarity threshold is set too low"]
    answer: 1
    why: "pg_trgm lowercases but doesn't fold letters; folding both sides with unaccent makes them identical."
  - q: "CREATE INDEX ... USING gin (lower(unaccent(name)) gin_trgm_ops) fails. Why?"
    options: ["unaccent is STABLE, and index expressions need IMMUTABLE functions", "GIN can't index text", "Only a superuser may index unaccent"]
    answer: 0
    why: "unaccent depends on a dictionary that could change. An IMMUTABLE wrapper is the accepted workaround, with the duty to reindex if the rules change."
  - q: "Under a nondeterministic collation with locale tr-u-ks-level2, which pair compares equal?"
    options: ["'IZMIR' and 'izmir'", "'ISPARTA' and 'ısparta'", "'Çeşme' and 'cesme'"]
    answer: 1
    why: "Turkish case pairs are I/ı and İ/i; level 2 still distinguishes accents, and in Turkish ç and ş are separate letters anyway."
  - q: "username text COLLATE tr_ci (nondeterministic). What does WHERE username ILIKE 'den%' do on PG 18?"
    options: ["Matches Deniz", "Raises an error: ILIKE doesn't support nondeterministic collations", "Matches nothing"]
    answer: 1
    why: "PG 18 added LIKE support for nondeterministic collations, not ILIKE. Use LIKE, which is case-insensitive under this collation."
further_reading:
  - title: "PostgreSQL docs: pg_trgm"
    url: "https://www.postgresql.org/docs/current/pgtrgm.html"
  - title: "PostgreSQL docs: unaccent"
    url: "https://www.postgresql.org/docs/current/unaccent.html"
  - title: "PostgreSQL docs: Nondeterministic Collations"
    url: "https://www.postgresql.org/docs/current/collation.html#COLLATION-NONDETERMINISTIC"
  - title: "PostgreSQL docs: Pattern Matching (LIKE with nondeterministic collations)"
    url: "https://www.postgresql.org/docs/current/functions-matching.html"
  - title: "PostgreSQL docs: CREATE FUNCTION (SQL-standard function bodies)"
    url: "https://www.postgresql.org/docs/current/sql-createfunction.html"
  - title: "PostgreSQL docs: citext"
    url: "https://www.postgresql.org/docs/current/citext.html"
---

## Concept

Full-text search matches words. People also expect a search box to forgive two other
things, and Postgres has a tool for each, plus one for case:

- **Typos:** `pg_trgm` cuts strings into trigrams and measures their overlap.
  `similarity()` gives a score, `%` tests it against a threshold, `<->` is the distance
  for "did you mean" ordering, and GIN or GiST trigram indexes serve `%`, `LIKE`, `ILIKE`
  and regular expressions.
- **Letters typed differently:** `unaccent` maps characters through a rules table, so
  *sanliurfa* and *Şanlıurfa* get the same key. It is STABLE, so indexing it takes an
  IMMUTABLE wrapper.
- **Case:** a nondeterministic ICU collation such as `tr-u-ks-level2` makes `=`, `LIKE`
  (PG 18), `UNIQUE` and indexes ignore case, with Turkish case pairs.

The Turkish angle is that people write *Iğdır* at least three ways: *Iğdır*, *IĞDIR*
and *igdir*. A collation handles case, trigrams handle typos, and only a folded key
handles a keyboard without Turkish letters. `pg_trgm` itself lowercases with the
database's rules (`C.UTF-8` here, I to i), and decides what counts as a letter through
the database's `LC_CTYPE`.

## Coming from C/C++/Java/Python/JS

- **Python:** `similarity` is in the spirit of `difflib.SequenceMatcher.ratio()`, but
  computed next to the data and indexable. The usual accent strip,
  `unicodedata.normalize('NFKD', s)` followed by dropping combining marks, turns ş into s
  but leaves ı alone (ı has no decomposition); `unaccent`'s table maps it to i.
- **Java:** a `Collator` for `Locale.forLanguageTag("tr")` with strength `SECONDARY` is
  `tr-u-ks-level2`; `PRIMARY` is level 1. Postgres applies the same idea (with ICU's
  rules) to `=`, `UNIQUE` and indexes rather than only to `compare()`.
- **JS:** `new Intl.Collator('tr', { sensitivity: 'accent' })` corresponds to
  `tr-u-ks-level2`, and `sensitivity: 'base'` to level 1.
- **Go:** `golang.org/x/text/collate` with `language.Turkish` and `collate.IgnoreCase`
  gives the same comparisons in application code.

## Design notes

Pick the tool by the job:

- **Display and sort:** the original text, ordered with `COLLATE "tr-x-icu"`.
- **Identity and uniqueness** (usernames, tags): a nondeterministic Turkish collation on
  the column. The citext docs themselves recommend nondeterministic collations over
  citext, which lowercases with the database's rules and so pairs I with i.
- **Finding:** a folded key (`lower(unaccent(...))` in an IMMUTABLE wrapper) with a
  trigram index, matched with `%` or `LIKE`. Store it as a generated column
  (`name_key text GENERATED ALWAYS AS (search_key(name)) STORED`) if many queries use it.
- **Words in prose:** full-text search with the `turkish` configuration. The pg_trgm docs
  suggest combining the two: collect the words of your documents with
  `ts_stat('SELECT to_tsvector(''simple'', body) FROM ...')`, index that word list with
  trigrams, and offer the nearest word when a search finds nothing.

**GIN or GiST.** Both serve filtering (`%`, `LIKE`, `ILIKE`, regex). Choose GiST when
you need `ORDER BY name <-> $1 LIMIT n` served from the index: the docs note that GIN
can't return rows by distance.

**Thresholds are per session.** `%` uses `pg_trgm.similarity_threshold` (0.3). Short
strings have few trigrams, so one typo moves their score a lot; test the threshold on
your real names, or order by `<->` with a `LIMIT` instead of filtering with `%`.

**casefold() isn't the Turkish answer either.** PG 18's `casefold()` follows the
collation: under this database's `C.UTF-8` it leaves İ as it is (`dİyarbakir`), with
`COLLATE "tr-x-icu"` it gives `diyarbakır`. For Turkish text, the collation decides.

**Letters come from LC_CTYPE, not the collation.** The pg_trgm docs don't say how it
finds words; the source does. Its letter test (contrib/pg_trgm/trgm.h, macro ISWORDCHR)
calls `t_isalnum_with_len` (src/backend/tsearch/ts_locale.c), which asks the C library
under the database's LC_CTYPE, whatever the collation provider; when LC_CTYPE is C it
tests single bytes, so every non-ASCII letter is a separator. Observed on PostgreSQL 18.7
here: `show_trgm('ığdır')` has six trigrams under today's LC_CTYPE 'English_United
States.1252', and had four, only those of d and r, under the playground's earlier
LC_CTYPE 'C' (the pitfall). What PostgreSQL 18 moved to the collation provider is
lower-casing: its release notes name pg_trgm under the change to full text search, and
contrib/pg_trgm/trgm_op.c now lower-cases with `str_tolower` and the default collation
where 17 used libc. On another server, check `datctype` in `pg_database`.
