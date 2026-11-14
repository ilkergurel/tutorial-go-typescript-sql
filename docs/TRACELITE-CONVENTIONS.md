# TraceLite conventions

TraceLite is the project the `fullstack.tracelite` lessons build step by step: a small
requirements-traceability app (requirements, test cases, links between them, and test
runs). Steps 1-5 set the conventions below; steps 6-15 must follow them so the lessons
read as one codebase. Read steps 1-5 (`content/fullstack/tracelite/`) before writing a
later step. Steps 1-5 were written and verified on 2026-11-13. The owner reviewed the
decisions recorded here, including those the agents made without asking (404 for
malformed ids, all-or-nothing import, refetch-on-event SSE, the Go clock for the job
queue, report files in bytea, one tsvector for stems and words, trigger-based audit),
and accepted them on 2026-11-14.

## Schema

- The canonical migration is `migrations/001_init.sql`, in goose format. It appears
  verbatim in step 2's "How it works" section; step 2 example 1 runs its projects and
  requirements tables verbatim, example 3 its trace_links and test_runs tables. Later
  steps add `002_…`, `003_…` and never edit 001.
- Tables:
  - `projects(id, name, created_at)`
  - `requirements(id, project_id, parent_id, code, title, description, status,
    created_at, updated_at)`: `parent_id` NULL means top level; `description` is
    `NOT NULL DEFAULT ''`; `status` is `draft`, `approved` or `obsolete`.
  - `test_cases(id, project_id, code, title, description, created_at, updated_at)`
  - `trace_links(project_id, requirement_id, test_case_id, created_at)`, primary key
    `(project_id, requirement_id, test_case_id)`
  - `test_runs(id, test_case_id, result, finished_at, duration_ms, notes)`: `result` is
    `passed`, `failed` or `skipped`; `duration_ms` is nullable.
- Codes are `text COLLATE "C"`, match `^[A-Z][A-Z0-9]*(-[A-Z0-9]+)*$`, have at most 40
  characters and are unique per project. Titles have 1-200 characters, counted as
  characters, not bytes.
- The same-project rule is enforced by composite foreign keys
  `(project_id, x_id) → (project_id, id)`, which need the `*_in_project UNIQUE
  (project_id, id)` constraints.
- Deletes:
  - The parent foreign key has no ON DELETE action, so a requirement with children
    can't be deleted.
  - Deleting a project cascades to its requirements, test cases and links.
  - Links cascade from both ends.
  - The `test_runs` foreign key is ON DELETE RESTRICT, so evidence survives; even the
    project cascade stops there. RESTRICT raises SQLSTATE 23001, not 23503.
- Constraint names the store maps to errors: `requirements_code_key`,
  `test_cases_code_key`, `requirements_parent_fkey`, `requirements_project_id_fkey`,
  `trace_links_requirement_fkey`, `trace_links_test_case_fkey`,
  `test_runs_test_case_fkey`, plus the `*_format`, `*_length` and `*_valid` CHECKs.
- Indexes: `requirements_parent (parent_id)`,
  `trace_links_test_case (project_id, test_case_id)`,
  `test_runs_latest (test_case_id, finished_at DESC, id DESC)`.
- Ids are `uuid DEFAULT uuidv7()` (PostgreSQL 18), not identity columns: ids appear in
  URLs and JSON, the step 7 importer can mint them, they don't reveal volume, and there
  is no 2^53 problem in JavaScript.
  - Go: the standard library's `uuid.UUID` (Go 1.27), `*uuid.UUID` when nullable.
  - Wire: lowercase UUID strings.
  - TS: branded `z.uuid().brand<"ProjectId" | "RequirementId" | "TestCaseId" |
    "TestRunId">()`.
- "Latest run" means ordered by `finished_at DESC, id DESC`.
- Coverage vocabulary:
  - `uncovered`: no linked test.
  - `failing`: some linked test's latest run failed.
  - `passing`: every linked test's latest run passed.
  - `incomplete`: anything else.
  - Compute it with `count(*) FILTER (…)`, not `bool_and`, which ignores NULLs.

