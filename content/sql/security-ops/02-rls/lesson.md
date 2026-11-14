---
id: sql.security-ops.rls
title: "Row-level security for multi-tenant isolation"
track: sql
module: security-ops
order: 2
level: advanced
minutes: 15
since: pg9.5
prerequisites: [sql.security-ops.roles, sql.modeling.multi-tenant]
compare_with: [sql.modeling.multi-tenant, sql.server-side.views, sql.security-ops.pooling-config, design.architecture.security, go.postgres.transactions]
tags: [row-level-security, rls, multi-tenant, policies, set_config, connection-pooling]
summary: >
  A policy turns "WHERE tenant_id = ?" into a property of the table: every query gets
  it, including the one somebody forgets to filter. Make it fail closed, force it on the
  owner, scope your keys by tenant, and set the tenant per transaction.
examples:
  - id: isolation
    file: examples/01_isolation.sql
    title: "One policy, every query: reads, silent writes, and a rejected insert"
    runtime: sql
    check: exact
    annotations:
      - lines: "14-16"
        note: >
          ENABLE switches policies on; with none defined the table would deny
          everything. A policy without FOR or TO covers every command and every
          role. current_setting(..., true) returns NULL rather than an error when
          the variable was never set, and nullif turns the empty string a session
          is left with later (example 3) into NULL as well.
      - lines: "19"
        note: >
          3 rows. Table owners bypass policies by default, and this session owns
          the table, as a service that runs its own migrations would.
      - lines: "21-22"
        note: >
          FORCE applies the policies to the owner too. Superusers and BYPASSRLS
          roles still skip them; nobody else does. With no tenant set the answer
          is zero rows, not all rows: failing closed is the property that matters.
      - lines: "24-25"
        note: >
          set_config(name, value, true) is SET LOCAL as a function call: the value
          lasts until the transaction ends, and unlike SET it accepts a bind
          parameter ($1) from the driver. The SELECT has no tenant filter and
          still returns only Acme's projects.
      - lines: "27"
        note: >
          The rewriter adds the policy's USING expression to the query, and the
          plan shows it as an ordinary Filter. current_setting() is stable, so
          with an index on (tenant_id, ...) the same predicate becomes an Index
          Cond, exactly like a hand-written WHERE. Lead your indexes with
          tenant_id.
      - lines: "30-31"
        note: >
          UPDATE 0, DELETE 0: a command only sees rows the policy lets it see,
          and the rest are skipped without an error. A handler that maps "no rows
          affected" to 404 then gives the same answer for "doesn't exist" and
          "belongs to someone else", which is what you want.
      - lines: "33-34"
        note: >
          Change the variable and you change the tenant. That is the trust model:
          any session that can run SQL as this role can call set_config. A policy
          keyed on a setting protects against a forgotten WHERE in your own code,
          not against an attacker who can run arbitrary SQL as the app role.
      - lines: "37"
        note: >
          SQLSTATE 42501. A policy with only USING applies it as WITH CHECK too,
          so INSERT and UPDATE can't produce a row outside the current tenant.
          Write a separate WITH CHECK only when the two must differ.
  - id: constraints
    file: examples/02_constraints.sql
    title: "Constraint checks look past policies: scope the keys by tenant"
    runtime: sql
    check: exact
    annotations:
      - lines: "1-12"
        note: >
          A global UNIQUE (code) and a single-column foreign key: the schema an
          application had before it became multi-tenant, with RLS added on top.
      - lines: "29-30"
        note: >
          Unique and foreign key checks always see every row, so that integrity
          holds whatever the policies say. INSERT 0 0 just told Acme that some
          tenant uses the code RAIL; without ON CONFLICT the duplicate-key error
          would say the same. The docs call these covert channels.
      - lines: "32-33"
        note: >
          The foreign key check found project 3, which Acme can't see, so the row
          went in: Acme's requirement now hangs off Globex's project, and the
          join shows Acme a dangling reference. Worse for Globex: it can no
          longer delete its own project, because of a referencing row it can't
          see.
      - lines: "36-42"
        note: >
          Tenant-scoped keys. Codes are unique per tenant, and the foreign key
          carries tenant_id, so a requirement can only point at a project of its
          own tenant. UNIQUE (tenant_id, id) exists to be the target of that
          composite key.
      - lines: "44"
        note: >
          Acme can now use RAIL too, so the probe above tells nothing any more.
          The primary key on id is still global; let an identity column assign
          ids (GENERATED ALWAYS) so a client can't pick an id to test for.
      - lines: "45"
        note: >
          SQLSTATE 23503: there is no project (1, 3), because project 3 belongs
          to tenant 2. The database refuses the cross-tenant reference whatever
          the application does.
  - id: pooled-context
    file: examples/03_pooled_context.sql
    title: "Tenant context on a reused connection: transaction-local or leaked"
    runtime: sql-nontx
    check: exact
    annotations:
      - lines: "11-14"
        note: >
          The pattern for a pooled service: begin, set the tenant with
          is_local = true, run the request's queries, commit. In Go it lives in
          the one helper that starts transactions:
          tx.Exec(ctx, "SELECT set_config('app.tenant', $1, true)", tenantID).
      - lines: "17-19"
        note: >
          After COMMIT the local value is gone, but the variable now exists in
          this session and reads as an empty string, not NULL. nullif maps it
          back to NULL, so a request that forgot the tenant sees nothing.
      - lines: "22"
        note: >
          is_local = false is a plain SET: it lasts for the rest of the session,
          and with a pool that means for whichever requests get this connection
          next.
      - lines: "26"
        note: >
          The leak. Request 4 belongs to another user, set no tenant, and reads
          tenant 2's projects, because a pool hands the same server session to
          the next borrower with its settings intact. Behind PgBouncer in
          transaction mode it is worse: consecutive transactions of one client
          may not even share a server connection.
      - lines: "29-32"
        note: >
          Without nullif, the empty leftover reaches the bigint cast and every
          query on the table fails with SQLSTATE 22P02, on every connection that
          ever had a tenant set. Still closed, but as an error instead of an
          empty result. RESET leaves the same empty value that COMMIT did.
