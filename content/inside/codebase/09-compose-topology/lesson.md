---
id: inside.codebase.compose-topology
title: "The Compose topology: who can talk to whom"
track: inside
module: codebase
order: 9
level: intermediate
minutes: 13
since: es2022
prerequisites: [inside.codebase.runner-sandbox, sql.security-ops.roles]
compare_with: [design.architecture.security, ts.services.realtime-cors, inside.codebase.ts-runner, inside.codebase.at-10x]
tags: [docker-compose, networks, least-privilege, csp, iframe, sandbox, localhost]
summary: >
  A network boundary only protects what it separates. Reading docker-compose.yml as data
  shows the sandboxed runner can reach the app database; local mode, the default, has
  no network boundaries at all apart from loopback, database roles and the browser iframe.
examples:
  - id: reachability
    file: examples/01_reachability.ts
    title: "docker-compose.yml as data: who can reach whom, and the one-line fix"
    runtime: ts-node
    check: exact
    annotations:
      - lines: "2-13"
        note: >
          Everything in docker-compose.yml that decides reachability: each
          service's networks, its published port, and one bind address that
          matters. frontend has no networks key, so Compose puts it on default.
      - lines: "15-24"
        note: >
          Membership is the whole rule. Containers on the same user-defined
          network can reach each other on every port; there is no per-port
          filter. The only extra check is what the process binds to.
      - lines: "16-19"
        note: >
          internal: true removes the route out. A service has egress when at
          least one of its networks is not internal; here that is default.
      - lines: "22"
        note: >
          mocksvc's main.go listens on 127.0.0.1:8091. Inside a container that
          is the container's own loopback, so no other service can connect,
          whatever networks they share.
      - lines: "30"
        note: >
          The finding. runner sits on backend-net, which is how app-db is
          reachable to it, so lesson code in the sandbox can open a connection
          to the learner's data. The password is in the same compose file.
      - lines: "37-41"
        note: >
          Both databases publish a port to 127.0.0.1 while sitting on internal
          networks only. The compose file's own comment says Docker publishes
          nothing for such a container; the host can't use these ports.
      - lines: "43-46"
        note: >
          The backend already reaches the runner over runner-net, so dropping
          backend-net from the runner loses nothing it needs.
  - id: iframe-boundary
    file: examples/02_iframe_boundary.ts
    title: "The tightest boundary in the app: what lesson code in the iframe cannot reach"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "11"
        note: >
          sandbox="allow-scripts" without allow-same-origin gives the frame an
          opaque origin, serialized as "null". It is same-origin with nothing,
          not even another frame built the same way.
      - lines: "13-17"
        note: >
          The frame's CSP is default-src 'none' with no connect-src, so every
          fetch is refused before a packet leaves, including one to the app's
          own backend. Scripts may only be inline or data: URLs, so a CDN import
          fails too.
      - lines: "19-22"
        note: >
          The opaque origin owns no storage and can't see the parent's DOM. Even
          a bug that ran untrusted code here couldn't read the learner's notes
          in localStorage.
      - lines: "24-25"
        note: >
          No 'unsafe-eval' in script-src, so eval and new Function throw. That is
          why the runner loads all code the way the CSP allows: its shim as an
          inline script, the transpiled lesson as a module with a data: URL, and
          the libraries as data: URLs in an import map.
      - lines: "27-28"
        note: >
          The only way out. BrowserTsRunner accepts a message only when
          e.source is its own frame and the data carries __stacklab_run: true;
          this one is ignored.
use_cases:
  - "Reviewing docker-compose.yml before running Docker mode: list what each service must reach, then compare with what it can reach"
  - "Judging what a lesson program could do on your machine in local mode, where it runs as you"
  - "Designing a sandbox for untrusted code in the browser: opaque origin, CSP, a single message channel"
strengths:
  - "Compose networks are declarative: the whole reachability graph is in one file and can be checked as data, unlike host firewall rules spread across machines"
  - "internal: true removes the route out entirely, so the runner and both databases have no internet access in Docker mode"
  - "Every published port binds 127.0.0.1, so nothing in either mode is reachable from the LAN, unlike Docker's default of publishing on all host addresses"
  - "The lesson iframe's boundary is enforced by the browser in both modes: no network, no app storage, no eval"
weaknesses:
  - "In Docker mode the runner shares backend-net with app-db, so the sandbox can reach the app database whose credentials sit in the compose file"
  - "In local mode lesson programs run as the owner with full network access: the internet, the backend API and both databases on 127.0.0.1:5432"
  - "Default credentials (stacklab/stacklab, playground/playground) live in config.go, the setup script and the compose file; fine for 127.0.0.1, wrong anywhere else"
  - "Docker mode is not exercised by any test or by verify-lessons, so the topology above has not been run since ADR-0009 and ADR-0010"