## Go

- Layout: `openapi.yaml` and `sqlc.yaml` at the root; `migrations/`, `queries/`,
  `cmd/tracelite/`.
  - `internal/store`: sqlc output plus a hand-written `store.go` with `Store`,
    `WithTx(ctx, func(*Queries) error)` (implemented with `pgx.BeginFunc`, as in step
    13), one method per transactional use case, and the
    errors `ErrNotFound`, `ErrDuplicateCode`, `ErrParentNotFound`, `ErrHasChildren`,
    `ErrCycle`, `ErrHasRuns`, `ErrNotInProject`.
  - `internal/tracelite`: the service (normalization and validation, shared with the
    CSV import). Named after the domain, not the layer (go.design.layers).
  - `internal/httpapi`: wire types named exactly like the OpenAPI schemas, `writeError`,
    cursors.
- `sqlc.yaml` uses `pgx/v5` with uuid overrides on the pattern sqlc's docs show for
  `github.com/google/uuid`, pointed at the standard `uuid` package, plus
  `pg_catalog.timestamptz → time.Time` and `emit_pointers_for_null_types`. sqlc isn't
  installed, so lessons show hand-written "generated" code and say so. Step 3's queries
  were checked once with sqlc 1.31.1 (2026-11-14): inside a recursive CTE sqlc can't
  resolve a column to infer a parameter's type ("column reference "id" is ambiguous"),
  so parameters there carry an explicit cast (`@node::uuid`).
- Moving a requirement locks the project row (`FOR UPDATE OF p`), then runs the walk-up
  cycle check with `CYCLE`. Step 3 shows MoveRequirement as a fenced block in "How it
  works"; its example 3 runs the statements in two sessions.

## JSON

- `encoding/json/v2` throughout. Decode with `UnmarshalRead` +
  `RejectUnknownMembers(true)` behind a 64 KiB `MaxBytesReader`. Its member-name
  matching is case-sensitive.
- Member names are camelCase. SQL stays snake_case; the mapping lives only in Go struct
  tags.
- Every member is always present: absent is `null`, an empty list is `[]`, empty text
  is `""`.
- Times are RFC 3339 UTC with `Z`. Call `.UTC()` in Go, because pgx returns local time.
- Enum values are lowercase strings.

## HTTP API

- Everything is under `/api`. The full list is in step 4; step 1 has the OpenAPI 3.1
  excerpt (requirement paths and all component schemas).
  - `GET|POST /api/projects`, `GET /api/projects/{projectId}`
  - `GET|POST /api/projects/{projectId}/requirements`
  - `GET /api/projects/{projectId}/requirement-tree` (flat, depth-first, with `depth`)
  - `GET|PATCH|DELETE /api/requirements/{requirementId}`
  - `PUT /api/requirements/{requirementId}/parent`, body `{"parentId": … | null}`
  - `GET|POST /api/projects/{projectId}/test-cases`
  - `GET|PATCH|DELETE /api/test-cases/{testCaseId}`
  - `GET /api/projects/{projectId}/trace-links`
  - `PUT|DELETE /api/requirements/{requirementId}/test-cases/{testCaseId}`
    (idempotent, 204)
  - `GET|POST /api/test-cases/{testCaseId}/runs`
- Statuses:
  - 200 for reads and updates; 201 with `Location` for creates; 204 for deletes, link
    and unlink.
  - 400 `about:blank`: the body can't be decoded, or a query parameter (limit, cursor)
    is invalid.
  - 404 `about:blank`: the path id is malformed or unknown, including the project of a
    nested list.
  - 409 conflict: duplicate code (with an `errors` entry for `#/code`), a requirement
    with children, a test case with runs.
  - 413: body over the limit.
  - 422 validation: well-formed content that breaks a rule.
  - 500: no detail; the cause is logged.
