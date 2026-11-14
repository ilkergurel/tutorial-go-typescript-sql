// LEARN(inside.codebase.compose-topology): the lesson iframe is the app's tightest network boundary: opaque origin, CSP, one way out.
async function attempt(what: string, f: () => unknown): Promise<void> {
  try {
    await f();
    console.log(`${what}: allowed`);
  } catch (e) {
    console.log(`${what}: blocked (${(e as Error).name})`);
  }
}

console.log(`this frame's origin: ${self.origin}`);

// Network: the CSP is default-src 'none', so connect-src is 'none' too.
await attempt("fetch the app's own /healthz", () => fetch("/healthz"));
await attempt("fetch https://pkg.go.dev/", () => fetch("https://pkg.go.dev/"));
const cdn = "https://cdn.jsdelivr.net/npm/zod/+esm";
await attempt("import a module from a CDN", () => import(cdn));

// Same-origin data: the opaque origin owns none of the app's state.
await attempt("read parent.document", () => parent.document.title);
await attempt("read localStorage", () => localStorage.length);
await attempt("read document.cookie", () => document.cookie);

// Code: no 'unsafe-eval', so strings can't become code at run time.
await attempt("eval a string", () => eval("6 * 7"));

// The one channel left: messages to the parent, which drops any it doesn't expect.
await attempt("postMessage to the parent", () => parent.postMessage({ hello: "app" }, "*"));