use_cases:
  - "Shared-schema multi-tenant SaaS: every tenant's rows in the same tables, separated by tenant_id and enforced by policies"
  - "Defence in depth under an application that already filters by tenant: a forgotten WHERE in a report or an ad-hoc admin query returns nothing instead of everyone's data"
  - "Per-user visibility rules (own rows, or rows of my team) when users map to database roles, keyed on current_user"
  - "Read-only analysts or support staff who may query production but only for the customers they are assigned, a KVKK-style data minimisation measure"
strengths:
  - "Enforced below every client: ORM queries, raw SQL, reports and psql sessions all get the same predicate, unlike Hibernate filters or Rails default scopes, which only cover queries that go through them"
  - "Fails closed when written with current_setting(..., true): no tenant means no rows, not all rows"
  - "Policies are plain SQL predicates, so the planner uses ordinary indexes on tenant_id"
  - "Writes are checked as well: WITH CHECK stops a request from creating or moving rows into another tenant"
weaknesses:
  - "Superusers, BYPASSRLS roles and (without FORCE) table owners skip every policy, so the role setup decides whether RLS means anything"
  - "Unique and foreign key checks ignore policies: global keys leak existence and allow cross-tenant references unless keys include tenant_id"
  - "A tenant kept in a session variable is only as trustworthy as the code that sets it; anyone who can run SQL as the app role can switch tenants"
  - "Hidden rows make UPDATE and DELETE silently affect nothing, which can mask bugs; and views run with their owner's policies unless created WITH (security_invoker = true) (PostgreSQL 15+)"