pitfalls:
  - title: "Putting a service on a network for convenience"
    detail: "Membership grants every port to every other member. The runner on backend-net is the example: it never talks to app-db, but it can."
  - title: "127.0.0.1 inside a container"
    detail: "Loopback is per network namespace. A server listening on 127.0.0.1 in a container is reachable only from that container; listen on :8091 (all interfaces of the container) and let the networks decide."
    bad: "http.ListenAndServe(\"127.0.0.1:8091\", mux) // in a container"
    good: "http.ListenAndServe(\":8091\", mux) // reachable only via the networks it joins"
  - title: "Publishing without a host address"
    detail: "\"5432:5432\" publishes on every host address, which can expose the database to the network. This file always writes 127.0.0.1:5432:5432."
  - title: "Trusting the diagram instead of the file"
    detail: "docs/ARCHITECTURE.md says the backend is the only service with an outbound path and reverse-proxies mocksvc. The frontend container has an outbound path too, and there is no /mock route."
quiz:
  - q: "In Docker mode, can lesson code in the runner container connect to app-db:5432?"
    options: ["No: the runner is on an internal network", "Yes: runner and app-db both sit on backend-net", "Only through the backend"]
    answer: 1
    why: "internal: true blocks traffic to other networks, not between members. Both services are members of backend-net."
  - q: "The backend and mocksvc share two networks. Why can't the backend reach http://mocksvc:8091?"
    options: ["Internal networks block HTTP", "mocksvc listens on 127.0.0.1, which inside its container is its own loopback", "Port 8091 isn't published"]
    answer: 1
    why: "Publishing is for the host. Between containers, what matters is a shared network and a process listening on an address that network reaches."
  - q: "In local mode, what stops a Go lesson from connecting to the stacklab database?"
    options: ["The internal network", "The playground role", "Nothing technical: same machine, and the default credentials are in config.go"]
    answer: 2
    why: "Lesson SQL from the sql runtimes runs as playground, but a Go program opens its own connections as whoever it likes. ADR-0003's threat model accepts this for a single-user, local-only app."
  - q: "Why does fetch('/healthz') fail inside a ts-browser lesson even though the app serves it?"
    options: ["The URL is relative", "The frame's CSP leaves connect-src at default-src 'none'", "The backend sends no CORS headers"]
    answer: 1
    why: "The CSP blocks the request before it is sent. CORS would only matter if the CSP let it through."
further_reading:
  - title: "PostgreSQL: listen_addresses (connection settings)"
    url: "https://www.postgresql.org/docs/current/runtime-config-connection.html"
  - title: "PostgreSQL: the pg_hba.conf file"
    url: "https://www.postgresql.org/docs/current/auth-pg-hba-conf.html"
  - title: "MDN: CSP default-src"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Content-Security-Policy/default-src"
  - title: "MDN: <iframe> sandbox"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/iframe"
  - title: "MDN: Same-origin policy"
    url: "https://developer.mozilla.org/en-US/docs/Web/Security/Same-origin_policy"
---

## Concept

A network boundary is a list of who may open a connection to whom. It is coarse and
cheap, and it protects only what it actually separates. StackLab has three topologies,
and the one you run by default has the fewest boundaries.

**Local mode** (ADR-0008, `npm start`). One machine: Vite on `127.0.0.1:5173`, the Go
backend on `127.0.0.1:8080`, one PostgreSQL service holding both databases. Lesson Go and
Node programs are child processes of the backend, running as you, with your network.
The boundaries that remain:

- **Bind addresses.** Everything listens on loopback. `scripts/setup-local-db.sql` sets
  PostgreSQL's `listen_addresses = 'localhost'`, because the Windows installer listens on
  every interface.
- **Database roles.** SQL from the `sql*` runtimes runs as `playground`, a
  non-superuser (sql.security-ops.roles), so it can't run `COPY ... TO PROGRAM`.
- **The browser sandbox** for ts-browser lessons (example 2).

Nothing separates a lesson program from the app: a Go lesson can call the backend API,
reach the internet, or connect to the `stacklab` database with the default credentials
in `server/internal/config/config.go`. ADR-0003's threat model, protecting the owner from
their own mistakes rather than from an attacker, is what makes that acceptable.

**Docker mode** (`docker-compose.yml`, optional). Six services, three networks:
`default`, plus `backend-net` and `runner-net`, both `internal: true`. Example 1 reads
the file as data:

- `frontend` reaches `backend` over `default` (Vite proxies `/api` to
  `http://backend:8080`).
- `backend` is on all three networks: app-db over `backend-net`, the runner over
  `runner-net`, the internet over `default`.
- `runner` needs `playground-db` (over `runner-net`) and nothing else. It also sits on
  `backend-net`, which puts `app-db` within its reach. That is the boundary this
  topology was meant to draw, and it isn't drawn.
