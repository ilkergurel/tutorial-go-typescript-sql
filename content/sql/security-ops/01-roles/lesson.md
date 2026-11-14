---
id: sql.security-ops.roles
title: "Roles and privileges: GRANT, default privileges, a least-privilege app user"
track: sql
module: security-ops
order: 1
level: intermediate
minutes: 15
since: pg15
prerequisites: [sql.basics.mental-model]
compare_with: [sql.security-ops.rls, design.architecture.security, inside.codebase.runner-sandbox, sql.files-movement.copy, go.postgres.migrations-sqlc]
tags: [roles, privileges, grant, revoke, default-privileges, least-privilege, kvkk]
summary: >
  A role's attributes decide what it may do to the server; privileges decide what it may
  do to each object; ownership is a third, irrevocable power. Run the app as a role that
  owns nothing, and give it exactly the grants its queries need.
examples:
  - id: whoami
    file: examples/01_whoami.sql
    title: "The sandbox role: what it is, and what it may not do"
    runtime: sql
    check: exact
    annotations:
      - lines: "3-5"
        note: >
          Attributes are switches on the role itself, set by CREATE ROLE and
          ALTER ROLE, not by GRANT. Not a superuser, so every permission check
          applies; no CREATEROLE or CREATEDB, so it can't mint identities or
          databases; no BYPASSRLS, so row-level policies (next lesson) bind it.
      - lines: "8-10"
        note: >
          'USAGE' asks whose privileges this role holds without SET ROLE.
          pg_database_owner is a predefined role whose one implicit member is
          the current database's owner. Since PostgreSQL 15 it owns every new
          database's public schema, so by default whoever owns a database
          controls its public schema.
      - lines: "14-20"
        note: >
          EXECUTE inside an EXCEPTION block runs in a subtransaction: a failure
          is undone and reported, and the list goes on. Only SQLSTATE 42501
          (insufficient_privilege) is caught, so any other error still stops the
          script and 'allowed' really means allowed. Successful statements stay
          in effect until the run's final ROLLBACK.
      - lines: "24-26"
        note: >
          Identity management is closed: no new roles, no new attributes for
          itself, and no switching into a role it is not a member of, not even a
          predefined one such as pg_read_all_stats.
      - lines: "27"
        note: >
          The reason this role is not a superuser (ADR-0008): COPY ... TO
          PROGRAM runs a shell command as the operating-system account the
          server runs under, here on your own machine. Only superusers and
          members of pg_execute_server_program may. psql's \copy runs on the
          client instead and is open to everyone.
      - lines: "28-29"
        note: >
          No server files and no password hashes. pg_authid is readable by
          superusers only; pg_roles is the public view of it, with the password
          column always reading ********.
      - lines: "30"
        note: >
          Logging settings have context 'superuser' in pg_settings: an ordinary
          role can neither silence nor flood the server log.
      - lines: "31-32"
        note: >
          Inside its own database the owner is strong: it creates schemas, and
          on a fresh database it could drop public through pg_database_owner.
          StackLab's setup gave the playground's public to postgres, so that
          drop is refused here. That is the intended blast radius: one
          database, where each sql run is rolled back and each sql-nontx run's
          schema dropped. Nothing outside it.
  - id: grant-revoke
    file: examples/02_grant_revoke.sql
    title: "GRANT and REVOKE, down to columns; ownership survives REVOKE ALL"
    runtime: sql
    check: exact
    annotations:
      - lines: "9"
        note: >
          relacl stays NULL until the first GRANT or REVOKE. NULL means the
          built-in default: every privilege for the owner, none for anyone else.
          Tables give PUBLIC nothing by default; functions do (example 3).
      - lines: "12-15"
        note: >
          A column grant: a reporting role can see who a customer is without the
          national ID. has_table_privilege is false because there is no
          table-wide SELECT; the column checks see the grant. The role name
          'public' asks about the PUBLIC pseudo-role, which every role belongs to.
      - lines: "18-19"
        note: >
          The app-role pattern, played on ourselves because this sandbox can't
          create roles: INSERT, UPDATE of name only, no DELETE or TRUNCATE. An
          owner may revoke its own ordinary privileges, and always keeps the
          grant options to give them back.
      - lines: "22-25"
        note: >
          An aclitem reads grantee=privileges/grantor, and an empty grantee is
          PUBLIC. r SELECT, a INSERT, w UPDATE, d DELETE, D TRUNCATE,
          x REFERENCES, t TRIGGER, m MAINTAIN (PostgreSQL 17). psql's \dp prints
          the same strings.
      - lines: "27-28"
        note: >
          Both succeed through column grants. The UPDATE needs UPDATE on name
          and, because its WHERE reads id, SELECT on id, which PUBLIC (so every
          role, this one included) holds.
      - lines: "31"
        note: >
          The right to alter or drop an object comes with ownership and can't be
          revoked. A service that connects as the owner of its tables can always
          DROP them, whatever you REVOKE: that is why the owner and the app are
          separate roles.
      - lines: "33"
        note: >
          SQLSTATE 42501. * expands to every column, including tckn (never
          granted) and email (added after the grants, so nobody has it yet). New
          columns stay private until someone grants them.
  - id: default-privileges
    file: examples/03_default_privileges.sql
    title: "Default privileges for future objects, and the sequence trap"
    runtime: sql
    check: exact
    annotations:
      - lines: "10-12"
        note: >
          Defaults differ by object type. A new schema and a new table give
          PUBLIC nothing, but every new function is executable by PUBLIC.
          Harmless for a pure function; for a SECURITY DEFINER function, which
          runs with its owner's rights, it's an open door.
      - lines: "16-17"
        note: >
          Default privileges belong to the creating role (here the current one)
          and are applied when an object is created. In production the creator is
          the migration role: ALTER DEFAULT PRIVILEGES FOR ROLE tracelite_owner
          IN SCHEMA tracelite GRANT ... TO tracelite_app.
      - lines: "18"
        note: >
          The documented way to drop PUBLIC's EXECUTE on future functions. It
          has to be global: per-schema defaults are added to the global ones, so
          a per-schema REVOKE can't take away what the global default grants.
      - lines: "19-22"
        note: >
          pg_default_acl is what psql's \ddp lists. Kind r is tables, f
          functions; the global entry has schema OID 0. The function entry now
          holds only the owner's EXECUTE.
      - lines: "32-34"
        note: >
          The new table got the default grants and the new function lost
          PUBLIC's EXECUTE, while projects, created earlier, kept its old ACL.
          Defaults never touch existing objects; GRANT ... ON ALL TABLES IN
          SCHEMA does that, once, in the same migration.
      - lines: "38-40"
        note: >
          The identity column's INSERT succeeds without any sequence privilege.
          serial calls nextval(), which needs USAGE or UPDATE on the sequence:
          SQLSTATE 42501. An app role built from table grants alone fails on its
          first INSERT into a serial table; grant ON SEQUENCES too, or use
          identity columns.
