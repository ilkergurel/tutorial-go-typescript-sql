// LEARN(ts.browser-dom.storage): filters belong in the URL: shareable, bookmarkable, parsed with URLSearchParams.

interface Filters {
  q: string;
  cities: string[];
  page: number;
  sort: "name" | "date";
}

// Parse defensively: anyone can type anything into the address bar.
function readFilters(params: URLSearchParams): Filters {
  const page = Number(params.get("page") ?? 1);
  return {
    q: params.get("q") ?? "",
    cities: params.getAll("city"), // repeated keys are a list
    page: Number.isInteger(page) && page > 0 ? page : 1,
    sort: params.get("sort") === "date" ? "date" : "name",
  };
}

// Write only what differs from the defaults: shorter, stabler URLs.
function writeFilters(f: Filters): URLSearchParams {
  const params = new URLSearchParams();
  if (f.q) params.set("q", f.q);
  for (const city of f.cities) params.append("city", city);
  if (f.page !== 1) params.set("page", String(f.page));
  if (f.sort !== "name") params.set("sort", f.sort);
  return params;
}

// In a page this is location.href; the sandbox's own URL is about:srcdoc.
const url = new URL(
  "https://example.com/orders?q=Ay%C5%9Fe+Y%C4%B1lmaz&city=%C4%B0zmir&page=abc",
);
const filters = readFilters(url.searchParams);
console.log(JSON.stringify(filters));

const next: Filters = {
  ...filters,
  cities: [...filters.cities, "Muğla"],
  page: 2,
};
url.search = writeFilters(next).toString();
console.log(url.href);
// On a page: history.replaceState(null, "", url) updates the address bar
// without a reload; pushState adds a Back-button entry.

// Two encodings of a space: form encoding (+) and percent encoding (%20).
console.log(new URLSearchParams({ q: "Ayşe Yılmaz" }).toString());
console.log(encodeURIComponent("Ayşe Yılmaz"));

// Resolve a relative link against a base, like the browser does for <a href>.
console.log(
  new URL("../export?format=csv", "https://example.com/app/orders/").href,
);
