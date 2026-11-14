import { z } from "zod";

// --- src/api/schemas.ts, abridged from example 1.
const Requirement = z.object({ code: z.string(), createdAt: z.iso.datetime() });
const FieldError = z.object({ pointer: z.string(), detail: z.string() });
const Problem = z.object({
  type: z.string(),
  title: z.string(),
  status: z.number().int(),
  errors: z.array(FieldError).optional(),
});
type Problem = z.infer<typeof Problem>;

// --- src/api/client.ts: responses are parsed; failures become an ApiError.
class ApiError extends Error {
  readonly problem: Problem;
  constructor(problem: Problem) {
    super(`${problem.status} ${problem.title}`);
    this.problem = problem;
  }
}

async function request<T>(
  schema: z.ZodType<T>,
  path: string,
  init?: RequestInit,
): Promise<T> {
  const headers = { "Content-Type": "application/json" };
  const res = await fetch(path, { ...init, headers });
  const body: unknown = await res.json().catch(() => null);
  if (!res.ok) {
    const parsed = Problem.safeParse(body);
    const { status } = res;
    const title = res.statusText || `HTTP ${status}`; // HTTP/2 sends none
    throw new ApiError(
      parsed.success ? parsed.data : { type: "about:blank", title, status },
    );
  }
  return schema.parse(body); // a drifted server fails here, at the boundary
}

const createRequirement = (projectId: string, code: string, title: string) =>
  request(Requirement, `/api/projects/${projectId}/requirements`, {
    method: "POST",
    body: JSON.stringify({ code, title }),
  });

// --- No network in the lesson iframe: fetch answers as the server would.
window.fetch = async (_path, init) => {
  const { code } = JSON.parse(String(init?.body));
  if (code === "AUTH-LOGİN") {
    const type = "https://tracelite.example/problems/validation";
    const errors = [{ pointer: "#/code", detail: "use A-Z, 0-9 and hyphens" }];
    const problem = { type, title: "Validation failed", status: 422, errors };
    return Response.json(problem, { status: 422 });
  }
  if (code === "AUTH") {
    return new Response("<h1>Bad Gateway</h1>", { status: 502 }); // a proxy
  }
  // A Go struct field that lost its json tag: the member's name drifted.
  const created = { code, CreatedAt: "2026-11-13T06:30:00Z" };
  return Response.json(created, { status: 201 });
};

const project = "01937b6e-0000-7000-8000-00000000000a";
const codes = ["auth-login".toLocaleUpperCase("tr"), "AUTH", "AUTH-2FA"];
for (const code of codes) {
  try {
    await createRequirement(project, code, "İki adımlı doğrulama");
  } catch (e) {
    if (e instanceof ApiError) {
      const errors = JSON.stringify(e.problem.errors ?? []);
      console.log(`${code}: ApiError ${e.message} ${errors}`);
    } else if (e instanceof z.ZodError) {
      console.log(`${code}: the response broke the contract`);
      console.log(z.prettifyError(e));
    } else throw e;
  }
}
