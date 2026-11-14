// LEARN(ts.services.realtime-cors): the server labels a response; the browser decides whether this page may read it.

// Real facts about this page: a sandboxed iframe has an opaque origin.
console.log(`origin: ${self.origin}`);
try {
  console.log(document.cookie);
} catch (e) {
  console.log(`document.cookie: ${e instanceof DOMException ? e.name : e}`);
}
const req = new Request("https://api.example.com/orders");
console.log(`fetch defaults: mode ${req.mode}, credentials ${req.credentials}`);

// A simplified model of the browser's two decisions in the Fetch standard.
// 1. Preflight? Yes for anything an HTML form couldn't have sent.
const FORM_TYPES = [
  "application/x-www-form-urlencoded",
  "multipart/form-data",
  "text/plain",
];
function preflight(method: string, headers: Record<string, string>) {
  if (!["GET", "HEAD", "POST"].includes(method)) return true;
  return Object.entries(headers).some(([name, value]) => {
    const n = name.toLowerCase();
    if (n !== "content-type") {
      return !["accept", "accept-language", "content-language"].includes(n);
    }
    const type = value.split(";")[0].trim().toLowerCase();
    return !FORM_TYPES.includes(type);
  });
}
const requests: [string, Record<string, string>][] = [
  ["GET", { Accept: "application/json" }],
  ["POST", { "Content-Type": "multipart/form-data; boundary=x" }],
  ["POST", { "Content-Type": "application/json" }],
  ["GET", { Authorization: "Bearer abc" }],
  ["DELETE", {}],
];
for (const [method, headers] of requests) {
  const how = preflight(method, headers) ? "OPTIONS first" : "sent directly";
  console.log(`${how}: ${method} ${JSON.stringify(headers)}`);
}

// 2. May the page read the response? By now the request has happened.
const ALLOW = "access-control-allow-origin";
const CREDS = "access-control-allow-credentials";
function mayRead(origin: string, cookies: boolean, h: Record<string, string>) {
  if (!(ALLOW in h)) return false;
  if (h[ALLOW] === "*") return !cookies; // never with credentials
  if (h[ALLOW] !== origin) return false;
  return !cookies || h[CREDS] === "true";
}
const app = "https://app.example.com";
const responses: [string, boolean, Record<string, string>][] = [
  [app, false, {}],
  [app, false, { [ALLOW]: "*" }],
  [app, true, { [ALLOW]: "*" }],
  [app, true, { [ALLOW]: app }],
  [app, true, { [ALLOW]: app, [CREDS]: "true" }],
  [self.origin, false, { [ALLOW]: app }],
  [self.origin, false, { [ALLOW]: "*" }],
  [self.origin, false, { [ALLOW]: "null" }],
];
for (const [origin, cookies, h] of responses) {
  const verdict = mayRead(origin, cookies, h) ? "read   " : "blocked";
  const from = `${origin}${cookies ? " with cookies" : ""}`;
  console.log(`${verdict} ${from}: ${JSON.stringify(h)}`);
}
