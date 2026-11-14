// LEARN(ts.everyday-types.null): null and undefined are two values with different behaviour at the edges.

// == null matches both; === tells them apart. typeof null is "object" (an old bug).
console.log(`${null == undefined} ${null === undefined} ${typeof null}`);

// JSON has null but no undefined: such properties vanish, array slots become null.
const row = { nickname: null, phone: undefined };
console.log(JSON.stringify(row), JSON.stringify([undefined]));

// A Go API: a nil *string marshals as null, an omitempty field is left out.
interface ApiUser {
  login: string;
  nickname: string | null;
  phone?: string;
}
const fromGo: ApiUser = JSON.parse('{"login":"ayse","nickname":null}');
console.log(`${fromGo.nickname === null} ${"phone" in fromGo}`);

// Default parameters and destructuring defaults apply to undefined only.
function greet(name: string | null = "misafir"): string {
  return `Merhaba, ${name}`;
}
console.log(greet(), "|", greet(undefined), "|", greet(null));
const { phone = "no phone", nickname = "no nickname" } = fromGo;
console.log(phone, "|", nickname);
