// LEARN(ts.type-level.utility): a mapped type loops over keys; the built-in utilities are a few lines each.

interface RequirementForm {
  title: string;
  priority: number;
  owner: string;
}

// Partial<T> is { [K in keyof T]?: T[K] }. The same loop builds any per-field type.
type FieldErrors<T> = { [K in keyof T]?: string };
type Validators<T> = { [K in keyof T]: (value: T[K]) => string | undefined };

const rules: Validators<RequirementForm> = {
  title: (v) => (v.trim() === "" ? "Başlık boş olamaz" : undefined),
  priority: (v) => ([1, 2, 3].includes(v) ? undefined : "Öncelik 1, 2 veya 3 olmalı"),
  owner: (v) => (v.length > 20 ? "Sahip en fazla 20 karakter" : undefined),
};
// const partial: Validators<RequirementForm> = { title: () => undefined };
// error TS2739: Type '{ title: () => undefined; }' is missing the following properties from type 'Validators<RequirementForm>': priority, owner

function validate<T>(value: T, validators: Validators<T>): FieldErrors<T> {
  const errors: FieldErrors<T> = {};
  for (const key in validators) {
    const message = validators[key](value[key]);
    if (message !== undefined) errors[key] = message;
  }
  return errors;
}

console.log(validate({ title: "Rapor dışa aktarma", priority: 2, owner: "Şule" }, rules));
console.log(validate({ title: "  ", priority: 7, owner: "Ayşe" }, rules));

// Modifiers: -readonly and -? remove them; Required<T> is the -? version of Partial.
type Mutable<T> = { -readonly [K in keyof T]: T[K] };
// Key remapping with as (TS 4.1): a key mapped to never is dropped.
type Public<T> = { [K in keyof T as K extends `_${string}` ? never : K]: T[K] };

interface Stored {
  readonly id: string;
  title: string;
  _etag: string;
}
const row: Stored = { id: "REQ-LOGIN", title: "Giriş", _etag: 'W/"7"' };
const copy: Mutable<Public<Stored>> = { id: row.id, title: row.title };
copy.id = "REQ-LOGIN-2"; // allowed: readonly was removed
console.log(copy);
