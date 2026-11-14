// LEARN(ts.react.components): props are one typed object; children is a prop; extend native props.
import type { ComponentPropsWithoutRef, ReactNode } from "react";
import { renderToStaticMarkup } from "react-dom/server";

type Tone = "neutral" | "danger";

// Every native <button> prop (onClick, disabled, aria-*, form...) plus ours.
interface ButtonProps extends ComponentPropsWithoutRef<"button"> {
  tone?: Tone;
}

function Button({ tone = "neutral", type = "button", ...rest }: ButtonProps) {
  return <button type={type} data-tone={tone} {...rest} />;
}

interface CardProps {
  title: string;
  warnings: number;
  footer?: ReactNode;
  children: ReactNode; // text, elements, arrays, null: anything renderable
}

function Card({ title, warnings, footer, children }: CardProps) {
  return (
    <section aria-label={title}>
      <h2>{title.toLocaleUpperCase("tr")}</h2>
      {warnings > 0 && <strong>{warnings} uyarı</strong>}
      {children}
      {footer}
    </section>
  );
}

const card = (
  <Card
    title="sipariş iptali"
    warnings={0}
    footer={<Button tone="danger">İptal et</Button>}
  >
    <p>Bu işlem geri alınamaz.</p>
  </Card>
);
console.log(renderToStaticMarkup(card).replaceAll("><", ">\n<"));

// The && trap: 0 is renderable, false is not.
const warnings = 0;
console.log(renderToStaticMarkup(<p>{warnings && <strong>uyarı</strong>}</p>));

// <Button tone="red">Sil</Button>;
// error TS2322: Type '"red"' is not assignable to type 'Tone | undefined'.
// <Card title="Sipariş" warnings={0}>{order}</Card>; // order = { id: 42 }
// error TS2322: Type '{ id: number; }' is not assignable to type 'ReactNode'.
