// LEARN(inside.codebase.ts-runner): one interface, two runners, one factory; the caller never branches on where code runs.
type Runtime = "go" | "sql" | "ts-browser";
interface RunRequest { runtime: Runtime; files: Record<string, string> }
interface RunResult { stdout: string; exitCode: number }

interface CodeRunner {
  run(req: RunRequest): Promise<RunResult>;
}

class ServerRunner implements CodeRunner {
  constructor(private readonly post: typeof fetch) {}
  async run(req: RunRequest): Promise<RunResult> {
    const res = await this.post("/api/run", { method: "POST", body: JSON.stringify(req) });
    if (!res.ok) {
      const body = (await res.json().catch(() => ({}))) as { detail?: string };
      throw new Error(body.detail || res.statusText);
    }
    return (await res.json()) as RunResult;
  }
}

class BrowserRunner implements CodeRunner {
  async run(req: RunRequest): Promise<RunResult> {
    const lines: string[] = [];
    const log = console.log;
    console.log = (...args: unknown[]) => void lines.push(args.join(" "));
    try {
      await import("data:text/javascript," + encodeURIComponent(Object.values(req.files)[0]));
      return { stdout: lines.join("\n"), exitCode: 0 };
    } catch (e) {
      return { stdout: [...lines, String(e)].join("\n"), exitCode: 1 };
    } finally {
      console.log = log;
    }
  }
}

// The backend, faked: Go runs succeed, the SQL run is refused (both slots busy).
const backend: typeof fetch = async (_url, init) => {
  const req = JSON.parse(String(init?.body)) as RunRequest;
  return req.runtime === "sql"
    ? new Response(JSON.stringify({ status: 503, detail: "server busy" }), { status: 503 })
    : new Response(JSON.stringify({ stdout: "merhaba from go", exitCode: 0 }));
};

const server = new ServerRunner(backend);
const browser = new BrowserRunner();
const runnerFor = (runtime: Runtime): CodeRunner => (runtime === "ts-browser" ? browser : server);

const presses: [Runtime, string][] = [
  ["go", "package main ..."],
  ["ts-browser", "console.log(6 * 7)"],
  ["ts-browser", "null.x"],
  ["sql", "SELECT 1"],
];
for (const [runtime, code] of presses) {
  try {
    const r = await runnerFor(runtime).run({ runtime, files: { main: code } });
    console.log(`${runtime.padEnd(10)} done   exit ${r.exitCode}: ${r.stdout.replace("\n", " | ")}`);
  } catch (err) {
    console.log(`${runtime.padEnd(10)} failed ${(err as Error).message}`);
  }
}
