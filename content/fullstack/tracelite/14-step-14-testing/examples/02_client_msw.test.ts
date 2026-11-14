// LEARN(fullstack.tracelite.step-14-testing): each layer gets the test that can see its bugs; the client's is real fetch against MSW answers.
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { afterAll, afterEach, beforeAll, expect, it } from "vitest";
import { z } from "zod";

// --- src/api, abridged from step 5: every answer is checked at the boundary.
const Requirement = z.object({
  code: z.string(),
  parentId: z.uuid().nullable(),
});
const FieldError = z.object({ pointer: z.string(), detail: z.string() });
const Problem = z.object({
  type: z.string(),
  title: z.string(),
  status: z.number(),
  errors: z.array(FieldError).optional(),
});

class ApiError extends Error {
  constructor(readonly problem: z.infer<typeof Problem>) {
    super(`${problem.status} ${problem.title}`);
  }
}

async function request<T>(
  schema: z.ZodType<T>,
  path: string,
  init: RequestInit,
) {
  // A browser resolves "/api/..." against the page; Node's fetch needs it done.
  const res = await fetch(new URL(path, location.href), init);
  const body: unknown = await res.json().catch(() => null);
  if (!res.ok) {
    const { status, statusText: title } = res;
    const made = { type: "about:blank", title, status };
    throw new ApiError(Problem.safeParse(body).data ?? made);
  }
  return schema.parse(body);
}

const save = () =>
  request(Requirement, "/api/projects/P1/requirements", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ code: "AUTH", parentId: null }),
  });

// --- The tests: MSW answers as the Go server, or a proxy before it, would.
const server = setupServer();
beforeAll(() => server.listen({ onUnhandledFrame: "error" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());
const answer = (res: Response) =>
  server.use(http.post("*/api/projects/:projectId/requirements", () => res));

it("passes the server's 409 problem on, field pointer included", async () => {
  const conflict = {
    type: "https://tracelite.example/problems/conflict",
    title: "Conflict",
    status: 409,
    errors: [{ pointer: "#/code", detail: "bu kod projede zaten var" }],
  };
  answer(HttpResponse.json(conflict, { status: 409 }));
  await expect(save()).rejects.toMatchObject({ problem: conflict });
});

it("makes a problem out of the proxy's 502 page", async () => {
  const page = "<html><body>Bad Gateway</body></html>";
  answer(new HttpResponse(page, { status: 502, statusText: "Bad Gateway" }));
  const problem = { type: "about:blank", title: "Bad Gateway", status: 502 };
  await expect(save()).rejects.toMatchObject({ problem });
});

it("refuses a 201 whose body broke the contract", async () => {
  // A Go tag typo, json:"parentID": the member the spec names is missing.
  answer(HttpResponse.json({ code: "AUTH", parentID: null }, { status: 201 }));
  await expect(save()).rejects.toThrow(z.ZodError);
});
