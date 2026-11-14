// LEARN(ts.objects-classes.modules): ES modules export live bindings, run once per specifier, and expose read-only namespaces.

// A second module, inlined as a data: URL because this sandbox runs one file.
const source = `
  console.log("counter module evaluated");
  export let count = 0;
  export function increment() { count += 1; }
`;
const url = `data:text/javascript,${encodeURIComponent(source)}`;

// What a .d.ts file does: describe JavaScript the compiler can't see.
// import() of a computed string is typed Promise<any>, so this annotation
// is a promise you make, exactly like a hand-written declaration file.
interface CounterModule {
  readonly count: number;
  increment(): void;
}

const counter: CounterModule = await import(url);
const { count: snapshot } = counter; // destructuring copies today's value
counter.increment();
counter.increment();
console.log(`live: ${counter.count}, snapshot: ${snapshot}`);

// Same specifier, same instance: the module body does not run again.
const again: CounterModule = await import(url);
console.log(`same instance: ${again === counter}, count: ${again.count}`);

// Importers read bindings; only the exporting module can assign them.
try {
  (counter as { count: number }).count = 100;
} catch (e) {
  console.log(e instanceof TypeError ? "TypeError: namespace is read-only" : e);
}
