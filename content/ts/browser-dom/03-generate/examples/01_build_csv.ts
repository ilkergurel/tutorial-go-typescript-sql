// LEARN(ts.browser-dom.generate): CSV for Turkish Excel: semicolons, decimal commas, quoting, CRLF.

interface Order {
  name: string;
  city: string;
  total: number;
  note: string;
}

const orders: Order[] = [
  { name: "Şule Kaya", city: "İzmir", total: 1234.5, note: "" },
  { name: "Zeynep Ak", city: "Ankara", total: 99.9, note: 'kapıda "nakit"' },
  { name: "Çağla Er", city: "Iğdır", total: 15, note: "zil bozuk; arayın" },
  { name: "Ayşe Yılmaz", city: "Çorum", total: 0.75, note: "2. kat\n3 numara" },
];

// One column = a header and how to read the value from a row.
interface Column<T> {
  header: string;
  value: (row: T) => string | number;
}

const SEP = ";"; // Excel with Turkish settings splits on ";", since "," is the decimal mark
const decimal = new Intl.NumberFormat("tr-TR", { useGrouping: false });

function cell(value: string | number): string {
  const text = typeof value === "number" ? decimal.format(value) : value;
  // RFC 4180: quote a field holding the separator, a quote or a line break,
  // and double the quotes inside it.
  return /[";\r\n]/.test(text) ? `"${text.replaceAll('"', '""')}"` : text;
}

function toCsv<T>(rows: readonly T[], columns: readonly Column<T>[]): string {
  const lines = [
    columns.map((c) => cell(c.header)),
    ...rows.map((row) => columns.map((c) => cell(c.value(row)))),
  ];
  return lines.map((fields) => fields.join(SEP) + "\r\n").join("");
}

// sort() compares UTF-16 code units, which puts Ç and Ş after Z.
// Intl.Collator("tr") uses the Turkish alphabet: C Ç ... I İ ... S Ş.
const names = orders.map((o) => o.name);
console.log(`sort(): ${[...names].sort().join(", ")}`);
const collator = new Intl.Collator("tr");
const sorted = [...orders].sort((a, b) => collator.compare(a.name, b.name));

const csv = toCsv(sorted, [
  { header: "Müşteri", value: (o) => o.name },
  { header: "Şehir", value: (o) => o.city },
  { header: "Tutar (₺)", value: (o) => o.total },
  { header: "Not", value: (o) => o.note },
]);
console.log(csv.replaceAll("\r\n", "⏎\n").trimEnd()); // ⏎ marks each CRLF
