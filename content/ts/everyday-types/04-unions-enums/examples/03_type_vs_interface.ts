// LEARN(ts.everyday-types.unions-enums): interfaces for object shapes, type aliases for unions and the rest.

interface BaseEvent {
  at: string;
}
interface Login extends BaseEvent {
  kind: "login";
  user: string;
}
interface Logout extends BaseEvent {
  kind: "logout";
  user: string;
  reason: "timeout" | "manual";
}
// A union (like a tuple or a primitive alias) can only be named with `type`.
type AuditEvent = Login | Logout;

// Declaration merging: a second `interface Login` adds a member to the first.
interface Login {
  ip?: string;
}

function describe(e: AuditEvent): string {
  if (e.kind === "login") {
    return `${e.at} ${e.user} giriş yaptı (${e.ip ?? "?"})`;
  }
  return `${e.at} ${e.user} çıktı: ${e.reason}`;
}

// extends reports a conflicting member at the declaration...
// interface Broken extends BaseEvent { at: number }
// error TS2430: Interface 'Broken' incorrectly extends interface 'BaseEvent'.
// ...while & silently makes it string & number, which is never:
type Odd = BaseEvent & { at: number };
// const odd: Odd = { at: "09:00" };
// error TS2322: Type 'string' is not assignable to type 'never'.

const events: AuditEvent[] = [
  { kind: "login", user: "ayşe", at: "09:00", ip: "10.0.0.7" },
  { kind: "logout", user: "ayşe", at: "17:30", reason: "manual" },
];
for (const e of events) console.log(describe(e));
