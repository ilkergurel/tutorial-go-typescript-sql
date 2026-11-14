import { expect, test } from "@playwright/test";
import { readFile } from "node:fs/promises";
import { createServer, type Server } from "node:http";
import type { AddressInfo } from "node:net";

const P = "01937b6e-0000-7000-8000-00000000000a";
const exportPath = `/api/projects/${P}/requirement-export`;

// Stand-in for the project page: its export link, as step 8 renders it.
const html = `<!doctype html><meta charset="utf-8"><title>Ödeme</title>
<h1>Ödeme</h1>
<a href="${exportPath}" download>CSV indir</a>`;

// Step 8's response: UTF-8 with BOM, ';', CRLF, and the RFC 6266 file name.
const csv =
  "﻿code;title;parent;status;description\r\n" +
  "AUTH;Kimlik doğrulama;;approved;\r\n" +
  "AUTH-RESET;Parola sıfırlama;AUTH;draft;\r\n";
const disposition =
  "attachment; filename*=utf-8''%C3%96deme%20gereksinimleri.csv";

// A real HTTP server on a free port, standing in for cmd/tracelite. In Edge,
// page.route never sees a download started by <a download> (Playwright #22650).
let server: Server;
let origin = "";
test.beforeAll(async () => {
  server = createServer((req, res) => {
    if (req.url !== exportPath) {
      res.writeHead(200, { "Content-Type": "text/html; charset=utf-8" });
      return res.end(html);
    }
    res.writeHead(200, {
      "Content-Type": "text/csv; charset=utf-8",
      "Content-Disposition": disposition,
    });
    res.end(csv);
  });
  await new Promise<void>((ready) => server.listen(0, "127.0.0.1", ready));
  origin = `http://127.0.0.1:${(server.address() as AddressInfo).port}`;
});
test.afterAll(() => new Promise<void>((done) => server.close(() => done())));

test("the CSV export arrives as Ödeme gereksinimleri.csv, BOM and all", async ({
  page,
}) => {
  await page.goto(`${origin}/projects/${P}`);
  const download = page.waitForEvent("download");
  await page.getByRole("link", { name: "CSV indir" }).click();
  const file = await download;
  expect(file.suggestedFilename()).toBe("Ödeme gereksinimleri.csv");

  const bytes = await readFile(await file.path());
  expect([...bytes.subarray(0, 3)]).toEqual([0xef, 0xbb, 0xbf]); // the BOM survived
  const [header, first] = bytes.toString("utf8").slice(1).split("\r\n");
  expect(header).toBe("code;title;parent;status;description");
  expect(first).toBe("AUTH;Kimlik doğrulama;;approved;");
});