pitfalls:
  - title: "Enabling RLS without FORCE while the app owns the tables"
    detail: "The owner bypasses policies, so everything keeps working and nothing is isolated. Either connect as a non-owner role (the roles lesson), or add FORCE, or both."
    bad: "ALTER TABLE projects ENABLE ROW LEVEL SECURITY;  -- app connects as the owner"
    good: "ALTER TABLE projects ENABLE ROW LEVEL SECURITY;\nALTER TABLE projects FORCE ROW LEVEL SECURITY;"
  - title: "current_setting without missing_ok, or without nullif"
    detail: "current_setting('app.tenant') raises an error when the variable was never set; after a transaction-local set, the variable reads '' for the rest of the session and a bare ::bigint cast fails. Use the two-argument form and map '' to NULL."
    bad: "USING (tenant_id = current_setting('app.tenant')::bigint)"
    good: "USING (tenant_id = nullif(current_setting('app.tenant', true), '')::bigint)"
  - title: "Setting the tenant for the session on a pooled connection"
    detail: "SET app.tenant or set_config(..., false) outlives the request and is inherited by the next borrower of the connection. Set it with is_local = true inside the request's transaction, every time."
    bad: "SELECT set_config('app.tenant', $1, false);"
    good: "BEGIN; SELECT set_config('app.tenant', $1, true); ... COMMIT;"
  - title: "A view owned by the table owner"
    detail: "By default a view reads its tables with its owner's privileges and policies. If the owner bypasses RLS (no FORCE), every user of the view bypasses it too. Create views over RLS tables WITH (security_invoker = true)."
    good: "CREATE VIEW open_requirements WITH (security_invoker = true) AS SELECT ... FROM requirements WHERE status = 'open';"
  - title: "Backing up as the app role"
    detail: "pg_dump sets row_security = off so a backup can't silently miss rows; a role that can't bypass RLS gets an error instead of a partial dump. Back up as a role with BYPASSRLS (or a superuser), never with --enable-row-security by accident."
quiz:
  - q: "RLS is enabled with a tenant policy, the service connects as the role that owns the tables, and FORCE is not set. What does a query without a tenant filter return?"
    options: ["Only the current tenant's rows", "All tenants' rows", "An error"]
    answer: 1
    why: "Table owners bypass row security unless the table is set to FORCE ROW LEVEL SECURITY."
  - q: "Tenant 1 runs UPDATE projects SET name = 'x' WHERE id = 3, and project 3 belongs to tenant 2. What happens?"
    options: ["permission denied for table projects", "new row violates row-level security policy", "UPDATE 0"]
    answer: 2
    why: "USING decides which existing rows the command can see; invisible rows are skipped silently. The WITH CHECK error only fires for rows the command produces."
  - q: "Policies isolate tenants, yet tenant 1 managed to insert a requirement pointing at tenant 2's project. What fixes it?"
    options: ["A second policy on projects", "A foreign key on (tenant_id, project_id) referencing projects (tenant_id, id)", "FORCE ROW LEVEL SECURITY on requirements"]
    answer: 1
    why: "Foreign key checks bypass row security; only a key that includes tenant_id keeps references inside one tenant."
  - q: "Why set_config('app.tenant', $1, true) rather than SET app.tenant = ... in a pooled Go service?"
    options: ["SET is slower", "The value ends with the transaction instead of leaking to the next borrower, and set_config takes a bind parameter", "SET cannot set custom variables"]
    answer: 1
    why: "is_local = true scopes the value to the transaction; SET is a utility command with no $1 parameters, so the value would have to be spliced into SQL text."
further_reading:
  - title: "PostgreSQL docs: Row Security Policies"
    url: "https://www.postgresql.org/docs/current/ddl-rowsecurity.html"
  - title: "PostgreSQL docs: CREATE POLICY"
    url: "https://www.postgresql.org/docs/current/sql-createpolicy.html"
  - title: "PostgreSQL docs: ALTER TABLE (ENABLE/FORCE ROW LEVEL SECURITY)"
    url: "https://www.postgresql.org/docs/current/sql-altertable.html"
  - title: "PostgreSQL docs: Configuration Settings Functions (current_setting, set_config)"
    url: "https://www.postgresql.org/docs/current/functions-admin.html#FUNCTIONS-ADMIN-SET"
  - title: "PostgreSQL docs: CREATE VIEW (security_invoker)"
    url: "https://www.postgresql.org/docs/current/sql-createview.html"
---

## Concept

Row-level security (RLS) attaches **policies** to a table. A policy has a `USING`
expression, which decides which existing rows a command can see (`SELECT`, and the rows
an `UPDATE` or `DELETE` can touch), and a `WITH CHECK` expression, which every new or
changed row must satisfy. Once `ENABLE ROW LEVEL SECURITY` is set, the planner adds the
policies to every query on the table, and a table with RLS enabled but no policy denies
everything.

