// LEARN(ts.functions.signatures): ? lets callers omit a parameter; = also fills it in.

// title?: string means callers may leave it out, so inside it is string | undefined.
function greet(name: string, title?: string): string {
  // return `${title.trim()} ${name}`;
  //   error TS18048: 'title' is possibly 'undefined'.
  return title === undefined ? name : `${title} ${name}`;
}

// A default makes the parameter optional for callers and a plain string
// inside. Its type is inferred from the default value.
function lower(text: string, locale = "tr-TR"): string {
  return text.toLocaleLowerCase(locale);
}

// The default is evaluated on every call, so each call gets a fresh array,
// unlike Python's shared mutable default.
function tag(label: string, into: string[] = []): string[] {
  into.push(label);
  return into;
}

console.log(greet("Ayşe"), "|", greet("Ayşe", "Dr."));

// Passing undefined explicitly selects the default as well.
console.log(
  lower("ISPARTA"),
  lower("ISPARTA", undefined),
  lower("ISPARTA", "en-US"),
);
// null is a value, not a missing argument: no default, and tsc rejects it.
// lower("ISPARTA", null);
//   error TS2345: Argument of type 'null' is not assignable to parameter of type 'string | undefined'.

// en-US keeps the dot of İ as a separate combining character (U+0307).
const city = "İZMİR";
console.log(lower(city), lower(city).length, lower(city, "en-US").length);

console.log(tag("a"), tag("b"));
