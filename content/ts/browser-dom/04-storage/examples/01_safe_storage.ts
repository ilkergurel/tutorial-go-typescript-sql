// LEARN(ts.browser-dom.storage): Web Storage can throw; wrap it, fall back to memory, store versioned JSON.

// The consumer declares what it needs: three methods, not all of Storage.
type KeyValue = Pick<Storage, "getItem" | "setItem" | "removeItem">;

class MemoryStore implements KeyValue {
  #data = new Map<string, string>();
  getItem(key: string): string | null {
    return this.#data.get(key) ?? null;
  }
  setItem(key: string, value: string): void {
    this.#data.set(key, String(value)); // strings only, like the real thing
  }
  removeItem(key: string): void {
    this.#data.delete(key);
  }
}

function openStore(kind: "localStorage" | "sessionStorage"): KeyValue {
  try {
    const store = window[kind]; // reading the property can already throw
    store.setItem("__probe", "1"); // and writing can fail on quota
    store.removeItem("__probe");
    return store;
  } catch (e) {
    const name = e instanceof DOMException ? e.name : "error";
    console.log(`${kind} unavailable (${name}): keeping data in memory`);
    return new MemoryStore();
  }
}

interface Prefs {
  theme: "light" | "dark";
  pageSize: number;
}
const DEFAULTS: Prefs = { theme: "light", pageSize: 20 };
const KEY = "prefs:v1"; // a new shape gets a new key, not a parse error

function isPrefs(v: unknown): v is Prefs {
  if (typeof v !== "object" || v === null) return false;
  const p = v as Record<string, unknown>;
  return (
    (p.theme === "light" || p.theme === "dark") &&
    typeof p.pageSize === "number"
  );
}

// Whatever is stored was written by an older version, another tab or the user.
function loadPrefs(store: KeyValue): Prefs {
  const raw = store.getItem(KEY);
  if (raw === null) return DEFAULTS;
  try {
    const parsed: unknown = JSON.parse(raw);
    return isPrefs(parsed) ? parsed : DEFAULTS;
  } catch {
    return DEFAULTS; // corrupted JSON is not worth crashing the page
  }
}

const store = openStore("localStorage");
console.log(`first visit: ${JSON.stringify(loadPrefs(store))}`);
store.setItem(KEY, JSON.stringify({ theme: "dark", pageSize: 50 }));
console.log(`saved:       ${JSON.stringify(loadPrefs(store))}`);
store.setItem(KEY, '{"theme":"blue"}');
console.log(`bad shape:   ${JSON.stringify(loadPrefs(store))}`);
