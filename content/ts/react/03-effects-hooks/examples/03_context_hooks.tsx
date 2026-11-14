// LEARN(ts.react.effects-hooks): context passes a value down the tree; custom hooks compose hooks.
import { createContext, useContext } from "react";
import { renderToStaticMarkup } from "react-dom/server";

type Locale = "tr-TR" | "en-US";

// No sensible default, so null, and a hook that refuses to run without a provider.
const LocaleContext = createContext<Locale | null>(null);

function useLocale(): Locale {
  const locale = useContext(LocaleContext);
  if (locale === null) {
    throw new Error("useLocale needs a LocaleContext provider");
  }
  return locale; // narrowed: callers never see null
}

// A custom hook is a function that calls hooks. It shares logic, not state.
function useSorted(items: readonly string[]): string[] {
  const collator = new Intl.Collator(useLocale());
  return [...items].sort(collator.compare);
}

function useAmount(amount: number): string {
  const format = { minimumFractionDigits: 2 };
  return new Intl.NumberFormat(useLocale(), format).format(amount);
}

const cities = ["Şanlıurfa", "Sivas", "Çanakkale", "Ceyhan", "Ödemiş", "Ordu"];

function Shipping() {
  return <p>{`${useSorted(cities).join(", ")} | ${useAmount(1249.9)} TL`}</p>;
}

for (const locale of ["tr-TR", "en-US"] as const) {
  const tree = (
    <LocaleContext.Provider value={locale}>
      <Shipping />
    </LocaleContext.Provider>
  );
  console.log(`${locale}: ${renderToStaticMarkup(tree)}`);
}

try {
  renderToStaticMarkup(<Shipping />);
} catch (e) {
  console.log(`no provider: ${e instanceof Error ? e.message : e}`);
}
