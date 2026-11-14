---
id: fullstack.tracelite.step-12-search
title: "Step 12: Search across requirements (full-text + trigram)"
track: fullstack
module: tracelite
order: 12
level: advanced
minutes: 15
since: pg18
prerequisites: [fullstack.tracelite.step-11-jobs, sql.search.fts, sql.search.turkish, sql.search.trigram]
compare_with: [sql.indexes.gin-brgin, sql.indexes.stats-pagination, go.services.pagination-streaming, go.postgres.querying]
tags: [tracelite, search, full-text-search, turkish, pg_trgm, unaccent, ranking, keyset-pagination, generic-plan, gin]
summary: >
  One search box, three matchers: a code prefix, Turkish words (stems or prefixes,
  lowered with Turkish rules on both sides) and a folded trigram key for typos and
  English keyboards. Results come in tiers, ranked, paged by a keyset cursor.
examples:
  - id: search-columns
    file: examples/01_search_columns.sql
    title: "Migration 005, and which matcher finds what people actually type"
    runtime: sql
    check: exact
    annotations:
      - lines: "12-13"
        note: >
          Both are trusted extensions: a role with CREATE privilege on the
          database, such as its owner, may create them without being a
          superuser (step 15 sets up that role). IF NOT EXISTS because
          another application may already have installed them in this
          database.
      - lines: "14-16"
        note: >
          lower() follows the database's locale; TraceLite's runs on whatever
          the customer's server has, which, unless it is a Turkish locale,
          maps I to i. Naming the ICU Turkish
          collation makes the result the same everywhere: I to ı, İ to i. The
          function is the one place where that rule lives.
      - lines: "17-19"
        note: >
          sql.search.trigram's folded key: ı, i, I and İ all become i, ş
          becomes s. It is the answer to users who type IKI ADIMLI or sifre,
          and it is used only for matching, never for display.
      - lines: "20-24"
        note: >
          The query side. plainto_tsquery ANDs the stems of the typed words;
          the second half ANDs every word as a prefix (:*). || between two
          tsqueries is OR. The words are cut at anything that isn't a letter or
          a digit, and quote_literal makes each one a single lexeme, so no
          input can be read as tsquery syntax.
      - lines: "25-28"
        note: >
          One stored vector, two spellings of every word: the Turkish stem and
          the word itself. The stemmer catches inflections (girişte is giriş);
          the words catch what it cuts too far (parola becomes parol) and make
          prefixes work. The title weighs A, the description B, for ranking.
      - lines: "29-30"
        note: >
          One GIN index per matcher: the tsvector for words, a trigram index on
          the folded title for typos. Codes need no new index: the B-tree of
          requirements_code_key already serves prefix ranges (example 3).
      - lines: "42"
        note: >
          tr_lower turned KARANLIK into karanlık before the stemmer saw it;
          lower() would have made it karanlik, a word nobody types. Stems and
          words differ in the description (kullan, kullanımı), and the stemmer
          errs on short words: gece (night) became geç (late), koyu (dark)
          became ko. The whole word is still there to match.
      - lines: "45-47"
        note: >
          A user with caps lock on a Turkish keyboard types SIFIRLAMA with
          dotless I, and tr_lower gives sıfırla, the same lexeme as in the
          column. A user who has typed only paro gets a prefix query that
          already finds parola.
      - lines: "51-52"
        note: >
          A code prefix is a range on codes: COLLATE "C" upper-cases only
          ASCII, as codes need (step 2's pitfall), and every code that starts
          with AUTH-L lies between AUTH-L and AUTH-L~, because the code CHECK
          allows only characters that sort before ~. An empty input would
          match every code, so the API refuses q="".
      - lines: "57-58"
        note: >
          Each matcher finds what the others can't: auth-l only as a code;
          parolalar through its stem (parola, matching the word parola);
          paro and e-posta as prefixes; IKI ADIMLI and the typo sifirlma only
          through the folded trigram key. Trigrams look at titles only, which
          is why e-posta (in descriptions) isn't found that way.
      - lines: "61-62"
        note: >
          The bug tr_lower prevents: lowered with the database's rules,
          SIFIRLAMA becomes sifirla, which is not in any vector, and caps-lock
          users find nothing.
  - id: ranked-pages
    file: examples/02_ranked_pages.sql
    title: "One query, three tiers, ranked, and paged by a keyset over the ranking"
    runtime: sql
    check: exact
    annotations:
      - lines: "31-52"
        note: >
          queries/search.sql, prepared exactly as pgx prepares it. Each row
          matched by any matcher gets a tier (1 code, 2 words, 3 similar) and
          an integer score; the order is tier, score descending, then code,
          which is unique per project and makes the order total.
      - lines: "40-42"
        note: >
          Scores are ts_rank or word_similarity scaled to integers. Floats in a
          cursor would have to survive JSON and a round trip to compare equal;
          integers do. Code hits all score 0 and come in code order.
      - lines: "43"
        note: >
          The typed text is turned into a prefix, a tsquery and a key once, in a
          one-row subquery, so each matcher's condition below is written once
          and compares a column with a value fixed for the whole query.
      - lines: "50"
        note: >
          The keyset, as in step 4: a row comparison with the last row's key.
          Negating the score turns its descending order into ascending, so one
          row comparison expresses tier ASC, score DESC, code ASC.
      - lines: "55"
        note: >
          A code hit comes first, then title hits by rank. AUTH-LOGIN outranks
          AUTH-RESET because its description mentions parolasıyla as well; the
          fourth row only says that a next page exists.
      - lines: "57"
        note: >
          Page 2 continues from the cursor. Three rows tie on score, since each
          mentions the word once in its description; the code breaks the tie,
          the same way on every request. Three rows are fewer than four: this
          is the last page.
      - lines: "59"
        note: >
          A typo plus an English keyboard: no code, no word matches, and the
          folded trigram key still finds the title, in tier 3.
  - id: indexes
    file: examples/03_indexes.sql
    title: "Which index serves which matcher, in the generic plans pgx ends up with"
    runtime: sql
    check: exact
    annotations:
      - lines: "26-28"
        note: >
          The range is the code matcher's form because it uses the B-tree with
          $2 unknown: both bounds are expressions of a parameter, which an index
          condition accepts.
      - lines: "31-33"
        note: >
          The same matcher as LIKE: with a literal pattern the planner derives
          the same range itself, but here the pattern is only known at run
          time, so it reads every row of the project and filters. psql shows
          the good plan; the application, after five executions, may get
          this one.
      - lines: "36-37"
        note: >
          tr_lower has been inlined, which keeps the expression usable in an
          index condition; EXPLAIN doesn't print its COLLATE. The GIN index
          answers @@ for any query value.
      - lines: "40-41"
        note: >
          <% (word_similarity above pg_trgm.word_similarity_threshold, 0.6 by
          default) is served by the trigram GIN index through its commutator
          %>, with the exact score checked on the rows it returns.
      - lines: "44-45"
        note: >
          Ordering by trigram distance is a different job: GIN can only filter,
          so this sorts the whole table. A GiST trigram index can return rows
          in distance order; TraceLite doesn't need it, since its similar tier
          is filtered first and sorted afterwards.
use_cases:
  - "A search box over records with short identifiers and prose fields: tickets, products, documents, requirements"
  - "Turkish or mixed-keyboard users: caps lock with dotless I, phones without Turkish letters, copy-pasted text"
  - "Type-ahead: prefixes of every word and of codes, answered from indexes on each keystroke after a short debounce"
  - "Search results in a paged API whose pages must stay stable while other users edit"
strengths:
  - "No search engine to run or keep in sync: the vector is a generated column, updated in the same transaction as the row"
  - "Each failure mode of typing has its own matcher, and the tier says which one fired, so the UI can show 'did you mean' for tier 3"
  - "Turkish lowering lives in one SQL function used by the column and every query, so the two sides can't drift"
  - "Integer scores and a code tiebreaker give a total order that keyset pagination can use unchanged from step 4"
weaknesses:
  - "Every page ranks all matches again; keyset pagination keeps pages stable, not cheap"
  - "ts_rank has no notion of rare words (no IDF), so a requirement that repeats a common word can outrank a better one"
  - "Two spellings per word make the vector and its GIN index roughly twice as large"
  - "The folded key merges distinct Turkish words (kır and kir), so tier 3 can return surprises; it is a fallback, ranked last"
pitfalls:
  - title: "LIKE with a parameter for prefix search"
    detail: "code LIKE $1 || '%' uses the B-tree only when the planner sees a constant pattern. pgx prepares statements, and after five executions PostgreSQL may switch to a generic plan that scans the whole project. Write the prefix as a range, which an index condition accepts with parameters."
    bad: "WHERE project_id = $1 AND code LIKE $2 || '%'"
    good: "WHERE project_id = $1 AND code >= $2 AND code < $2 || '~'  -- codes contain only characters below ~"
  - title: "Lowering the document and the query differently"
    detail: "If the column uses tr_lower and the query plain lower() (or Go's strings.ToLower), IŞIK becomes ışık in the column and işik in the query, and nothing matches. One SQL function, called on both sides."
  - title: "Building tsquery text from user input"
    detail: "to_tsquery('simple', $1 || ':*') fails with a syntax error on input like 'parola & (' and lets users write operators. Split the input into words in SQL, quote each with quote_literal, and append :* to each."
  - title: "Floats in a keyset cursor"
    detail: "A ts_rank of 0.6079271 that went through JSON and came back as a parameter may not compare equal to the value recomputed by the query, and the page boundary repeats or skips a row. Scale scores to integers before they become part of the sort key."
  - title: "Searching on unaccent alone"
    detail: "The folded key finds sifre for şifre, but also merges kır and kir and defeats the Turkish stemmer. Keep it as the last tier, after code and word matches."
quiz:
  - q: "A user with caps lock on types PAROLA SIFIRLAMA. Why does the word matcher find Parola sıfırlama only with tr_lower?"
    options: ["The turkish configuration is case-sensitive", "lower() maps I to i, giving sifirla, while the column holds sıfırla", "plainto_tsquery drops upper-case words"]
    answer: 1
    why: "The database's lowering isn't Turkish; tr_lower maps I to ı on both sides, so the lexemes agree."
  - q: "Why does the code matcher use code >= $2 AND code < $2 || '~' instead of LIKE $2 || '%'?"
    options: ["LIKE is case-sensitive", "With a parameter, the planner can't turn LIKE into an index range, and pgx's prepared statements may get a generic plan", "LIKE doesn't work with COLLATE \"C\""]
    answer: 1
    why: "B-tree prefix matching for LIKE requires a constant pattern at planning time; a range with parameters is an index condition in any plan."
  - q: "Which matcher finds IKI ADIMLI typed on an English keyboard?"
    options: ["The code prefix", "The Turkish word matcher", "The folded trigram key"]
    answer: 2
    why: "tr_lower makes it ıkı adımlı, which isn't iki adımlı; folding both sides with unaccent gives iki adimli, which the trigram index matches."
  - q: "Why are scores scaled to integers before they go into the cursor?"
    options: ["JSON can't hold floats", "So the cursor's value compares equal to the recomputed score, and pages neither repeat nor skip rows", "Integers sort faster"]
    answer: 1
    why: "The keyset condition compares the cursor with values the query computes again; integers survive the round trip exactly."
further_reading:
  - title: "PostgreSQL docs: Controlling text search (ranking, weights)"
    url: "https://www.postgresql.org/docs/current/textsearch-controls.html"
  - title: "PostgreSQL docs: pg_trgm (word_similarity, operators, index support)"
    url: "https://www.postgresql.org/docs/current/pgtrgm.html"
  - title: "PostgreSQL docs: Index types (B-tree and LIKE)"
    url: "https://www.postgresql.org/docs/current/indexes-types.html"
  - title: "PostgreSQL docs: PREPARE (custom and generic plans)"
    url: "https://www.postgresql.org/docs/current/sql-prepare.html"
  - title: "PostgreSQL docs: Row constructor comparison"
    url: "https://www.postgresql.org/docs/current/functions-comparisons.html#ROW-WISE-COMPARISON"
---

## Concept

People don't search like a `WHERE` clause. They type half a code (`auth-l`), a word in
a different form than the title uses (`parolalar`), the start of a word (`paro`), a
title with caps lock on a Turkish keyboard (`PAROLA SIFIRLAMA`), or with no Turkish
letters at all, and with typos (`sifirlma`). TraceLite's search runs three matchers
over a project's requirements and merges them into one ranked list:

1. **Code prefix** (tier 1): `upper(q)` as a range over `code`, served by the existing
   `requirements_code_key` B-tree.
2. **Words** (tier 2): a stored `tsvector` holding, for title (weight A) and description
   (B), both the Turkish stem and the whole word of every word. The query asks for all
   typed words as stems, or all of them as prefixes. Both sides are lowered with
   Turkish rules by one function, `tr_lower`. Ranked with `ts_rank`.
3. **Similar** (tier 3): `word_similarity` between folded keys (`search_key`, which maps
   ş to s and ı, I, İ to i) of the input and the title, through a trigram index. Ranked
   by the similarity.

A row matched by several matchers counts in its best tier. Within a tier, a higher
score comes first; the code breaks ties. That sort key is also the cursor.

## Coming from C/C++/Java/Python/JS

- **Elasticsearch/Lucene**: the three tiers are a `bool` query with `should` clauses
  (a `prefix` on a keyword field, a `match` on a Turkish-analyzed field, a `fuzzy`
  match), and the tiers stand in for boosts. Lucene scores with BM25 and has a Turkish
  lowercase filter; PostgreSQL's `ts_rank` has no IDF, and Turkish lowering is
  `tr_lower`. In exchange there is no index refresh and nothing to keep in sync.
- **Java/Hibernate Search, Python/django-watson**: both keep a search index next to the
  ORM. Here the index is a generated column, so a raw SQL update or the step 7 importer
  updates it too.
- **JS (Fuse.js, match-sorter)**: fuzzy search in the browser over data already
  loaded. Fine for a hundred rows; TraceLite searches on the server so that the page
  never needs all requirements.
- **Go**: `strings.ToLower` uses Unicode's default mapping (I to i), like the database's
  `lower()`; `golang.org/x/text/cases.Lower(language.Turkish)` is `tr_lower`'s
  equivalent. TraceLite lowers only in SQL, so Go never sees the question.

## How it works

`migrations/005_search.sql` is example 1's middle part. The endpoint, in `openapi.yaml`:

```yaml
  /api/projects/{projectId}/requirement-search:
    parameters: [{ $ref: "#/components/parameters/ProjectId" }]
    get:
      operationId: searchRequirements
      parameters:
        - { name: q, in: query, required: true, schema: { type: string, minLength: 1, maxLength: 100 } }
        - { $ref: "#/components/parameters/Limit" }
        - { $ref: "#/components/parameters/Cursor" }
      responses:
        "200":
          description: Best matches first
          content: { application/json: { schema: { $ref: "#/components/schemas/SearchPage" } } }
        "400": { $ref: "#/components/responses/Problem" }   # q blank or too long, bad cursor
        "404": { $ref: "#/components/responses/Problem" }
# components.schemas:
    SearchHit:
      allOf:
        - { $ref: "#/components/schemas/Requirement" }
        - type: object
          required: [match]
          properties: { match: { type: string, enum: [code, words, similar] } }
    SearchPage:
      type: object
      required: [items, nextCursor]
      properties:
        items: { type: array, items: { $ref: "#/components/schemas/SearchHit" } }
        nextCursor: { type: [string, "null"] }
```

The handler trims `q` and refuses it when blank (400; an empty prefix would match every
code), decodes the cursor into `[tier, score, code]` (step 4's format: base64url JSON,
the last row's sort key), runs `SearchRequirements` with `limit+1` and maps the tier to
`match`. The request path is the same as the list's; only the query differs.

Which index serves which matcher (example 3):

| Matcher | Condition | Index |
|---|---|---|
| code | `code >= $p AND code < $p \|\| '~'` | `requirements_code_key` (B-tree, from step 2) |
| words | `search @@ words_query($q)` | `requirements_search` (GIN on the tsvector) |
| similar | `search_key($q) <% search_key(title)` | `requirements_title_key` (GIN, `gin_trgm_ops`) |

For a project of a few hundred requirements the planner may well read the project's
rows through `requirements_code_key` and check all three conditions on each, which can
cost less than three index scans; the plan depends on the statistics. The GIN indexes start to matter in big projects, and they
are what make a search across all projects possible later.

In the browser, the search box waits 200 ms after the last keystroke, then runs a
`useInfiniteQuery` keyed `["projects", projectId, "search", q]` with
`placeholderData: keepPreviousData`, so the old results stay on screen while the new
ones load. The key sits under the project prefix, so step 9's events refresh open
results too. Each hit shows its `match` as a small label: `kod`, `kelime`, `benzer`.

## Design notes

**Three matchers, not one clever one.** A single trigram search over code, title and
description would find most things, ranked by an opaque similarity. Separate matchers
make each rule explainable ("codes match by prefix", "words match by stem or start",
"everything else by similarity") and let the UI say why a result is there. The order of
the tiers encodes what users mean: a typed code is a precise request; a fuzzy match is a
guess.

**Turkish rules in one function.** `tr_lower` names the ICU Turkish collation, so the
behavior doesn't depend on how the customer's PostgreSQL was initialized. The same
function lowers the stored vector and every query; Go code never lowers search text.
`search_key` deliberately does the opposite (it forgets Turkish letters) and is used
only where that is the point.

**Keyset over a ranking.** The cursor holds the last row's (tier, score, code). That
keeps pages stable when requirements are added between requests, and it is the same
cursor format as every other list. It doesn't make later pages cheaper: each page
ranks all matches again. A search returns tens of rows, not thousands, so that cost is
accepted; past that, a smaller `limit` on deep pages or a cap on matches would come
first.

**Not done here.** Highlighting the matched words is `ts_headline` (sql.search.fts),
called for the page's rows only. A "did you mean" for queries with no results would
offer the nearest word from `ts_stat` over the project's vectors, as the pg_trgm docs
suggest. Synonyms (şifre for parola) need a synonym dictionary file on the server, which
a migration can't install.
