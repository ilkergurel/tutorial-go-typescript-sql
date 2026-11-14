// LEARN(ts.everyday-types.null): ?. stops at null/undefined; ?? defaults only for null/undefined, unlike ||.

interface Settings {
  pageSize?: number;
  title?: string;
  owner?: { name: string; email?: string } | null;
}

const settings: Settings = { pageSize: 0, title: "", owner: null };

// || replaces every falsy value (0, "", false, NaN); ?? only null and undefined.
const { pageSize, title } = settings;
console.log(`pageSize: ${pageSize || 50} vs ${pageSize ?? 50}`);
console.log(`title: "${title || "Untitled"}" vs "${title ?? "Untitled"}"`);

// ?. short-circuits the rest of the chain when owner is null or undefined.
// const email = settings.owner.email;
// error TS18049: 'settings.owner' is possibly 'null' or 'undefined'.
console.log(settings.owner?.email?.toLowerCase() ?? "no email");

const team: Settings = { owner: { name: "Işıl" } };
console.log(team.owner?.name.toLocaleUpperCase("tr") ?? "no owner");

// Optional call: invoke the callback only if one was passed.
function save(onDone?: (id: string) => void): void {
  onDone?.("a1");
}
save();
save((id) => console.log(`saved ${id}`));

// ??= assigns only when the current value is null or undefined.
const cache: { token?: string } = {};
cache.token ??= "t-1";
cache.token ??= "t-2";
console.log(cache.token);
