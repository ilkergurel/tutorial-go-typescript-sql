Refactor example 2 so the counting logic lives in:

    func countLevels(r io.Reader) (map[string]int, error)

and `main` only opens the file and wires it up. Hidden tests call `countLevels`
with `strings.NewReader` — that's the point: the logic no longer knows or cares
where the bytes come from.

Cases the tests cover: empty input, mixed levels, malformed lines (skipped).
Compare maps with `maps.Equal`.
