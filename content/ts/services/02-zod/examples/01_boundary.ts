// LEARN(ts.services.zod): types vanish at run time; a zod schema checks the data and produces the type.
import { z } from "zod";

// What the server sent after a deploy: id became a string.
const body = '{"id":"7","name":"Şule Yılmaz","email":"sule(at)example.com"}';

// 1. A cast compiles and checks nothing.
interface UserCast {
  id: number;
  name: string;
  email: string;
}
const cast = JSON.parse(body) as UserCast;
console.log(`cast: next id = ${cast.id + 1}`);

// 2. One schema; the static type is derived from it, never written twice.
const User = z.object({
  id: z.number().int().positive(),
  name: z.string().trim().min(1),
  email: z.email(),
});
type User = z.infer<typeof User>;

const result = User.safeParse(JSON.parse(body));
if (result.success) {
  console.log(`next id = ${result.data.id + 1}`); // data is User here
} else {
  console.log(z.prettifyError(result.error));
  for (const issue of result.error.issues) {
    console.log(`${issue.path.join(".")}: ${issue.code}`);
  }
}

// 3. Valid input: unknown keys are dropped and trim() is applied.
const input = { id: 7, name: " Şule ", email: "sule@example.com", admin: true };
const user: User = User.parse(input);
console.log(JSON.stringify(user));

// parse() throws a ZodError; a strict object rejects unknown keys instead.
try {
  z.strictObject(User.shape).parse(input);
} catch (e) {
  if (e instanceof z.ZodError) console.log(e.issues[0].message);
}
