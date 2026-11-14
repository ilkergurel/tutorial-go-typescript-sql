// LEARN(ts.objects-classes.classes): abstract classes share an algorithm and force the varying steps; interfaces only describe shapes.

abstract class Exporter {
  // The template: the fixed steps live here, the varying ones are abstract.
  export(rows: readonly (readonly string[])[]): string {
    return [this.header(), ...rows.map((r) => this.row(r))].join("\n");
  }
  protected abstract header(): string;
  protected abstract row(cells: readonly string[]): string;
}

class CsvExporter extends Exporter {
  protected override header(): string {
    return "ad;şehir";
  }
  protected override row(cells: readonly string[]): string {
    return cells.join(";");
  }
}
// new Exporter();
// error TS2511: Cannot create an instance of an abstract class.
// class Broken extends Exporter {}
// error TS2654: Non-abstract class 'Broken' is missing implementations for the following members of 'Exporter': 'header', 'row'.

const rows = [
  ["Ayşe", "İzmir"],
  ["Işık", "Ankara"],
];
console.log(new CsvExporter().export(rows));

// The same variation point as data: an interface and a function.
interface Format {
  header: string;
  row(cells: readonly string[]): string;
}
function exportWith(format: Format, data: readonly (readonly string[])[]): string {
  return [format.header, ...data.map((r) => format.row(r))].join("\n");
}
const markdown: Format = {
  header: "| ad | şehir |",
  row: (cells) => `| ${cells.join(" | ")} |`,
};
console.log(exportWith(markdown, rows));

// override catches a base method renamed under your feet:
// class Tsv extends CsvExporter { protected override heading() { return ""; } }
// error TS4113: This member cannot have an 'override' modifier because it is not declared in the base class 'CsvExporter'.

// A class exists at run time, so instanceof can test for it; an interface can't.
// console.log(markdown instanceof Format);
// error TS2693: 'Format' only refers to a type, but is being used as a value here.
console.log(new CsvExporter() instanceof Exporter);