use_cases:
  - "A separate owner role for migrations and a login role with DML only for the running service"
  - "A read-only reporting or BI role with column grants that leave out personal data (TCKN, phone, address), one of the technical measures KVKK Article 12 asks for"
  - "A monitoring login that is a member of pg_monitor instead of a superuser"
  - "Auditing a database: who can read or change what, from relacl, attacl, has_*_privilege and information_schema.table_privileges"
strengths:
  - "Privileges are enforced by the server on every statement, whatever the client: an ORM, psql or an injected query all hit the same check, unlike permission checks in application code"
  - "Grants reach down to single columns, so personal data can be hidden from a role without a view or a second table"
  - "Roles are users and groups in one concept: grant privileges to a NOLOGIN group role, then grant the group to login roles"
  - "Predefined roles (pg_monitor, pg_read_all_data, pg_maintain, pg_signal_backend) give operational powers without superuser"
weaknesses:
  - "Ownership can't be restricted: the owner can always ALTER, DROP and re-grant, so least privilege depends on the app not owning anything"
  - "Default privileges are per creating role and only affect future objects, two details that make 'the app can't read the new table' a classic deploy failure"
  - "PUBLIC gets EXECUTE on every new function and CONNECT and TEMPORARY on every new database unless you revoke them"
  - "Roles are cluster-wide and not dumped by pg_dump: they live in migrations or provisioning scripts outside the database's own backup"
pitfalls:
  - title: "The service connects as the owner of its tables"
    detail: "REVOKE does nothing against ownership, and table owners bypass row-level security unless FORCE is set. An SQL injection or a bad migration in the service can DROP or rewrite anything it owns. Own the schema with a NOLOGIN role; log in as a role that owns nothing."
    bad: "CREATE TABLE requirements (...);  -- run by the role the service logs in as"
    good: "CREATE SCHEMA tracelite AUTHORIZATION tracelite_owner;  -- migrations SET ROLE tracelite_owner; the service logs in as tracelite_app"
  - title: "ALTER DEFAULT PRIVILEGES run as the wrong role"
    detail: "Default privileges apply to objects created by the role they were set for. Run by an admin without FOR ROLE, they cover the admin's future tables, not the ones the migration role creates, and the app gets 'permission denied' after the next deploy."
    bad: "ALTER DEFAULT PRIVILEGES IN SCHEMA tracelite GRANT SELECT ON TABLES TO tracelite_app;  -- as postgres"
    good: "ALTER DEFAULT PRIVILEGES FOR ROLE tracelite_owner IN SCHEMA tracelite GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO tracelite_app;"
  - title: "Forgetting sequences"
    detail: "TABLES defaults don't cover sequences. serial columns need USAGE on their sequence; identity columns don't. Grant both kinds explicitly."
    good: "ALTER DEFAULT PRIVILEGES FOR ROLE tracelite_owner IN SCHEMA tracelite GRANT USAGE ON SEQUENCES TO tracelite_app;"
  - title: "A SECURITY DEFINER function anyone can call"
    detail: "New functions are executable by PUBLIC, and a SECURITY DEFINER function runs with its owner's privileges. Revoke PUBLIC in the same transaction that creates it, and pin its search_path with pg_temp last so callers can't shadow the objects it uses."
    good: "BEGIN; CREATE FUNCTION ... SECURITY DEFINER SET search_path = tracelite, pg_temp; REVOKE ALL ON FUNCTION ... FROM PUBLIC; GRANT EXECUTE ON FUNCTION ... TO tracelite_app; COMMIT;"
