import { cleanup, render, screen } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { useEffect, useState } from "react";
import { afterAll, afterEach, beforeAll, describe, expect, it } from "vitest";

type City = { name: string; plate: number };
const API = "https://api.example.test";

// Production code: plain fetch, nothing test-specific.
async function fetchCities(region: string): Promise<City[]> {
  const res = await fetch(`${API}/cities?region=${encodeURIComponent(region)}`);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
}

function Cities({ region }: { region: string }) {
  const [cities, setCities] = useState<City[]>();
  const [error, setError] = useState("");
  useEffect(() => {
    fetchCities(region).then(setCities, (e: Error) => setError(e.message));
  }, [region]);
  if (error) return <p role="alert">No cities ({error})</p>;
  if (!cities) return <p>Loading…</p>;
  return (
    <ul>
      {cities.map((c) => (
        <li key={c.plate}>{`${c.plate} ${c.name}`}</li>
      ))}
    </ul>
  );
}

// The network as the tests see it: a handler per endpoint.
const server = setupServer(
  http.get(`${API}/cities`, ({ request }) => {
    const region = new URL(request.url).searchParams.get("region");
    if (region !== "Güneydoğu Anadolu") return HttpResponse.json([]);
    return HttpResponse.json([
      { name: "Şanlıurfa", plate: 63 },
      { name: "Mardin", plate: 47 },
    ]);
  }),
);
beforeAll(() => server.listen({ onUnhandledFrame: "error" }));
afterEach(() => {
  cleanup();
  server.resetHandlers(); // drop the overrides a test added
});
afterAll(() => server.close());

describe("Cities", () => {
  it("lists the cities the API returns", async () => {
    render(<Cities region="Güneydoğu Anadolu" />);
    const items = await screen.findAllByRole("listitem");
    expect(items.map((li) => li.textContent)).toEqual([
      "63 Şanlıurfa",
      "47 Mardin",
    ]);
  });

  it("shows an error when the API fails", async () => {
    server.use(
      http.get(`${API}/cities`, () => new HttpResponse(null, { status: 503 })),
    );
    render(<Cities region="Ege" />);
    expect((await screen.findByRole("alert")).textContent).toBe(
      "No cities (HTTP 503)",
    );
  });

  it("fails loudly on a request nobody mocked", async () => {
    await expect(fetch(`${API}/weather`)).rejects.toThrow("fetch failed");
  });
});
