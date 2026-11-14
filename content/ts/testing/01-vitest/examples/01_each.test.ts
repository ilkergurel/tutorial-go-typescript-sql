import { describe, expect, it } from "vitest";

// A URL slug from a Turkish title: lower-case the Turkish way, map ı to i,
// strip accents (ç -> c, ğ -> g, ş -> s), and join words with "-".
function slugify(title: string): string {
  return title
    .toLocaleLowerCase("tr")
    .replaceAll("ı", "i")
    .normalize("NFD")
    .replace(/\p{M}/gu, "") // combining marks left over by NFD
    .replaceAll("'", "")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "");
}

function capitalize(word: string): string {
  return word.charAt(0).toLocaleUpperCase("tr") + word.slice(1);
}

describe("slugify", () => {
  // One row per case; $title and $want put each row into the test name.
  it.each([
    { title: "İstanbul'da Hava", want: "istanbulda-hava" },
    { title: "Iğdır Kalesi", want: "igdir-kalesi" },
    { title: "  Çay & Simit  ", want: "cay-simit" },
    { title: "Öğrenci Ücretleri 2026", want: "ogrenci-ucretleri-2026" },
  ])("$title -> $want", ({ title, want }) => {
    expect(slugify(title)).toBe(want);
  });
});

describe("capitalize", () => {
  // Array rows are spread into the parameters; %s formats them in order.
  it.each([
    ["istanbul", "İstanbul"],
    ["ırmak", "Irmak"],
    ["çorum", "Çorum"],
  ])("capitalize(%s) is %s", (word, want) => {
    expect(capitalize(word)).toBe(want);
  });
});
