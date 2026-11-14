// LEARN(inside.codebase.compose-topology): a Compose topology is data; who can reach whom is a set intersection you can check.
// docker-compose.yml, reduced to what decides reachability.
type Service = { networks: string[]; published?: string; listen?: string };
const internalNets = new Set(["backend-net", "runner-net"]); // internal: true

const compose: Record<string, Service> = {
  "app-db": { networks: ["backend-net"], published: "127.0.0.1:5432" },
  "playground-db": { networks: ["runner-net"], published: "127.0.0.1:5433" },
  backend: { networks: ["default", "backend-net", "runner-net"], published: "127.0.0.1:8080" },
  frontend: { networks: ["default"], published: "127.0.0.1:5173" }, // no networks key: default
  runner: { networks: ["runner-net", "backend-net"] },
  mocksvc: { networks: ["runner-net", "backend-net"], listen: "127.0.0.1:8091" },
};

function reach(c: Record<string, Service>, from: string, to: string): string {
  if (to === "internet") {
    const out = c[from].networks.filter((n) => !internalNets.has(n));
    return out.length ? `yes, via ${out.join(", ")}` : "no";
  }
  const shared = c[from].networks.filter((n) => c[to].networks.includes(n));
  if (shared.length === 0) return "no";
  if (c[to].listen?.startsWith("127.")) return `no: shares ${shared.join(", ")}, but listens on ${c[to].listen}`;
  return `yes, via ${shared.join(", ")}`;
}

const questions: [string, string][] = [
  ["frontend", "backend"],
  ["frontend", "app-db"],
  ["runner", "playground-db"],
  ["runner", "app-db"],
  ["backend", "mocksvc"],
  ["runner", "internet"],
  ["frontend", "internet"],
];
for (const [from, to] of questions) console.log(`${from} -> ${to}: ${reach(compose, from, to)}`);

for (const [name, s] of Object.entries(compose)) {
  if (s.published && s.networks.every((n) => internalNets.has(n))) {
    console.log(`${name} publishes ${s.published} but sits on internal networks only`);
  }
}

// The fix for the finding above: the runner needs runner-net only.
const fixed = { ...compose, runner: { networks: ["runner-net"] } };
console.log("after the fix:");
for (const to of ["app-db", "backend", "playground-db"]) console.log(`  runner -> ${to}: ${reach(fixed, "runner", to)}`);