- Errors are RFC 9457 problems, `{type, title, status, detail?, errors?: [{pointer:
  "#/field", detail}]}`, all written by one `writeError`. Types: `about:blank` (title =
  status text), `https://tracelite.example/problems/validation` (title "Validation
  failed") and `https://tracelite.example/problems/conflict` (title "Conflict").
- Pagination: `?limit=` (1-200, default 50) and `?cursor=`; responses are
  `{items, nextCursor: string | null}`. The cursor is base64url without padding of a
  JSON array holding the last item's sort key in ORDER BY order, such as
  `["AUTH-LOGIN"]`, or `["<finishedAt>", "<id>"]` for runs. The server fetches limit+1
  rows. An undecodable cursor gets 400.

## TypeScript

- `web/src/api/` holds four files:
  - `schema.d.ts`: openapi-typescript output.
  - `schemas.ts`: zod schemas declared with `satisfies
    z.ZodType<components["schemas"][X]>`; types come from `z.infer`; times use
    `z.iso.datetime()`.
  - `client.ts`: `request()` validates every response and throws `ApiError{problem}`,
    synthesizing an `about:blank` problem when the body isn't a problem.
  - `queries.ts`: query keys `["projects", id, …]`, `["requirements", id]` and
    `["test-cases", id]`; `useInfiniteQuery` with `initialPageParam: null` and
    `getNextPageParam: (p) => p.nextCursor`; mutations invalidate the project prefix.
- Dev setup: Vite proxies `/api` to 127.0.0.1:8080 and every URL is relative, so there
  is one origin.

## Runner limits met so far

- ts-browser: the iframe sandbox has no `allow-forms`, so `requestSubmit()` never fires
  the submit event. Step 5's harness dispatches the event itself and an annotation
  explains why.
- A test can't prove that a lock is needed: without the lock the race window is a few
  milliseconds and the transactions still serialize in practice. Assert the outcome and
  say so in an annotation. Step 3 example 3 shows the race and the lock with
  sql-multisession instead, which interleaves the statements deterministically.

## Steps 6-10 additions

Steps 6-10 were written and verified on 2026-11-13. Later steps follow these as well.

### Migrations

- `002_change_events.sql` (step 9): function `notify_change()` and the row-level AFTER
  INSERT/UPDATE/DELETE triggers `requirements_notify`, `trace_links_notify` and
  `test_runs_notify`. They call `pg_notify('tracelite_changes', '{"projectId": …,
  "topic": …}')`; topics are `requirements`, `trace-links` and `test-runs` (test_runs finds
  its project through test_cases). Uses goose `StatementBegin`/`StatementEnd`. It is
  step 9's fixture `fixtures/migrations/002_change_events.sql` and appears verbatim in
  step 9's "How it works". (The step 8 report query is `queries/report.sql`, step 8
  example 3.)
- `003_webhooks.sql` (step 10): `webhooks(id uuid DEFAULT uuidv7(), project_id → projects
  ON DELETE CASCADE, url, secret, events text[] DEFAULT '{test-run.failed}', created_at)`,
  constraints `webhooks_url_valid` (`^https?://`) and `webhooks_events_valid` (`events <@
  '{test-run.failed}'`, replaced when a type is added), index `webhooks_project
  (project_id)`. The secret is stored in clear (it is an HMAC key).
- Steps 6-8 add no migration. The step 7 importer stages in `CREATE TEMP TABLE
  import_rows (...) ON COMMIT DROP`, inside its transaction.
