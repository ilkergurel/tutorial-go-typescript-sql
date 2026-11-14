// LEARN(ts.mental-model.tsconfig): noUncheckedIndexedAccess adds undefined to every index read.

const roleByLogin: Record<string, string> = {
  ışık: "admin",
  ayşe: "editor",
};

// Without the flag, roleByLogin[key] is typed string. The `!` below spells out
// the assumption that, without the flag, every index read makes silently.
function roleUnchecked(login: string): string {
  return roleByLogin[login.toLowerCase()]!.toUpperCase();
}

// With the flag the read is string | undefined, so dropping the check fails:
// return roleByLogin[key].toUpperCase();
// error TS2532: Object is possibly 'undefined'.
function roleOf(login: string): string {
  const key = login.toLocaleLowerCase("tr");
  const role = roleByLogin[key];
  return role === undefined ? "GUEST" : role.toUpperCase();
}

// Turkish dotless I: the default lowercasing gives "işik", not "ışık".
console.log("IŞIK".toLowerCase(), "IŞIK".toLocaleLowerCase("tr"));
try {
  console.log(roleUnchecked("IŞIK"));
} catch (e) {
  console.log(e instanceof TypeError ? "TypeError: no role for işik" : e);
}
console.log(roleOf("IŞIK"), roleOf("AYŞE"), roleOf("mehmet"));

// Arrays too: split() can return fewer parts than you index.
const [user, domain] = "ayse".split("@");
console.log(user, domain === undefined ? "(no domain)" : domain);