quiz:
  - q: "Your service logs in as the role that owns its tables. You run REVOKE DELETE, TRUNCATE ON requirements FROM that role. What can the service still do?"
    options: ["Nothing destructive", "DROP TABLE requirements, or GRANT DELETE back to itself", "Only TRUNCATE"]
    answer: 1
    why: "Altering and dropping come with ownership and can't be revoked, and owners always hold the grant options to restore their own privileges."
  - q: "As tracelite_owner you run ALTER DEFAULT PRIVILEGES IN SCHEMA tracelite GRANT SELECT ON TABLES TO reporting. Which tables can reporting read?"
    options: ["Every existing table in tracelite", "Tables tracelite_owner creates in tracelite from now on", "Tables any role creates in tracelite from now on"]
    answer: 1
    why: "Default privileges apply at creation time, to objects created by the role they belong to. Existing tables need GRANT ... ON ALL TABLES IN SCHEMA."
  - q: "Why does lesson SQL here run as a non-superuser?"
    options: ["Superusers cannot create schemas", "A superuser can run shell commands and read files on the machine through COPY ... TO PROGRAM and the file functions", "Superusers are slower because every check is logged"]
    answer: 1
    why: "Superusers bypass all permission checks, including the ones that keep SQL inside the database. ADR-0008 runs lessons on the owner's own machine, so that boundary matters."
  - q: "An app role has INSERT on a table whose id column is serial, and no other grant. What does its INSERT do?"
    options: ["Succeeds", "Fails: permission denied for sequence", "Fails: permission denied for table"]
    answer: 1
    why: "serial's default calls nextval(), which needs USAGE or UPDATE on the sequence. An identity column would have succeeded."
further_reading:
  - title: "PostgreSQL docs: Privileges (including the ACL abbreviations)"
    url: "https://www.postgresql.org/docs/current/ddl-priv.html"
  - title: "PostgreSQL docs: GRANT"
    url: "https://www.postgresql.org/docs/current/sql-grant.html"
  - title: "PostgreSQL docs: ALTER DEFAULT PRIVILEGES"
    url: "https://www.postgresql.org/docs/current/sql-alterdefaultprivileges.html"
  - title: "PostgreSQL docs: Role Attributes"
    url: "https://www.postgresql.org/docs/current/role-attributes.html"
  - title: "PostgreSQL docs: Predefined Roles"
    url: "https://www.postgresql.org/docs/current/predefined-roles.html"
  - title: "PostgreSQL docs: Access Privilege Inquiry Functions"
    url: "https://www.postgresql.org/docs/current/functions-info.html#FUNCTIONS-INFO-ACCESS"
  - title: "PostgreSQL docs: Writing SECURITY DEFINER Functions Safely"
    url: "https://www.postgresql.org/docs/current/sql-createfunction.html#SQL-CREATEFUNCTION-SECURITY"
---

## Concept

PostgreSQL answers "may this connection do that?" with three independent mechanisms:

- **Role attributes** are cluster-wide switches on the role: `SUPERUSER`, `CREATEROLE`,
  `CREATEDB`, `BYPASSRLS`, `LOGIN`. A superuser bypasses every check except the right to
  log in.
- **Privileges** are granted per object: `CONNECT` on a database, `USAGE` on a schema,
  `SELECT`/`INSERT`/`UPDATE`/`DELETE` on a table or on single columns, `EXECUTE` on a
  function. `PUBLIC` is an implicit group containing every role, present and future.
- **Ownership** is the third power. The owner of an object holds all privileges by
  default, can always grant them again, and alone can `ALTER` or `DROP` it. That right
  can't be granted or revoked.

Least privilege follows directly: the role your service logs in with should own nothing
and hold only the grants its queries need. A separate role owns the schema and runs the
migrations.

