// LEARN(ts.architecture.patterns): an adapter is a function from their shape to yours; a factory picks one.

// The port: the only shape the notes feature depends on.
interface NotesStore {
  load(lessonId: string): Promise<string>;
  save(lessonId: string, body: string): Promise<void>;
}

// Their shape: a callback-style SDK with its own key scheme.
type Callback<T> = (err: Error | null, value?: T) => void;
const legacySdk = {
  data: new Map([["notes/ts.architecture.patterns", "strateji = fonksiyon"]]),
  get(key: string, cb: Callback<string>) {
    setTimeout(() => cb(null, this.data.get(key)), 0);
  },
  put(key: string, value: string, cb: Callback<void>) {
    this.data.set(key, value);
    setTimeout(() => cb(null), 0);
  },
};

// Adapter: a function that returns an object of the right shape. No class needed.
function fromLegacySdk(sdk: typeof legacySdk): NotesStore {
  const key = (id: string) => `notes/${id}`;
  return {
    load: (id) =>
      new Promise((resolve, reject) =>
        sdk.get(key(id), (err, value) => (err ? reject(err) : resolve(value ?? ""))),
      ),
    save: (id, body) =>
      new Promise((resolve, reject) =>
        sdk.put(key(id), body, (err) => (err ? reject(err) : resolve())),
      ),
  };
}

// A second adapter, for tests and offline mode: a Map is enough.
function inMemory(map = new Map<string, string>()) {
  return {
    load: async (id: string) => map.get(id) ?? "",
    save: async (id: string, body: string) => {
      map.set(id, body);
    },
  } satisfies NotesStore; // the shape is checked here, where it is defined
}

// Factory: callers ask for a NotesStore and never name a concrete adapter.
function createNotesStore(mode: "sdk" | "memory"): NotesStore {
  return mode === "sdk" ? fromLegacySdk(legacySdk) : inMemory();
}

for (const mode of ["sdk", "memory"] as const) {
  const store = createNotesStore(mode);
  await store.save("ts.architecture.state", "URL mi?");
  const notes = [await store.load("ts.architecture.patterns"), await store.load("ts.architecture.state")];
  console.log(mode.padEnd(7), JSON.stringify(notes));
}

// A double cast skips the check that satisfies (or a return type) performs.
const halfDone = { load: async () => "" } as unknown as NotesStore;
try {
  await halfDone.save("x", "y");
} catch (e) {
  console.log(`cast    ${e instanceof Error ? e.name : e}, typeof save = ${typeof halfDone.save}`);
}
