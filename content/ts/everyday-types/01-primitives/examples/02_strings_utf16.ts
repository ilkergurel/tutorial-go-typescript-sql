// LEARN(ts.everyday-types.primitives): string is UTF-16 code units; case mapping depends on locale.

const city = "İstanbul";

// length counts UTF-16 code units, not characters and not UTF-8 bytes.
const codePoints = [...city].length;
const utf8Bytes = new TextEncoder().encode(city).length;
console.log(`code units ${city.length}, code points ${codePoints}`);
console.log(`UTF-8 bytes ${utf8Bytes}`);

// Default lowercasing maps İ to "i" + U+0307 (combining dot): one char more.
const lower = city.toLowerCase();
console.log(`${lower} length ${lower.length}`);

// The Turkish locale maps İ to i and I to ı, as a Turkish reader expects.
const trLower = city.toLocaleLowerCase("tr");
console.log(`${trLower} length ${trLower.length}`);
console.log(`${"iyi".toUpperCase()} ${"iyi".toLocaleUpperCase("tr")}`);

// So a case-insensitive comparison needs the locale too.
const typed = "ISPARTA";
const stored = "ısparta";
console.log(typed.toLowerCase() === stored); // "isparta"
console.log(typed.toLocaleLowerCase("tr") === stored);

// Outside the BMP, one character is two code units.
const thumbs = "👍";
console.log(`${thumbs.length} ${[...thumbs].length}`);