The sandbox you have been using is built exactly this way. `scripts/setup-local-db.sql`
creates `playground` with `NOSUPERUSER NOCREATEDB NOCREATEROLE` and makes it the owner of
the `playground` database, nothing more. Ownership is enough for the runner
(`server/internal/sqlrun`) to create a throwaway `run_...` schema per run; the missing
superuser bit is what keeps lesson SQL from reaching your machine (ADR-0008).

## Coming from C/C++/Java/Python/JS

- **Unix:** a superuser is root. Ownership works like a file owner's right to `chmod`:
  the owner can always restore its own permissions, so revoking privileges from an owner
  is a seatbelt, not a lock. Predefined roles such as `pg_monitor` or `pg_signal_backend`
  are closer to Linux capabilities: one specific power without root.
- **Java/Spring:** many services ship with `spring.datasource.username=postgres`. Flyway
  and Liquibase can run migrations with their own credentials; that split (owner for DDL,
  app role for DML) is what this lesson builds.
- **MySQL:** accounts are `user@host` and roles came late (8.0). In PostgreSQL a user is
  simply a role with `LOGIN`, groups are roles too, and host rules live separately in
  `pg_hba.conf`.
- **Application-level checks:** an `if user.isAdmin` in your code protects one code path.
  A missing `GRANT` protects the data from every code path, including an injected query.

## How it works

A statement passes several gates, in this order: the `LOGIN` attribute and `pg_hba.conf`
at connect time, `CONNECT` on the database, `USAGE` on each schema it names, the
privilege on each table or column it touches (`SELECT` on every column a `WHERE` reads,
too), and finally row-level security policies. A superuser passes every gate after login.

A least-privilege setup for TraceLite needs a superuser (or a `CREATEROLE` role) once, so
it can't run in this sandbox. As a provisioning script:

```sql
CREATE ROLE tracelite_owner NOLOGIN;
CREATE ROLE tracelite_migrator LOGIN PASSWORD '...' IN ROLE tracelite_owner;
CREATE ROLE tracelite_app LOGIN PASSWORD '...';

CREATE DATABASE tracelite OWNER tracelite_owner;
\c tracelite
REVOKE ALL ON DATABASE tracelite FROM PUBLIC;          -- CONNECT, TEMPORARY by default
GRANT CONNECT ON DATABASE tracelite TO tracelite_app, tracelite_migrator;
CREATE SCHEMA tracelite AUTHORIZATION tracelite_owner;
GRANT USAGE ON SCHEMA tracelite TO tracelite_app;

ALTER DEFAULT PRIVILEGES FOR ROLE tracelite_owner IN SCHEMA tracelite
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO tracelite_app;
ALTER DEFAULT PRIVILEGES FOR ROLE tracelite_owner IN SCHEMA tracelite
  GRANT USAGE ON SEQUENCES TO tracelite_app;
ALTER DEFAULT PRIVILEGES FOR ROLE tracelite_owner
  REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;
```

Migrations log in as `tracelite_migrator` and start with `SET ROLE tracelite_owner`, so
every table is owned by the `NOLOGIN` role and picks up its default privileges. The
service logs in as `tracelite_app`: it can read and write rows, and can't `DROP`,
`ALTER`, `TRUNCATE`, or create anything. Roles are cluster-wide, so `pg_dump` of the
database doesn't contain them; keep this script (or `pg_dumpall --globals-only`) with
your backups.

## Design notes

- **Who owns, who logs in.** One `NOLOGIN` owner per schema, one login role per kind of
  client: the service, a reporting role with column grants that leave out personal data,
  a monitoring role in `pg_monitor`. KVKK Article 12 obliges a data controller to take
  technical measures against
  [unlawful access to personal data](https://www.kvkk.gov.tr/Icerik/6601/Obligations-Concerning-Data-Security-);
  "the reporting role cannot read the TCKN column" is such a measure, and it is enforced
  by the database rather than by every report author.
- **Grant to groups, not people.** Create `tracelite_readonly` as a `NOLOGIN` role, grant
  privileges to it, and grant the role to logins. Removing a person is then one `REVOKE`.
- **Revoke PUBLIC's defaults** on databases (`CONNECT`, `TEMPORARY`) and functions
  (`EXECUTE`). Since PostgreSQL 15 the `public` schema no longer lets every role create
  objects; a cluster upgraded from an older release, or a restored dump, keeps the old, open grant.
- **StackLab itself** keeps it simple: the app's own `stacklab` role owns the `stacklab`
  database and runs its own migrations, which is acceptable for a local single-user tool.
  The lesson sandbox is where the line is drawn. In local mode `playground` is an ordinary
  role, so these outputs hold. In the optional Docker mode the image creates `playground`
  as its container's superuser: the container is the boundary there, and these examples
  would print `allowed` throughout.