For a shared-schema multi-tenant design (tenancy patterns are compared in the modeling
module), the policy is the tenant filter itself:

```sql
CREATE POLICY tenant_isolation ON projects
  USING (tenant_id = nullif(current_setting('app.tenant', true), '')::bigint);
```

The service sets `app.tenant` at the start of each transaction from the authenticated
request. From then on, a query that forgets `WHERE tenant_id = $1` is still correct,
because the database filters for it.

Who is exempt matters as much as the policy: superusers and `BYPASSRLS` roles always,
and the table's owner unless the table is set to `FORCE ROW LEVEL SECURITY`.

## Coming from C/C++/Java/Python/JS

- **ORM tenant filters** (Hibernate's `@TenantId` and filters, Rails' default scopes and
  tenant gems, a Django manager that adds `.filter(tenant=...)`) work on queries that go
  through the ORM. A raw SQL report, a migration script or a psql session skips them.
  A policy is applied by the server to all of them.
- **Middleware that sets request context** (a Go `context.Context` value, a Java
  `ThreadLocal`, an `AsyncLocalStorage` store in Node) has a database twin here: the
  transaction-local setting. Both must be set at the start of every request, and both
  leak if they outlive it.
- **Unix permissions** decide per file; policies decide per row, and the check runs
  inside the query, so the planner can still use indexes.

## How it works

- **Combining policies.** Policies are per command (`FOR SELECT`, `INSERT`, `UPDATE`,
  `DELETE`, or `ALL`) and per role (`TO ...`, default `PUBLIC`). Permissive policies
  (the default) are combined with `OR`; `AS RESTRICTIVE` policies are combined with
  `AND` on top. A tenant rule plus "support staff see only open tickets" is a permissive
  policy and a restrictive one.
- **Order of evaluation.** Policy predicates run before conditions from the user's
  query, so a user-written function in `WHERE` can't observe rows the policy hides.
  Functions marked `LEAKPROOF` (such as `=` on integers and text) may run earlier, which
  is what lets the planner use an index on the user's own conditions.
- **Indexes.** With an index on `(tenant_id, id)`, `EXPLAIN` on the first example's
  table shows the policy as `Index Cond: (tenant_id = (NULLIF(current_setting(...),
  ''))::bigint)`, the same plan a hand-written filter gets.
- **What policies don't cover.** `TRUNCATE` and `REFERENCES` act on the whole table,
  and unique, primary key and foreign key checks always see every row (example 2).
- **Backups.** `pg_dump` sets `row_security = off`, which doesn't bypass anything; it
  makes any query that a policy would filter fail instead of returning fewer rows. The
  backup role therefore needs `BYPASSRLS`.

## Design notes

- **Where the tenant comes from.** Only from the authenticated session in the service,
  never from a request body or a URL. Set it in the one helper that opens transactions,
  with `set_config(..., true)`, and run every request in a transaction, reads included.
- **Two roles, belt and braces.** The service logs in as a role that doesn't own the
  tables (the roles lesson), so policies apply to it by default, and `FORCE` covers the
  day someone runs the service as the owner. Migrations, backups and support tooling use
  roles that bypass RLS on purpose.
- **Keys carry the tenant.** `UNIQUE (tenant_id, code)`, foreign keys on `(tenant_id,
  parent_id)`, indexes that start with `tenant_id`. In TraceLite, projects, requirements,
  test cases, runs and trace links all carry `tenant_id`, and every link between them
  includes it.
- **Session variable or database role.** Keying policies on `current_user` (one role
  per tenant or per user) is a stronger boundary, because a role can't be switched with
  a function call, but it costs a role per tenant and a connection pool per role. The
  session variable is the common compromise: it protects against bugs in your own
  queries, not against SQL injection with full statement control, which needs to be
  prevented anyway (parameters, never string concatenation).
- **In this sandbox** `playground` owns every table it creates, which is why the examples
  need `FORCE`. In the optional Docker mode the playground role is the container's
  superuser, and superusers bypass all policies: these examples assume local mode.
