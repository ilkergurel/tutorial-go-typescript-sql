// LEARN(ts.services.openapi): generated types describe every path, parameter and response; one generic client turns them into checked calls.

// What openapi-typescript generates from the spec, abbreviated: the real file
// also lists headers, query?: never, put?: never, ... and an operations type.
interface components {
  schemas: {
    Order: { id: number; customer: string; status: "open" | "shipped" };
    Problem: { title: string; status: number; detail?: string };
  };
}
type Schemas = components["schemas"];
interface paths {
  "/orders/{id}": {
    get: {
      parameters: { path: { id: number } };
      responses: {
        200: { content: { "application/json": Schemas["Order"] } };
        404: { content: { "application/problem+json": Schemas["Problem"] } };
      };
    };
  };
}

// The client is written once, against whatever paths type was generated.
type Get<P extends keyof paths> = paths[P]["get"];
type Responses<P extends keyof paths> = Get<P>["responses"];
// One member per documented status, so checking the status types the body.
type Reply<P extends keyof paths> = {
  [S in keyof Responses<P>]: {
    status: S;
    body: Responses<P>[S] extends { content: infer C } ? C[keyof C] : never;
  };
}[keyof Responses<P>];

async function get<P extends keyof paths>(
  doFetch: typeof fetch,
  path: P,
  params: Get<P>["parameters"]["path"],
): Promise<Reply<P>> {
  const values: Record<string, unknown> = params;
  const url = path.replace(/\{(\w+)\}/g, (_, name: string) =>
    encodeURIComponent(String(values[name])),
  );
  const res = await doFetch(`https://api.example.com${url}`);
  // The one cast: the contract says which body goes with which status.
  return { status: res.status, body: await res.json() } as Reply<P>;
}

const server: typeof fetch = async (input) => {
  const id = Number(String(input).split("/").pop());
  if (id === 7) return Response.json({ id, customer: "Ayşe", status: "open" });
  const problem = { title: "Not Found", status: 404, detail: `order ${id}` };
  return Response.json(problem, {
    status: 404,
    headers: { "Content-Type": "application/problem+json" },
  });
};

for (const id of [7, 8]) {
  const reply = await get(server, "/orders/{id}", { id });
  // reply.body.customer here: TS2339, body is still Order | Problem.
  if (reply.status === 200) {
    console.log(`200: ${reply.body.customer}, ${reply.body.status}`);
  } else {
    console.log(`${reply.status}: ${reply.body.title} (${reply.body.detail})`);
  }
}

// Rejected by tsc, so never sent:
// get(server, "/order/{id}", { id: 7 });
//   TS2345: Argument of type '"/order/{id}"' is not assignable to
//   parameter of type '"/orders/{id}"'.
// get(server, "/orders/{id}", { id: "7" });
//   TS2322: Type 'string' is not assignable to type 'number'.
