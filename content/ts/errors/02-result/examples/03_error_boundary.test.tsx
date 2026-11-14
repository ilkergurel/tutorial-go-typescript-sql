// LEARN(ts.errors.result): an error boundary turns a render-time throw into fallback UI for one subtree.

import { Component, useState, type ErrorInfo, type ReactNode } from "react";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, test } from "vitest";

afterEach(cleanup); // no Vitest globals here, so unmount between tests by hand

type BoundaryState = { failed: boolean }; // a flag, not the error: null can be thrown too
class Boundary extends Component<{ name: string; children: ReactNode }, BoundaryState> {
  state: BoundaryState = { failed: false };
  static getDerivedStateFromError(): BoundaryState {
    return { failed: true }; // render the fallback; side effects go below
  }
  componentDidCatch(error: unknown, info: ErrorInfo) {
    const where = /at (\w+)/.exec(info.componentStack ?? "")?.[1];
    const what = error instanceof Error ? error.name : typeof error;
    console.log(`[${this.props.name}] report: ${what} in ${where}`); // send to monitoring here
  }
  render() {
    if (this.state.failed) return <p role="alert">{this.props.name}: gösterilemiyor</p>;
    return this.props.children;
  }
}

type Row = { id: string; tests: { passed: boolean }[] };
function Coverage({ row }: { row: Row }) {
  return <p>{row.id}: {row.tests.filter((t) => t.passed).length}/{row.tests.length}</p>;
}
const good: Row = { id: "REQ-LOGIN", tests: [{ passed: true }, { passed: false }] };
const broken = JSON.parse('{"id":"REQ-EXPORT","tests":null}') as Row; // the type lies

test("a render error replaces only its own subtree", () => {
  render(
    <>
      <Boundary name="REQ-LOGIN"><Coverage row={good} /></Boundary>
      <Boundary name="REQ-EXPORT"><Coverage row={broken} /></Boundary>
    </>,
  );
  expect(screen.getByText("REQ-LOGIN: 1/2")).toBeTruthy();
  expect(screen.getByRole("alert").textContent).toBe("REQ-EXPORT: gösterilemiyor");
});

function SaveButton({ save }: { save: () => Promise<void> }) {
  const [failure, setFailure] = useState<{ error: unknown } | null>(null);
  if (failure) throw failure.error; // rethrown during render, where a boundary catches it
  const onClick = () => save().catch((error: unknown) => setFailure({ error }));
  return <button onClick={onClick}>Kaydet</button>;
}

test("an async failure reaches the boundary only when rethrown in render", async () => {
  const save = () => Promise.reject(new Error("HTTP 503"));
  render(<Boundary name="Kaydet"><SaveButton save={save} /></Boundary>);
  fireEvent.click(screen.getByRole("button"));
  expect((await screen.findByRole("alert")).textContent).toBe("Kaydet: gösterilemiyor");
});
