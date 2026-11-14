// LEARN(inside.codebase.ts-runner): a browser has no process exit, so the runner must decide when a run is over.
// A virtual event loop, so the timings below are exact: timers run in due order.
class Loop {
  now = 0;
  private seq = 0;
  private timers = new Map<number, { at: number; fn: () => void }>();
  setTimeout(fn: () => void, ms: number): number {
    this.timers.set(++this.seq, { at: this.now + ms, fn });
    return this.seq;
  }
  get pending(): number {
    return this.timers.size;
  }
  /** Runs the earliest timer if it is due by `limit`; false when none is. */
  step(limit: number): boolean {
    const next = [...this.timers].sort((a, b) => a[1].at - b[1].at)[0];
    if (!next || next[1].at > limit) return false;
    const [id, t] = next;
    this.timers.delete(id);
    this.now = t.at;
    t.fn();
    return true;
  }
}

type Program = (loop: Loop, log: (s: string) => void) => void;

// A typical async lesson: data after 30 ms, rendered 90 ms later.
const fetchThenRender: Program = (loop, log) => {
  log("start");
  loop.setTimeout(() => {
    log("data loaded");
    loop.setTimeout(() => log("rendered"), 90);
  }, 30);
};
// A heartbeat nobody stops: what an uncleared setInterval looks like.
const heartbeat: Program = (loop, log) => {
  const beat = () => loop.setTimeout(beat, 1000);
  log("start");
  beat();
};

function run(program: Program, rule: "fixed 50 ms" | "no timer pending"): string {
  const loop = new Loop();
  const out: string[] = [];
  program(loop, (s) => out.push(`${s} @${loop.now}`));
  // The module's last statement has run: the shim's mainDone.
  let end: string;
  if (rule === "fixed 50 ms") {
    while (loop.step(50));
    end = "done at 50 ms";
  } else {
    while (loop.pending > 0 && loop.step(10_000));
    end = loop.pending === 0 ? `done at ${loop.now} ms` : "timed out after 10000 ms, exit 124";
  }
  return `${program.name}, ${rule}: ${out.join(", ")} -> ${end}`;
}

console.log(run(fetchThenRender, "fixed 50 ms"));
console.log(run(fetchThenRender, "no timer pending"));
console.log(run(heartbeat, "no timer pending"));
