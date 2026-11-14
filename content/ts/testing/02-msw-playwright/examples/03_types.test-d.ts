import { describe, expectTypeOf, test } from "vitest";

type City = { name: string; plate: number };
type Result<T> = { ok: true; data: T } | { ok: false; error: string };

// A typed API client: the path you pass decides the response type.
type Routes = { "/cities": City[]; "/cities/:plate": City };
declare function get<P extends keyof Routes>(
  path: P,
): Promise<Result<Routes[P]>>;

function pick<T, K extends keyof T>(obj: T, ...keys: K[]): Pick<T, K> {
  const out = {} as Pick<T, K>;
  for (const k of keys) out[k] = obj[k];
  return out;
}

describe("client types", () => {
  test("the path decides the response type", () => {
    expectTypeOf(get("/cities")).resolves.toEqualTypeOf<Result<City[]>>();
    expectTypeOf(get("/cities/:plate")).resolves.toEqualTypeOf<Result<City>>();
    expectTypeOf(get)
      .parameter(0)
      .toEqualTypeOf<"/cities" | "/cities/:plate">();
    // @ts-expect-error -- "/towns" is not a route
    get("/towns");
  });

  test("data exists only after checking ok", () => {
    const r = {} as Result<City>;
    expectTypeOf(r).not.toHaveProperty("data");
    if (r.ok) expectTypeOf(r.data).toEqualTypeOf<City>();
    else expectTypeOf(r.error).toBeString();
  });

  test("pick keeps exactly the named keys", () => {
    const izmir: City = { name: "İzmir", plate: 35 };
    expectTypeOf(pick(izmir, "name")).toEqualTypeOf<{ name: string }>();
    // @ts-expect-error -- "region" is not a key of City
    pick(izmir, "region");
  });

  test("toExtend allows extra properties, toEqualTypeOf does not", () => {
    const row = { name: "Iğdır", plate: 76, region: "Doğu Anadolu" };
    expectTypeOf(row).toExtend<City>();
    expectTypeOf(row).not.toEqualTypeOf<City>();
  });
});
