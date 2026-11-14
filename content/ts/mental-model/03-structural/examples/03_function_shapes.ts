// LEARN(ts.mental-model.structural): a function taking fewer parameters fits a slot that passes more.

// map calls its callback as (value, index, array). parseInt(string, radix?)
// accepts that shape, so this compiles, and index is used as the radix.
const ids = ["10", "10", "10"];
console.log(ids.map(parseInt).join(" "));

// Say which arguments you mean, or pass a function that reads only the first.
console.log(ids.map((s) => parseInt(s, 10)).join(" "));
console.log(ids.map(Number).join(" "));

// The rule exists so callbacks can ignore what they don't need:
type OnSave = (id: string, attempt: number) => void;
const log: string[] = [];
const onSave: OnSave = (id) => {
  log.push(`saved ${id}`);
};
onSave("a1", 1);
console.log(log.join(", "));

// Requiring MORE than the caller passes is still an error:
// const strict: OnSave = (id: string, attempt: number, force: boolean) => {};
// error TS2322: Type '(id: string, attempt: number, force: boolean) => void' is not assignable to type 'OnSave'.
//   Target signature provides too few arguments. Expected 3 or more, but got 2.
