// LEARN(ts.services.fetch): fetch resolves for every HTTP status; only a failed request rejects.

// A fake server with fetch's exact signature, answering with real Response
// objects: the lesson sandbox has no network.
const fakeFetch: typeof fetch = async (input) => {
  const { pathname } = new URL(String(input));
  if (pathname === "/users/7") return Response.json({ id: 7, name: "Şule" });
  if (pathname === "/users/8") {
    return Response.json({ error: "user 8 not found" }, { status: 404 });
  }
  // The API is down, and the proxy in front of it answers instead.
  return new Response("<h1>502 Bad Gateway</h1>", {
    status: 502,
    headers: { "Content-Type": "text/html" },
  });
};
const api = (id: number) => `https://api.example.com/users/${id}`;
const nameOf = (e: unknown) => (e instanceof Error ? e.name : String(e));

interface User {
  id: number;
  name: string;
}

// Naive: whatever arrives is treated as the happy path.
async function getUserNaive(id: number): Promise<User> {
  const res = await fakeFetch(api(id));
  return res.json(); // Promise<any>: compiles whatever the body is
}

class HttpError extends Error {
  readonly status: number;
  constructor(status: number, body: string) {
    super(`HTTP ${status}: ${body.slice(0, 40)}`);
    this.name = "HttpError";
    this.status = status;
  }
}

async function getUser(id: number): Promise<User> {
  const res = await fakeFetch(api(id));
  // An error body may be JSON, HTML or empty: read it as text.
  if (!res.ok) throw new HttpError(res.status, await res.text());
  return (await res.json()) as User; // still a cast: the zod lesson fixes it
}

const variants = { naive: getUserNaive, checked: getUser };
for (const [label, get] of Object.entries(variants)) {
  for (const id of [7, 8, 9]) {
    try {
      console.log(`${label} ${id}: name=${(await get(id)).name}`);
    } catch (e) {
      const why = e instanceof HttpError ? e.message : nameOf(e);
      console.log(`${label} ${id}: ${why}`);
    }
  }
}

// A real fetch: the sandbox's CSP blocks the network, so there is no response.
try {
  await fetch(api(7));
} catch (e) {
  console.log(`real fetch: ${nameOf(e)}`);
}
