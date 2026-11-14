// LEARN(ts.services.zod): a schema maps an input type to an output type: coerce, default, transform, refine.
import { z } from "zod";

const Booking = z
  .object({
    city: z
      .string()
      .trim()
      .overwrite((s) => s.toLocaleUpperCase("tr")),
    guests: z.coerce.number().int().min(1), // query strings carry text
    from: z.iso.date(),
    to: z.iso.date(),
    createdAt: z.iso.datetime({ offset: true }).transform((s) => new Date(s)),
    note: z.string().default(""),
  })
  .refine((b) => b.from < b.to, {
    message: "must be after from",
    path: ["to"],
  });

type BookingInput = z.input<typeof Booking>; // what arrives
type Booking = z.output<typeof Booking>; // what the app works with

const wire: BookingInput = {
  city: " izmir ",
  guests: "2",
  from: "2026-11-20",
  to: "2026-11-22",
  createdAt: "2026-11-13T10:00:00.123456+03:00", // Go's time.Time, in Istanbul
};
const booking: Booking = Booking.parse(wire);
const upper = z.string().toUpperCase().parse("izmir");
console.log(`${booking.city}, but zod's toUpperCase() gives ${upper}`);
console.log(`guests: ${typeof booking.guests}, note: "${booking.note}"`);
console.log(`createdAt: ${booking.createdAt.toISOString()}`);

// Every problem is reported, each with its path.
const bad = Booking.safeParse({ ...wire, guests: "0", to: "2026-11-19" });
if (!bad.success) console.log(z.prettifyError(bad.error));

// Without { offset: true }, only a Z suffix passes.
const strict = z.iso.datetime().safeParse(wire.createdAt);
console.log(`z.iso.datetime() without offset: ${strict.success}`);
