---
id: sql.basics.mental-model
title: "Client/server, databases vs schemas vs roles"
track: sql
module: basics
order: 1
level: basic
minutes: 12
since: pg15
prerequisites: []
compare_with: [go.tooling.project-layout, go.postgres.pgx, sql.security-ops.roles]
tags: [architecture, schemas, roles, search-path]
summary: >
  One PostgreSQL server (a "cluster") holds many databases; a connection is one
  role inside one database, and schemas are namespaces within it, resolved
  through search_path. Roles live above all databases, which is why they can
  own databases and schemas alike.
examples:
  - id: connection
    file: examples/01_connection.sql
    title: "Who and where is this connection?"
    runtime: sql
    check: exact
    annotations:
      - lines: "5-7"
        note: >
          pg_stat_activity has one row per server process. 'client backend'
          is a process serving a client, as opposed to background processes
          such as the checkpointer. The server starts one per connection,
          which is why opening a connection is expensive and why Go and
          Node apps keep a pool of them (pgxpool, pg.Pool) instead.
      - lines: "11"
        note: >
          session_user is the role that logged in. current_user is the role
          privileges are checked against; it moves away from session_user
          after SET ROLE and inside SECURITY DEFINER functions.
      - lines: "16-19"
        note: >
          The path holds only the sandbox schema, yet current_schemas(true)
          lists pg_catalog first: when the path doesn't name it, pg_catalog is
          searched implicitly, before everything else. That is how upper(),
          now() and every built-in type resolve without a prefix.
  - id: schemas
    file: examples/02_schemas.sql
    title: "Same table name, two schemas: search_path picks"
    runtime: sql
    check: exact
    annotations:
      - lines: "3-6"
        note: >
          A schema is a namespace, not a separate store: creating one needs
          no new connection, and queries can combine tables from different
          schemas freely, as the last query does.
      - lines: "11-15"
        note: >
          The same unqualified name, tickets, means a different table after
          each SET: the first schema on the path that has a match wins.
          SET LOCAL lasts until the transaction ends; a plain SET would last
          for the whole session, which matters when connections are pooled.
      - lines: "18-22"
        note: >
          refunds has no prefix, so it lands in the current schema: the first
          existing schema on the path, billing by now. Migration tools and
          ORMs that create unprefixed tables depend on this setting.
      - lines: "25-28"
        note: >
          A qualified name never consults the path. Code that must not be
          redirected by someone else's search_path (functions, migrations)
          qualifies its names or sets the path explicitly.
  - id: cluster
    file: examples/03_cluster.sql
    title: "Cluster-wide vs per-database, read from the catalogs"
    runtime: sql
    check: exact
    annotations:
      - lines: "3-6"
        note: >
          relisshared marks catalogs stored once for the whole server:
          pg_authid holds the roles, pg_database the databases. pg_class
          (tables) and pg_namespace (schemas) exist separately inside every
          database, so a connection can only ever see its own database's.
      - lines: "8-15"
        note: >
          In a new database (PG 15+) public belongs to pg_database_owner, a
          role whose one implicit member is the current database's owner.
          Before 15 the bootstrap superuser owned it and every role could
          create objects in it; clusters upgraded from 14 keep those old
          permissions. StackLab's setup-local-db.sql hands the playground's
          public to postgres: lesson code runs as the database owner, and
          this way it can use public but not create objects in it.
      - lines: "17-19"
        note: >
          The playground role owns the playground database. That ownership,
          not superuser rights, is what lets StackLab's runner create a
          throwaway schema for every run.
      - lines: "23"
        note: >
          The first part of a three-part name is the database (the SQL
          standard calls it the catalog), and it may only name the one you
          are connected to. Reaching another database takes a second
          connection or an extension such as postgres_fdw.
use_cases:
  - "One database per application; schemas inside it for parts that must still be joined (app, audit, reporting)"
  - "Separate databases for unrelated projects on one server, so they cannot see each other's tables"
  - "Distinct roles for running migrations (owns the schema) and for the app itself (only the privileges it needs)"
strengths:
  - "Schemas are cheap namespaces you can join across. What MySQL calls a database is closer to a PostgreSQL schema; a PostgreSQL database is a harder boundary"
  - "Roles cover users and groups with one concept (since 8.1): grant privileges to a group role, then grant the group to login roles"
  - "Databases are isolated by construction: a connection to one cannot read another by accident, no matter what SQL it sends"
weaknesses:
  - "No cross-database queries: data that must be joined has to live in one database, or go through postgres_fdw"
  - "One server process per connection makes connections expensive; hundreds of idle ones cost real memory, so applications need a pool"
  - "search_path is implicit session state: the same SQL text can reach different tables under different settings"
pitfalls:
  - title: "Planning on cross-database joins"
    detail: "Coming from MySQL or SQL Server, it's natural to split modules into databases and join across them. In PostgreSQL that fails with 'cross-database references are not implemented'. Use schemas in one database for anything you'll join."
  - title: "permission denied for schema public on a fresh database"
    detail: "Since PG 15 ordinary roles can no longer create objects in public; only the database owner can. A migration that ran on an old server fails on a new one. Have the database owner run migrations, or create a schema for the app and grant CREATE on it."
    bad: "CREATE TABLE users (...);  -- as a role that doesn't own the database"
    good: "CREATE SCHEMA app AUTHORIZATION app_owner;  -- then create tables in app"
  - title: "A SET that outlives your request"
    detail: "SET search_path (or any SET) changes the server session, and a pooled connection carries it into the next request that borrows it. Use SET LOCAL inside a transaction, or configure the setting per role with ALTER ROLE ... SET."
