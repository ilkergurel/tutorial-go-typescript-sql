import { expect, test } from "@playwright/test";

// The app under test: a static page whose script calls /api/cities.
const html = `<!doctype html><meta charset="utf-8">
<form><label>Region <input name="region"></label> <button>Search</button></form>
<ul></ul><p role="alert" hidden></p>
<script>
  document.querySelector("form").onsubmit = async (e) => {
    e.preventDefault();
    const region = new FormData(e.target).get("region");
    const res = await fetch("/api/cities?region=" + encodeURIComponent(region));
    const alert = document.querySelector("[role=alert]");
    alert.hidden = res.ok;
    alert.textContent = res.ok ? "" : "Search failed (" + res.status + ")";
    const cities = res.ok ? await res.json() : [];
    document.querySelector("ul").replaceChildren(...cities.map((c) =>
      Object.assign(document.createElement("li"), { textContent: c.plate + " " + c.name })));
  };
</script>`;

// No server: Playwright answers the browser's requests itself.
test.beforeEach(async ({ page }) => {
  await page.route("https://app.test/", (route) =>
    route.fulfill({ contentType: "text/html", body: html }),
  );
});

test("search lists the cities the API returns", async ({ page }) => {
  await page.route("https://app.test/api/**", async (route) => {
    const url = new URL(route.request().url());
    const ege = url.searchParams.get("region") === "Ege";
    await route.fulfill({ json: ege ? [{ name: "İzmir", plate: 35 }] : [] });
  });
  await page.goto("https://app.test/");
  await page.getByLabel("Region").fill("Ege");
  await page.getByLabel("Region").press("Enter");
  await expect(page.getByRole("listitem")).toHaveText(["35 İzmir"]);
  await expect(page.getByRole("alert")).toBeHidden();
});

test("a failing API shows an alert and no list", async ({ page }) => {
  await page.route("https://app.test/api/**", (route) =>
    route.fulfill({ status: 503 }),
  );
  await page.goto("https://app.test/");
  await page.getByRole("button", { name: "Search" }).click();
  await expect(page.getByRole("alert")).toHaveText("Search failed (503)");
  await expect(page.getByRole("listitem")).toHaveCount(0);
});
