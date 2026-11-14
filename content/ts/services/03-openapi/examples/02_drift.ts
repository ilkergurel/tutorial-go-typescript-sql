// LEARN(ts.services.openapi): generated types are a compile-time promise; when the server drifts from the spec, only a run-time check notices.
import { z } from "zod";

// Generated from the spec: items is a required array, id an int64.
interface components {
  schemas: {
    Order: {
      /** Format: int64 */
      id: number;
      customer: string;
      items: string[];
    };
  };
}
type Order = components["schemas"]["Order"];

// What the Go server actually sends. A nil []string marshals as null, and an
// int64 above 2^53 is still a plain JSON number.
const wire = '{"id":9007199254740993,"customer":"Ayşe","items":null}';

const order = JSON.parse(wire) as Order; // what a typed client does
console.log(`id: ${order.id}`);
try {
  console.log(`items: ${order.items.length}`);
} catch (e) {
  console.log(`items: ${e instanceof Error ? e.name : e}`);
}

// A schema annotated with the generated type: tsc rejects it if it stops
// matching the contract, and at run time it catches what types can't.
const OrderSchema: z.ZodType<Order> = z.object({
  id: z.number().int(), // int() accepts only safe integers, up to 2^53 - 1
  customer: z.string(),
  items: z.array(z.string()),
});
const checked = OrderSchema.safeParse(JSON.parse(wire));
if (!checked.success) console.log(z.prettifyError(checked.error));

// The fix belongs in the contract, on both sides: an empty slice instead of
// nil in Go, and the id as a string (`json:"id,string"`, type: string).
const fixed = '{"id":"9007199254740993","customer":"Ayşe","items":[]}';
const Fixed = z.object({
  id: z.string().regex(/^\d+$/),
  customer: z.string(),
  items: z.array(z.string()),
});
const ok = Fixed.parse(JSON.parse(fixed));
console.log(
  `fixed: id ${ok.id}, next ${BigInt(ok.id) + 1n}, items ${ok.items.length}`,
);
