package gorun

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// ErrNoNodeProject means ts-test lessons can't run here: they need the app's
// node_modules (Vitest, Testing Library, MSW, Playwright), which only local
// mode has. The Docker runner image doesn't.
var ErrNoNodeProject = errors.New("gorun: ts-test needs the app's node_modules (local mode)")

// RunNodeTests runs a ts-test lesson file with the real tools: Vitest (jsdom,
// Testing Library, MSW; --typecheck for *.test-d.ts type tests) or, when the
// file imports @playwright/test, Playwright driving the installed Microsoft
// Edge (channel msedge), so no browser download is needed.
//
// The packages are the app's own devDependencies, resolved the normal Node
// way: walking up from the test file to projectDir/node_modules. That is why
// the work dir lives in projectDir/.lesson-runs rather than the system temp
// dir. Rejected alternative: a separate node_modules per run, which would mean
// an npm install (network, minutes) for every Run.
func RunNodeTests(ctx context.Context, req Request, projectDir string) (Result, error) {
	if projectDir == "" {
		return Result{}, ErrNoNodeProject
	}
	base := filepath.Join(projectDir, ".lesson-runs")
	if err := os.MkdirAll(base, 0o700); err != nil {
		return Result{}, fmt.Errorf("gorun: lesson runs dir: %w", err)
	}

	playwright := mentionsAny(req.Files, `"@playwright/test"`, `'@playwright/test'`)
	scaffold, args := vitestScaffold, []string{filepath.Join(projectDir, "node_modules", "vitest", "vitest.mjs"), "run"}
	if playwright {
		scaffold = playwrightScaffold
		args = []string{filepath.Join(projectDir, "node_modules", "@playwright", "test", "cli.js"), "test"}
	} else if hasTypeTests(req.Files) {
		args = append(args, "--typecheck")
	}

	workDir, err := newWorkDirIn(base, req, scaffold)
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(workDir)

	// No colors: output is compared as text and shown as plain text. Each tool
	// has its own switch, and only that one may be set (both are dropped from
	// the inherited environment first):
	//   - Vitest turns colors on for CI and on Windows unless NO_COLOR is set.
	//   - Playwright starts its workers with FORCE_COLOR=1 and strips their
	//     color codes in the runner when the runner has FORCE_COLOR=0. A
	//     NO_COLOR next to the workers' FORCE_COLOR=1 makes Node in every
	//     worker warn that NO_COLOR is ignored, into the learner's output.
	colorOff := "NO_COLOR=1"
	if playwright {
		colorOff = "FORCE_COLOR=0"
	}
	env := append(withoutEnv(os.Environ(), "NO_COLOR", "FORCE_COLOR"), colorOff, "CI=1")
	env = append(env, req.Env...)
	res := runProcess(ctx, workDir, testTTL(req), "", env, "node", args...)
	res.Files = listWritten(workDir, req, scaffold)
	return res, nil
}

// withoutEnv drops the named variables from env, comparing names without
// regard to case, as Windows does.
func withoutEnv(env []string, names ...string) []string {
	return slices.DeleteFunc(env, func(kv string) bool {
		name, _, _ := strings.Cut(kv, "=")
		return slices.ContainsFunc(names, func(n string) bool { return strings.EqualFold(n, name) })
	})
}

// mentionsAny reports whether any file contains one of the strings (here, an
// import specifier in quotes).
func mentionsAny(files map[string]string, needles ...string) bool {
	for _, src := range files {
		for _, n := range needles {
			if strings.Contains(src, n) {
				return true
			}
		}
	}
	return false
}

func hasTypeTests(files map[string]string) bool {
	for name := range files {
		if strings.Contains(name, ".test-d.") || strings.Contains(name, ".spec-d.") {
			return true
		}
	}
	return false
}

// tsconfigForTests: strict checking for --typecheck, and the React 17+ JSX
// transform so .tsx tests need no `import React`.
const tsconfigForTests = `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "jsx": "react-jsx",
    "strict": true,
    "noEmit": true,
    "skipLibCheck": true,
    "types": []
  }
}
`

var vitestScaffold = map[string]string{
	"package.json":  "{\"type\": \"module\"}\n",
	"tsconfig.json": tsconfigForTests,
	// jsdom gives Testing Library a DOM; verbose lists every test by name.
	"vitest.config.mjs": `export default {
  esbuild: { jsx: 'automatic' },
  test: { environment: 'jsdom', watch: false, reporters: ['verbose'] },
};
`,
}

var playwrightScaffold = map[string]string{
	"package.json":  "{\"type\": \"module\"}\n",
	"tsconfig.json": tsconfigForTests,
	// The installed Edge, headless, one worker so output order is stable;
	// tests build their pages with page.setContent or page.route, no server.
	"playwright.config.mjs": `export default {
  testDir: '.',
  reporter: [['list']],
  workers: 1,
  timeout: 10000,
  outputDir: './test-results',
  use: { channel: 'msedge', headless: true },
};
`,
}
