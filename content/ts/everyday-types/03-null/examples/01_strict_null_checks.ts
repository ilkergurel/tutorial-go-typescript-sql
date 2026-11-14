// LEARN(ts.everyday-types.null): with strictNullChecks, absence is part of the type and must be handled.

interface User {
  login: string;
  nickname: string | null; // always present, may be null (a nullable column)
  phone?: string; // may be missing altogether
}

const users: User[] = [
  { login: "ayse", nickname: "Ayşe" },
  { login: "mehmet", nickname: null, phone: "+90 555 000 0000" },
];

// find() can miss, so its type is User | undefined.
const ali = users.find((u) => u.login === "ali");
// console.log(ali.login);
// error TS18048: 'ali' is possibly 'undefined'.
if (ali === undefined) {
  console.log("no user ali");
} else {
  console.log(ali.login); // narrowed to User in this branch
}

// Map.get is the same: V | undefined.
const byLogin = new Map(users.map((u) => [u.login, u]));
const mehmet = byLogin.get("mehmet");
if (mehmet !== undefined) {
  // console.log(mehmet.nickname.length);
  // error TS18047: 'mehmet.nickname' is possibly 'null'.
  console.log(`${mehmet.login}: ${mehmet.nickname ?? "(no nickname)"}`);
}

// The non-null assertion ! silences the check and moves the crash to run time.
try {
  console.log(byLogin.get("ali")!.login);
} catch (e) {
  console.log(e instanceof TypeError ? "TypeError: the ! was wrong" : e);
}
