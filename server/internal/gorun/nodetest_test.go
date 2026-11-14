// nodetest_test.go: ts-test lessons with the real tools, from the app's own
// node_modules (npm install first). Integration: each test starts Node, and
// the Playwright one starts Microsoft Edge.
//
//go:build integration

package gorun

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func projectDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestRunNodeTestsVitestWithTestingLibraryAndMSW(t *testing.T) {
	src := `import { describe, it, expect, beforeAll, afterAll } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";

const server = setupServer(http.get("https://api.example.test/city", () => HttpResponse.json({ name: "İzmir" })));
beforeAll(() => server.listen());
afterAll(() => server.close());

function Counter() {
  const [n, setN] = useState(0);
  return <button onClick={() => setN(n + 1)}>clicked {n}</button>;
}

describe("counter", () => {
  it.each([[1], [3]])("counts %i clicks", async (times) => {
    render(<Counter />);
    for (let i = 0; i < times; i++) await userEvent.click(screen.getByRole("button"));
    expect(screen.getByRole("button").textContent).toBe("clicked " + times);
    screen.getByRole("button").remove();
  });
  it("fetch is mocked by msw", async () => {
    const res = await fetch("https://api.example.test/city");
    expect(await res.json()).toEqual({ name: "İzmir" });
  });
});
`
	res, err := RunNodeTests(context.Background(), Request{Files: map[string]string{"counter.test.tsx": src}}, projectDir(t))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, want := range []string{"counts 1 clicks", "counts 3 clicks", "fetch is mocked by msw", "3 passed"} {
		if !strings.Contains(res.Stdout, want) {
			t.Fatalf("missing %q (exit %d)\nstdout:\n%s\nstderr:\n%s", want, res.ExitCode, res.Stdout, res.Stderr)
		}
	}
	if res.ExitCode != 0 {
		t.Fatalf("exit %d\nstdout:\n%s\nstderr:\n%s", res.ExitCode, res.Stdout, res.Stderr)
	}
	assertPlainOutput(t, res)
}

// Output is compared as text: no color codes, and no Node warning that
// NO_COLOR is ignored (Playwright's workers printed it while NO_COLOR was set).
func assertPlainOutput(t *testing.T, res Result) {
	t.Helper()
	if strings.Contains(res.Stdout+res.Stderr, "\x1b[") || strings.Contains(res.Stdout+res.Stderr, "NO_COLOR") {
		t.Fatalf("colored output or a color warning\nstdout:\n%s\nstderr:\n%s", res.Stdout, res.Stderr)
	}
}

// timeout_s bounds a ts-test run as it bounds a Go program: a hanging test
// ends at the lesson's limit, not at Vitest's own 60 s test timeout.
func TestRunNodeTestsHonorsTimeout(t *testing.T) {
	src := `import { it } from "vitest";
it("never settles", () => new Promise(() => {}), 60_000);
`
	res, err := RunNodeTests(context.Background(), Request{Files: map[string]string{"hang.test.ts": src}, TimeoutS: 4}, projectDir(t))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !res.TimedOut || res.DurationMs >= (4*time.Second+pipeGrace).Milliseconds() {
		t.Fatalf("want a timeout at 4s, got timedOut=%v after %dms\nstdout:\n%s\nstderr:\n%s", res.TimedOut, res.DurationMs, res.Stdout, res.Stderr)
	}
}

func TestRunNodeTestsTypecheck(t *testing.T) {
	src := `import { expectTypeOf, test } from "vitest";
const id = <T,>(x: T) => x;
test("id keeps the literal type", () => {
  expectTypeOf(id("a" as const)).toEqualTypeOf<"a">();
  expectTypeOf(id(1)).toEqualTypeOf<string>();
});
`
	res, err := RunNodeTests(context.Background(), Request{Files: map[string]string{"id.test-d.ts": src}}, projectDir(t))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	// The second assertion is wrong on purpose: a type test must fail the run.
	if res.ExitCode == 0 || !strings.Contains(res.Stdout+res.Stderr, "id keeps the literal type") {
		t.Fatalf("want a failing type test, got exit %d\nstdout:\n%s\nstderr:\n%s", res.ExitCode, res.Stdout, res.Stderr)
	}
	assertPlainOutput(t, res)
}

func TestRunNodeTestsPlaywrightInEdge(t *testing.T) {
	src := `import { test, expect } from "@playwright/test";
test("form shows a greeting", async ({ page }) => {
  await page.setContent('<input aria-label="Name"><button onclick="document.querySelector(\'p\').textContent = \'Merhaba \' + document.querySelector(\'input\').value">Greet</button><p></p>');
  await page.getByLabel("Name").fill("Ayşe");
  await page.getByRole("button", { name: "Greet" }).click();
  await expect(page.locator("p")).toHaveText("Merhaba Ayşe");
});
`
	res, err := RunNodeTests(context.Background(), Request{Files: map[string]string{"greet.spec.ts": src}}, projectDir(t))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.ExitCode != 0 || !strings.Contains(res.Stdout, "form shows a greeting") || !strings.Contains(res.Stdout, "1 passed") {
		t.Fatalf("exit %d\nstdout:\n%s\nstderr:\n%s", res.ExitCode, res.Stdout, res.Stderr)
	}
	assertPlainOutput(t, res)
}

// A failing expect is formatted in Playwright's worker, which Playwright
// starts with FORCE_COLOR=1; the runner must still print it without colors.
func TestRunNodeTestsPlaywrightFailureIsPlain(t *testing.T) {
	src := `import { test, expect } from "@playwright/test";
test("heading says Merhaba", async ({ page }) => {
  await page.setContent("<h1>Hello</h1>");
  await expect(page.locator("h1")).toHaveText("Merhaba", { timeout: 500 });
});
`
	res, err := RunNodeTests(context.Background(), Request{Files: map[string]string{"fail.spec.ts": src}}, projectDir(t))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.ExitCode == 0 || !strings.Contains(res.Stdout, "1 failed") || !strings.Contains(res.Stdout, `Expected: "Merhaba"`) {
		t.Fatalf("want a failing test with its expect message, exit %d\nstdout:\n%s\nstderr:\n%s", res.ExitCode, res.Stdout, res.Stderr)
	}
	assertPlainOutput(t, res)
}

func TestRunNodeTestsNeedsProject(t *testing.T) {
	if _, err := RunNodeTests(context.Background(), Request{Files: map[string]string{"a.test.ts": ""}}, ""); err != ErrNoNodeProject {
		t.Fatalf("err = %v, want ErrNoNodeProject", err)
	}
}
