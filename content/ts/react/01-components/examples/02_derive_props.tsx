// LEARN(ts.react.components): derive prop types from one source of truth instead of copying fields.
import type { ComponentProps } from "react";
import { renderToStaticMarkup } from "react-dom/server";

// The API layer's type: the one place these fields are declared.
interface User {
  id: number;
  name: string;
  city: string;
  email: string;
}

// The badge needs two of them: pick, don't redeclare.
function UserBadge({ name, city }: Pick<User, "name" | "city">) {
  return <span>{`${name} (${city})`}</span>;
}

// Reuse another component's props by its type, then add your own.
type UserRowProps = ComponentProps<typeof UserBadge> & { online: boolean };

function UserRow({ online, ...badge }: UserRowProps) {
  return (
    <li>
      {online ? "çevrimiçi: " : "çevrimdışı: "}
      <UserBadge {...badge} />
    </li>
  );
}

// Mutually exclusive props: a link has href, a button has onClick, never both.
type ActionProps =
  | { kind: "link"; label: string; href: string }
  | { kind: "button"; label: string; onClick: () => void };

function Action(props: ActionProps) {
  if (props.kind === "link") return <a href={props.href}>{props.label}</a>;
  return (
    <button type="button" onClick={props.onClick}>
      {props.label}
    </button>
  );
}

const ayse: User = { id: 7, name: "Ayşe", city: "İzmir", email: "a@x.tr" };
const html = renderToStaticMarkup(
  <section>
    <ul>
      <UserRow name={ayse.name} city={ayse.city} online />
    </ul>
    <Action kind="link" label="Profil" href={`/users/${ayse.id}`} />
    <Action kind="button" label="Engelle" onClick={() => {}} />
  </section>,
);
console.log(html.replaceAll("><", ">\n<"));

// <Action kind="link" label="Profil" href="/users/7" onClick={() => {}} />;
// error TS2322: Type '{ kind: "link"; label: string; href: string; onClick: () => void; }' is not assignable to type 'IntrinsicAttributes & ActionProps'.
//   Property 'onClick' does not exist on type 'IntrinsicAttributes & { kind: "link"; label: string; href: string; }'.
