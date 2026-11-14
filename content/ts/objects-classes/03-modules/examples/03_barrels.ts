// LEARN(ts.objects-classes.modules): a barrel evaluates every module it re-exports, and export * hides name clashes.

const mod = (source: string) => `data:text/javascript,${encodeURIComponent(source)}`;

// Three feature modules, each announcing when its body runs.
const dates = mod(`
  console.log("evaluating dates");
  export const formatDate = (d) => d.toISOString().slice(0, 10);`);
const money = mod(`
  console.log("evaluating money");
  export const formatMoney = (n) => n.toLocaleString("tr-TR", { minimumFractionDigits: 2 });
  export const VERSION = "money 1.4";`);
const charts = mod(`
  console.log("evaluating charts (imagine 300 kB)");
  export const drawChart = () => "chart";
  export const VERSION = "charts 2.0";`);

// The barrel: an index.ts made of `export *` lines.
const barrel = mod(`
  export * from "${dates}";
  export * from "${money}";
  export * from "${charts}";`);

interface Shared {
  formatMoney(n: number): string;
  VERSION?: string;
}

// A direct import evaluates just that module.
const direct: Shared = await import(money);
console.log(direct.formatMoney(1250.5), direct.VERSION);

// Through the barrel, the same function costs every sibling module as well.
// money is cached and does not run again: one instance per specifier.
const shared: Shared = await import(barrel);
console.log(`same function: ${shared.formatMoney === direct.formatMoney}`);

// Two modules export VERSION, so export * exports neither: no error, no value.
console.log(`VERSION: ${shared.VERSION}; exports: ${Object.keys(shared).join(", ")}`);