- The next migration number is 004. (003 appears verbatim in step 10's "How it works",
  004 in step 11's.)

### Endpoints

- `GET /api/projects/{projectId}/coverage` (step 6): `CoverageList {items}` of
  `RequirementCoverage {requirementId, linkedTests, passingTests, coverage}`, one per
  requirement, not paged. Store query `ProjectCoverage` in `queries/coverage.sql` (step
  2's rule, `count(*) FILTER` throughout).
- `POST /api/projects/{projectId}/requirement-import` (step 7): body `text/csv;
  charset=utf-8|windows-1254`, at most 10 MiB. 200 `ImportResult {imported}`; 422 problem
  type `https://tracelite.example/problems/import`, title "Import failed", extension
  member `rows: [{line, column (header name or null), detail}]`; 404, 413, 415.
- `GET /api/projects/{projectId}/requirement-export` (step 8): the import's CSV format,
  UTF-8 with BOM, `;`, CRLF, ordered by code; `Content-Disposition: attachment`, file
  name "<project> gereksinimleri.csv".
- `GET /api/projects/{projectId}/traceability-report` (step 8): `TraceabilityReport
  {project {name}, requirements [{code, title, depth, coverage, tests [{code, result,
  finishedAt}]}], summary {requirements, counted, failing, uncovered, incomplete,
  passing}}`, tree order, summary last; file name "<project> izlenebilirlik.json".
- `GET /api/projects/{projectId}/events` (step 9): server-sent events named
  `requirements`, `trace-links`, `test-runs` and `resync`, data `{"topic": <name>}`, no
  `id:`, `retry: 3000`, a `: ping` comment every 25 s; 404 for an unknown project.
- `GET /api/standards/{ref}` (step 10): `{ref, title, stale}`; 404 when the catalogue has
  no such clause; 503 when it is unreachable and nothing is cached. Catalogue URL from
  `TRACELITE_STANDARDS_URL`.

### CSV files and downloads (steps 7 and 8)

- The CSV format: header names `code;title;parent;status;description` in any order,
  `code` and `title` required, `parent` a code in the file or the project, empty
  `status` means `draft`. The import detects `;` or `,` from the header, strips a BOM,
  reports `line` as `csv.Reader.FieldPos(0)` (header = line 1), counts title characters
  after decoding, and is all or nothing: one transaction, the project row locked `FOR
  UPDATE`, every row staged, Go-side and SQL-side problems merged by line.
- The CSV export guards against formula injection (owner's decision, 2026-11-14):
  `excelText` puts a tab before a `title` or `description` that starts with `=`, `+`,
  `-` or `@` (OWASP's advice for Excel); csv.Writer quotes the field, and the import's
  `strings.TrimSpace` removes the tab, so export → Excel edit → import round-trips.
  Step 8's example 1 leaves the guard out for length; its Design notes show it.
- The browser labels the charset itself: a strict `TextDecoder("utf-8", {fatal: true})`
  over the first 64 KiB with `{stream: true}`; failure means `windows-1254`. The file is
  the request body, not multipart.
- Streaming responses (step 8, and any later export): check the project and read the
  query's first row before any header; write rows as they are read, with no explicit
  Flush (csv.Writer, jsontext.Encoder and the ResponseWriter pass bytes on as their
  buffers fill); after the first byte, a failure is logged and ends in
  `panic(http.ErrAbortHandler)`; the failure path never writes closing JSON tokens. JSON
  is written with `jsontext.Encoder`. Browsers download through `<a href=… download>`,
  not through `request()`.

### Real-time (step 9)

- `internal/events`: `Hub` (a map of every open stream's buffered channel (8) to its
  project; events are topic strings; a full subscriber is dropped, never waited for) and
  `Listen(ctx, dsn, hub.Publish, hub.Resync)` (one dedicated `pgx.Conn`, never the pool;
  after every connect, `LISTEN` first, then `Resync`; an undecodable payload ends the
  connection, so the reconnect resyncs).
- The SSE handler clears the write deadline with
  `ResponseController.SetWriteDeadline(time.Time{})`.
- The browser hook `useProjectEvents(projectId, connect)` invalidates `["projects",
  projectId]` on every event and on every `open`; states `connecting | live |
  reconnecting | closed` (closed: a non-200 answer, which EventSource never retries).

### Outbound calls and webhooks (step 10)

- Outbound HTTP: a per-attempt `context.WithTimeout` derived from the caller (2 s for the
  catalogue, 10 s for webhooks), an `http.Client` with a `Timeout` backstop, never
  `http.DefaultClient`. Retry only network errors, timeouts, 429 and 5xx; full jitter
  (`rand.N(100ms << attempt)`); `Retry-After` honored but never past the caller's
  deadline; 3 attempts for lookups.
- The catalogue cache: answers 24 h, 404s 10 min, a stale entry returned with
  `stale: true` when the catalogue fails. In memory, per process.
- Webhook event `test-run.failed`, published after the failed run is committed:
  `{id, type, createdAt, data {projectId, testCase, requirements, finishedAt}}`. The id
  is minted once and reused on every retry. Header `TraceLite-Signature: t=<unix>,
  v1=<hex HMAC-SHA256 of "<t>.<body>">` (the scheme of go.services.webhooks-hmac), signed
  per attempt over body bytes marshaled once. 2xx = delivered; 410 deletes the webhook;
  429, 5xx and timeouts are retried; any other 4xx stops. Receivers check a 5-minute
  window and deduplicate on `id`.
- Step 11 moves webhook delivery into the job table: a delivery row inserted in the same
  transaction as the failed run (an outbox), retried by `run_after` on a
  minutes-to-hours schedule, with the signing and status rules above unchanged.

### UI and client (step 6)

- Routes: `/projects/:projectId` (tree and matrix) and
  `/projects/:projectId/requirements/:requirementId` (detail). Route params become ids
  only through the branded schema's `safeParse`; a bad id renders "bulunamadı" without a
  request.
- Feature folders: `web/src/features/requirements/` (`tree.ts` with `nest` and
  `tallies`, `TreeView.tsx`, `RequirementPage.tsx`), `web/src/features/matrix/`
  (`TraceMatrix.tsx`, `useToggleLink.ts`), `web/src/features/import/`.
- Query keys added: `[…, "coverage"]`, `[…, "test-cases", "all"]` and
  `[…, "trace-links"]`; the matrix loads every page in one query with `fetchAll`.
- The tree nests the flat depth-first list in one pass and keeps the server's order. Its
  roll-up: a requirement with children and no tests of its own adds nothing; every other
  requirement adds its own coverage; a parent shows the worst value in its subtree, in
  the order failing, uncovered, incomplete, passing. The report summary (step 8) counts
  the same way.
- UI labels are Turkish: coverage `başarısız`, `testsiz`, `eksik`, `geçiyor`; status
  `taslak`, `onaylı`, `geçersiz`. Code and wire values stay English.
- Forms are uncontrolled, keyed by `${id}@${updatedAt}`, and PATCH only the members that
  changed. Matrix cells are buttons with `aria-label="<requirement> × <test case>"` and
  `aria-pressed`, toggled optimistically; coverage is never computed in the browser.

### Lesson data (fakes and seeds of steps 6-10)

- Project Ödeme, id `01937b6e-0000-7000-8000-00000000000a` in the TS fakes. Tree: AUTH >
  AUTH-LOGIN > (AUTH-LOGIN-2FA, AUTH-LOGIN-LOCKOUT); AUTH > AUTH-RESET; EXPORT >
  EXPORT-CSV. Test cases TC-1 to TC-4; links: 2FA to TC-1 and TC-2, LOCKOUT to TC-1 and
  TC-4, RESET to TC-3. Latest runs: TC-1 and TC-2 passed, TC-3 failed, TC-4 never ran.
  Coverage: 2FA passing, LOCKOUT incomplete, RESET failing, EXPORT-CSV uncovered.

### Runner limits met in steps 6-10

- ts-browser: a finished TanStack Query mutation keeps a garbage-collection timer (5
  minutes by default), so the run hits its timeout unless the client sets
  `mutations: { gcTime: Infinity }` (queries likewise).
- ts-browser: no network, so EventSource is simulated with an `EventTarget` behind a
  small `Source` interface; the hook takes a `connect` function (step 9).
- pgx `CopyFrom`: when the `CopyFromSource` fails, pgx returns the server's "COPY from
  stdin failed" `PgError` (SQLSTATE 57014), whose message repeats the source error's
  text but not its type; keep the source error yourself (step 7). The sql runtimes
  refuse `COPY FROM STDIN` (sqlrun/guard.go), so COPY lessons use pgx.
- PostgreSQL computed the target list below the Sort in a tree query with `ORDER BY`
  (observed on 18.x), so a per-row SQL error arrived before the first row. Step 8's
  mid-stream failure uses `0 / (1500 - n)` over `generate_series` with no ORDER BY, so
  it fails after 1,499 rows have been sent.

## Steps 11-15 additions

Steps 11-15 were written and verified on 2026-11-13. They complete the series.

### Migrations

- `004_jobs.sql` (step 11): `jobs(id uuid DEFAULT uuidv7(), project_id → projects ON DELETE
  CASCADE, kind, payload jsonb DEFAULT '{}', state DEFAULT 'queued', attempts DEFAULT 0,
  run_after DEFAULT now(), lease_token uuid, last_error DEFAULT '', result bytea,
  created_at, finished_at)`; constraints `jobs_kind_valid` (`report`, `webhook`) and
  `jobs_state_valid` (`queued`, `running`, `done`, `failed`); indexes `jobs_due (run_after,
  id) WHERE state IN ('queued', 'running')` and `jobs_project (project_id, created_at DESC)`.
- `005_search.sql` (step 12): `CREATE EXTENSION IF NOT EXISTS pg_trgm` and `unaccent`;
  functions `tr_lower(text)` (`lower($1 COLLATE "tr-x-icu")`), `search_key(text)`
  (`lower(unaccent('unaccent'::regdictionary, $1))`), both IMMUTABLE with a `RETURN` body,
  and `words_query(text)` (STABLE: `plainto_tsquery('turkish', tr_lower(q))` OR every word
  as a `simple` prefix, words split on `[^[:alnum:]]+` and quoted with `quote_literal`);
  column `requirements.search tsvector GENERATED ALWAYS AS (…) STORED`: turkish stems ||
  simple words of `tr_lower(title)` weighted A, of `tr_lower(description)` weighted B;
  indexes `requirements_search` (GIN) and `requirements_title_key` (GIN,
  `search_key(title) gin_trgm_ops`).
- `006_audit.sql` (step 13): `audit_log(id bigint identity, at DEFAULT now(), xact xid8
  DEFAULT pg_current_xact_id(), actor, action DEFAULT '', project_id uuid (no foreign
  key), table_name, row_id uuid, op, changes jsonb)`, indexes `audit_log_row (row_id, id)`
  and `audit_log_project (project_id, id)`; function `audit_change()` (TG_ARGV[0] names the
  row's id column); AFTER ROW triggers `requirements_audit`, `test_cases_audit`,
  `trace_links_audit` (`requirement_id`). `changes` is the row for INSERT/DELETE and
  `{column: [old, new]}` for UPDATE, without `created_at`/`updated_at`; an update that
  changed nothing else writes no row. `webhooks` (its secret) and `test_runs` (evidence
  already) are not audited.
- The next migration number is 007.

### Endpoints

- `POST /api/projects/{projectId}/traceability-report-jobs` (step 11): 202, `Location:
  /api/jobs/{jobId}`, body `Job {id, kind, state, attempts, lastError, createdAt,
  finishedAt}`; 404 for an unknown project. Step 8's streaming report endpoint stays.
- `GET /api/jobs/{jobId}` (step 11): `Job`. `GET /api/jobs/{jobId}/file`: the finished
  report through `http.ServeContent` (Content-Length, Range → 206), step 8's file name;
  404 for an unknown job or one that isn't a report; 409 conflict problem until done.
- `GET /api/projects/{projectId}/requirement-search?q=&limit=&cursor=` (step 12):
  `SearchPage {items: SearchHit[], nextCursor}`, `SearchHit` = `Requirement` + `match:
  code | words | similar`; `q` trimmed, 1-100 characters, else 400; cursor `[tier, score,
  code]`.
- `GET /api/requirements/{requirementId}/history?limit=&cursor=` (step 13): `{items: [{at,
  actor, action, table, op, changes}], nextCursor}`, newest first, keyset by `audit_log.id`;
  `changes` keeps column names (snake_case) because it describes the table.
- `GET /api/health` (step 15): 200 `{status: "ok", version, database: "ok"}`, 503 when the
  database doesn't answer.
- Browser: the report job is polled with a `refetchInterval` function (1 s while
  `queued`/`running`), then downloaded through a plain link. Search waits 200 ms after the
  last keystroke and uses `useInfiniteQuery` keyed `["projects", id, "search", q]` with
  `placeholderData: keepPreviousData`; `match` labels are `kod`, `kelime`, `benzer`.

### Jobs (step 11)

- `internal/jobs`: `Queue` (`Enqueue(ctx, tx, …)` inside the caller's transaction,
  `Claim`, `Extend`, `Finish`, `Release`) and `Pool`; handlers registered by kind.
- Claim is one statement: `UPDATE jobs SET state = 'running', attempts = attempts + 1,
  run_after = <now + lease>, lease_token = <new uuid> WHERE id = (SELECT id … WHERE state IN
  ('queued', 'running') AND run_after <= <now> ORDER BY run_after, id LIMIT 1 FOR UPDATE
  SKIP LOCKED) RETURNING …`. `run_after` is the lease's end while running, so a dead
  worker's job comes back by itself. Every later update is fenced with `AND lease_token =
  $2`; zero rows is `ErrLeaseLost`.
- Time comes from Go (`now func() time.Time`, `time.Now` in main). Lease 2 min, heartbeat
  every lease/3 (a lost lease cancels the handler), 4 workers, poll every second, 20 s
  drain. Outcomes are written with `context.WithoutCancel` plus a 5 s timeout. Workers
  claim on the job context (cancelled at the drain deadline), not on the shutdown signal,
  so a claim already sent when SIGTERM arrives completes instead of stranding a lease.
- Outcomes: nil → `done` (a report's file stored in `result` by the same fenced UPDATE); a
  `Permanent` error → `failed`; any other error → `queued` with `run_after = now +
  schedule[attempts-1]` plus up to 10% jitter, schedule 1 min, 5 min, 30 min, 2 h, 6 h,
  `failed` after the sixth attempt; interrupted by shutdown → `Release` (attempt given
  back, `run_after = now`, `last_error 'interrupted by shutdown'`).
- Webhooks: one job per subscribed webhook, inserted in the failed run's transaction;
  payload `{webhookId, body}` with the event JSON as a string, marshaled once; one POST
  per attempt, signed per attempt; 2xx done, 410 deletes the webhook (done), other 4xx
  `Permanent`, 429/5xx/timeouts retried. Step 10's in-memory retry loop is gone.
- Done and failed jobs older than 30 days are deleted once a day, report files included.

### Search (step 12)

- Three matchers, best tier wins: 1 code prefix (`code >= upper(q COLLATE "C") AND code <
  upper(q COLLATE "C") || '~'`, a range because `LIKE $1 || '%'` loses the index in a
  generic plan), 2 words (`search @@ words_query(q)`), 3 similar (`search_key(q) <%
  search_key(title)`). Order `tier, score DESC, code`; scores are integers: 0 for codes,
  `1e6 × ts_rank`, `1e6 × word_similarity`.
- Turkish lowering happens only in SQL, through `tr_lower`, for the column and every
  query; Go never lowers search text.

### Audit (step 13)

- `Store.WithTx` first runs `SELECT set_config('tracelite.actor', $1, true)` with
  `ActorFrom(ctx)` (set by a middleware: `TRACELITE_OWNER` in a single-user install, the
  signed-in user once there is authentication). A use case may set `tracelite.action`
  (`move`, `import`) the same way. Never a session-level `SET`.
- The trigger's actor is `coalesce(nullif(current_setting('tracelite.actor', true), ''),
  'db:' || session_user)`; a `db:` actor in the log means a write outside `WithTx`.

### Testing (step 14)

- Layout: `internal/httpapi/*_integration_test.go` (`//go:build integration`,
  `TEST_DATABASE_URL`, run with `go test -tags integration -count=1 ./...`),
  `internal/httpapi/contract_test.go` (responses checked against `openapi.yaml`),
  `web/src/**/*.test.tsx` (Vitest, jsdom, MSW handlers in `web/src/test/`),
  `web/e2e/*.spec.ts` (Playwright, `channel: "msedge"`, `webServer` runs the built binary
  on 127.0.0.1:8081, not `go run`).
- Database tests isolate by data: every test creates its own project and may use
  `t.Parallel`; `TestMain` creates the test database and runs the migrations.
- Vitest: resolve API paths with `new URL(path, location.href)` (Node's fetch doesn't use
  jsdom's location); MSW routes as `"*/api/…"`, `server.listen({ onUnhandledFrame:
  "error" })`.
- One e2e test only, for browser-decided behavior (the export download); downloads are
  served by a real HTTP server in the spec, not `page.route`.

### Packaging (step 15)

- Embedding packages: `web/web.go` (`//go:embed all:dist`) and `migrations/migrations.go`
  (`//go:embed *.sql`); `npm --prefix web run build` must run before `go build`.
- SPA handler: real files through `http.FileServerFS`; `assets/*` with `Cache-Control:
  public, max-age=31536000, immutable`; a missing path with an extension or under
  `assets/` is 404; every other path gets `index.html` with `Cache-Control: no-cache`;
  unknown `/api/` paths are 404 problems.
- Configuration (environment only, all problems reported at once with `errors.Join`):
  `DATABASE_URL` (required, `postgres://` or `postgresql://`, logged with
  `url.URL.Redacted`), `TRACELITE_OWNER` (required), `TRACELITE_ADDR` (default
  `127.0.0.1:8080`; a non-loopback address needs `TRACELITE_ALLOW_REMOTE=yes`),
  `TRACELITE_WORKERS` (1-32, default 4), `TRACELITE_STANDARDS_URL` (optional).
- Version: `var version = "dev"` in `cmd/tracelite`, set with `-ldflags "-X
  main.version=…"`, falling back to `debug.ReadBuildInfo` (module version, `vcs.revision`,
  `vcs.modified`). Release builds: `CGO_ENABLED=0`, `-trimpath`, `-ldflags "-s -w -X …"`,
  for windows/amd64 and linux/amd64.
- Startup order: config, logger, pool, migrate (goose, embedded), store and service, job
  pool, listener, HTTP. Shutdown in reverse: HTTP (10 s), jobs (20 s drain), listener,
  pool.
- Customer database: role `tracelite` owning database `tracelite` (CREATE privilege for
  step 12's trusted extensions); after creating it, `show_trgm('ığdır')` must return six
  entries. Windows service through `golang.org/x/sys/windows/svc` (or a wrapper), Linux
  through systemd.

### Runner limits met in steps 11-15

- sql runtime: the statement splitter used to cut SQL-standard `BEGIN ATOMIC … END`
  bodies at their inner semicolons ("syntax error at end of input"). Fixed on
  2026-11-14 (sqlrun/split.go follows psql's rule); step 13's trigger functions are
  PL/pgSQL with dollar-quoted bodies anyway.
- pgx: an argument whose placeholder the statement never uses makes the prepare fail
  ("could not determine data type of parameter"). In step 11 this silently broke a
  heartbeat whose error was ignored; log the errors of background loops.
- go runtime: no `-ldflags` and no Git checkout, so version stamps print `dev` (step 15).
- pgxpool opened its second connection 50-170 ms into a run on the local runner; a
  timing-based trigger in a pgx example must allow for that, or wait on database state
  (step 11).
- Playwright: in Chromium and Edge, `page.route`'s handler never runs for a download
  started by a link with the `download` attribute (Playwright issue #22650, closed
  unfixed; a probe on 2026-11-14 counted 0 route hits). The request goes to the network,
  and the file name comes from whatever answers there. A `node:http` server started in
  the spec behaves like the real server (step 14).
- EXPLAIN doesn't print the COLLATE of an inlined SQL function such as `tr_lower`.