- `mocksvc` binds `127.0.0.1:8091` inside its container, so no other service can reach
  it. Nothing calls it yet (ADR-0008).

## Coming from C/C++/Java/Python/JS

- **Cloud networking**: a Compose network is a security group with one rule, "members may
  talk to members on any port". Kubernetes NetworkPolicy can filter by port and
  direction; Compose can't.
- **Java/Python services**: `localhost` in a container means the container itself. A
  server that binds `127.0.0.1` is unreachable from its neighbors, which is the opposite
  of what you want for a service and exactly what you want for a debug port.
- **JS**: the browser has the most precise boundary in this app. Same-origin policy
  decides what a page may read, CSP decides what it may load or connect to, and
  `sandbox` decides its origin. ts.services.realtime-cors covers CORS, which only comes
  into play when the CSP lets a request through.
- **C/C++**: think of `bind()` addresses and `iptables` rules. Compose generates the
  rules from network membership; you never write them.

## How it works

What the Docker documentation guarantees, and what the examples rely on:

- [Compose networks](https://docs.docker.com/reference/compose-file/networks/):
  `internal: true` creates "an externally isolated network", and services without a
  `networks` key join the implicit `default` network.
- [`docker network create --internal`](https://docs.docker.com/reference/cli/docker/network/create/):
  "Containers on an internal network may communicate between each other, but not with
  any other network", and firewall rules drop traffic to or from other networks.
- [Bridge networks](https://docs.docker.com/engine/network/drivers/bridge/): containers
  on the same user-defined bridge "effectively expose all ports to each other".
- [Port publishing](https://docs.docker.com/engine/network/port-publishing/): publishing
  to `127.0.0.1` means "only the Docker host can access the published container port";
  without an address, Docker publishes on all host addresses.

What the documentation doesn't settle: `app-db` and `playground-db` publish
`127.0.0.1:5432` and `:5433` but sit on internal networks only. The comment on the
backend's `networks` in the compose file says "Docker publishes nothing for a container
that sits on internal networks only", and ADR-0008 lists "internal-network port
publishing" among the reasons local mode replaced Docker. If that holds, the two
published database ports are dead, and `psql` from the host can't reach them.
<!-- VERIFY: Docker's network docs say internal networks drop traffic to/from other networks but do not state that published ports of an internal-only container are unreachable; the claim rests on the compose comment and ADR-0008. -->

The lesson iframe (example 2) is built by `BrowserTsRunner`: `sandbox="allow-scripts"`
without `allow-same-origin` (opaque origin), and a CSP of
`default-src 'none'; script-src 'unsafe-inline' data:; worker-src blob:; img-src data: blob:; style-src 'unsafe-inline'`.
`connect-src` isn't listed, so it falls back to `default-src 'none'`.

## Design notes

Least privilege for networks works like least privilege for database roles: list what
each service must reach, then give it exactly the networks that cover that list. For this
file:

- `runner` on `runner-net` only (the fix in example 1). The backend still reaches it there.
- Better still, a third internal network for the sandbox's own dependencies:
  `runner-net` for backend and runner, `sandbox-net` for runner, `playground-db` and
  `mocksvc`. Then the backend can't reach the playground database either, which it
  never needs in Docker mode.
- `mocksvc` listening on `:8091`, once something calls it.

The documentation has drifted from the file. `docs/ARCHITECTURE.md` says the backend is
"the only service with an outbound path" (the `frontend` container is on `default` too,
which it needs for `npm install`) and that the backend reverse-proxies `/mock/*` to
mocksvc (the router has no such route). The compose comment says internal members have
no published ports "except where the owner inspects them directly", which describes the
two database ports that probably can't be reached.

Docker mode has not been run since today's runner changes (ADR-0009, ADR-0010), and
nothing tests it. One thing to fix before relying on it: the final stage of
`runner/Dockerfile` is the Node image plus the runner binary, `ca-certificates` and
`curl`. No step installs the Go toolchain the runner invokes as `go`, although the
stage's comment says one does. This is read from the Dockerfile and `gorun.go`, not run.
What a Go lesson would get follows from the code: `goBuild` and `go-test` both start
`go` through `runProcess`, which appends the start error, so a `go` or `go-server`
lesson would fail with exit code 1 (a `go-test` one with -1) and the stderr line
`gorun: exec: "go": executable file not found in $PATH`.

In local mode the network is not the boundary, and pretending otherwise would be
dishonest. What holds is that nothing listens beyond 127.0.0.1. If the app ever leaves
this machine, ADR-0003 says the runner must be rebuilt (gVisor, Firecracker or a
separate host), and the first network rule to add is the one example 1 is missing:
lesson code must not be able to reach the app's data.
