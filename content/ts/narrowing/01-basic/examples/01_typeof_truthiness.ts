// LEARN(ts.narrowing.basic): typeof, truthiness and equality checks narrow a union.

// Spreadsheet imports turn the Ankara postcode "06100" into the number 6100.
function postcode(code: string | number): string {
  if (typeof code === "number") {
    return String(code).padStart(5, "0"); // code: number
  }
  return code.trim(); // code: string; the return above removed number
}

// typeof null is "object", and the compiler knows it.
function tagCount(tags: string[] | string | null): number {
  if (typeof tags === "object") {
    // return tags.length;
    //   error TS18047: 'tags' is possibly 'null'.
    return tags === null ? 0 : tags.length; // tags: string[] | null
  }
  return tags.split(",").length; // tags: string
}

// Truthiness narrowing drops 0 and "" together with null and undefined.
function stockNaive(qty: number | undefined): string {
  return qty ? `${qty} left` : "unknown";
}
function stock(qty: number | undefined): string {
  return qty !== undefined ? `${qty} left` : "unknown";
}

// == null is the one idiomatic loose comparison: it matches null and undefined,
// and nothing else. "" is still a string here.
function displayName(name: string | null | undefined): string {
  return name == null ? "(anonymous)" : name.toLocaleUpperCase("tr-TR");
}

console.log(postcode(6100), postcode(" 34000 "));
console.log(tagCount(null), tagCount(["a", "b"]), tagCount("x,y,z"));
console.log(stockNaive(0), "|", stock(0));
console.log([displayName(undefined), displayName("ilgaz"), displayName("")]);