quiz:
  - q: "Your Go service needs to join orders (database shop) with users (database auth) on the same server. What happens?"
    options:
      - "It works: write shop.public.orders JOIN auth.public.users"
      - "It fails: a connection reaches one database; put both in one database (two schemas) or use postgres_fdw"
      - "It works as long as both databases have the same owner"
    answer: 1
    why: "Every connection is bound to one database, and a three-part name may only name that one. Schemas are the joinable unit."
  - q: "search_path is billing, support and both schemas have a tickets table. What do SELECT * FROM tickets and CREATE TABLE refunds (...) use?"
    options:
      - "billing.tickets, and refunds is created in billing"
      - "support.tickets, and refunds is created in support"
      - "An ambiguity error, and refunds is created in public"
    answer: 0
    why: "Lookup walks the path in order and takes the first match; creation goes to the first existing schema on the path."
  - q: "Where do roles live?"
    options:
      - "In each database, like schemas"
      - "In each schema"
      - "At the cluster level, shared by every database on the server"
    answer: 2
    why: "Roles are global to the cluster (pg_authid is a shared catalog), which is why a role can own a database and objects in several databases."
  - q: "On a new PostgreSQL 18 database, a role that doesn't own the database runs CREATE TABLE public.t (id int). What happens?"
    options:
      - "The table is created; public is open to everyone"
      - "permission denied for schema public: since PG 15 only the database owner (via pg_database_owner) can create there by default"
      - "The table is created in the role's own schema instead"
    answer: 1
    why: "PG 15 removed PUBLIC's CREATE privilege on public and made pg_database_owner its owner."
further_reading:
  - title: "PostgreSQL docs: Architectural Fundamentals"
    url: "https://www.postgresql.org/docs/current/tutorial-arch.html"
  - title: "PostgreSQL docs: Managing Databases, Overview"
    url: "https://www.postgresql.org/docs/current/manage-ag-overview.html"
  - title: "PostgreSQL docs: Schemas (including the search path and usage patterns)"
    url: "https://www.postgresql.org/docs/current/ddl-schemas.html"
  - title: "PostgreSQL docs: Database Roles"
    url: "https://www.postgresql.org/docs/current/database-roles.html"
---

## Concept

PostgreSQL is a server program. Clients (psql, your Go service through pgx, a GUI) connect
to it over TCP or a local socket, and every connection names two things: a **role** to
log in as and a **database** to work in. The server starts a separate process for each
connection; that process holds your session's state (settings, the open transaction)
until you disconnect.

The containers, from the outside in:

- **Cluster**: one running server with its data directory, usually one port. The
  docs call it a "database cluster"; it has nothing to do with multi-node clustering.
- **Database**: an isolated collection of schemas. A connection is bound to exactly
  one, and no query can reach into another.
- **Schema**: a namespace inside a database, holding tables, views, functions, types
  and sequences. Schemas cannot nest, and joining across them is ordinary SQL.
- **Role**: a user, a group, or both. Roles live at the cluster level, next to the
  databases rather than inside them, so one role can own objects in many databases.

## Coming from C/C++/Java/Python/JS

- **MySQL**: `CREATE SCHEMA` is a synonym for `CREATE DATABASE` there, and you can join
  across them. That unit corresponds to a PostgreSQL *schema*. A PostgreSQL database is
  a much harder wall.
- **Go and Java packages, C++ namespaces**: a schema is the same idea: `billing.tickets`
  vs `support.tickets`, like `billing.Ticket` vs `support.Ticket`. `search_path` is like
  a list of C++ `using namespace` directives, except that where C++ reports an
  ambiguity, PostgreSQL silently takes the first match.
- **Python's `sys.path`** is a closer analogy still: an ordered search list that the
  environment can change under you. Code that must be robust qualifies its names.
- **Node and Go connection handling**: a connection is a server process, not a cheap
  socket. That's why `pgxpool` and `pg.Pool` exist, and why a `SET` on a pooled
  connection leaks into whichever request borrows it next.

## How it works

```mermaid
flowchart TB
  client1["psql"] -- "role + database" --> server
  client2["Go service (pgxpool)"] -- "role + database" --> server
  subgraph server["cluster: one server, one data directory"]
    roles["roles (shared by all databases)"]
    subgraph db1["database: playground"]
      s1["schema: public"]
      s2["schema: run_… (one per lesson run)"]
    end
    subgraph db2["database: stacklab"]
      s3["schema: public"]
    end
  end
```

Name resolution happens when a statement is parsed. An unqualified `tickets` is looked
up in each schema on `search_path`, in order. `pg_catalog`, home of the built-in
functions and types, is searched first unless the path places it explicitly. The
default path is `"$user", public`: a schema named after your role if one exists, then
`public`. New objects without a prefix go into the first schema on the path that exists.

Every StackLab SQL run sits inside this picture: the runner connects as the
`playground` role to the `playground` database, creates schema `run_<id>`, sets the
path to it, runs your SQL in one transaction and rolls it back.

## Design notes

Choose the boundary by asking whether the data will ever be joined. Unrelated
applications get separate databases, and the docs recommend exactly that for projects
that should be unaware of each other. Parts of one application (core tables, an
audit trail, reporting views) get schemas in one database, so a report can join them
and one transaction can span them.

The PG 15 change to `public` is part of the same design: a database is owned by
someone, and by default only that owner may create objects in the shared schema. The
security module builds on this with a least-privilege application role. It is also why
StackLab's local setup makes the playground role the owner of its database instead of a
superuser: owning the database is enough to create schemas in it, and nothing more is
handed out.
