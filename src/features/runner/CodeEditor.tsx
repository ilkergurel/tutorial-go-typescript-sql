/**
 * CodeEditor: Monaco from the installed monaco-editor package.
 *
 * @monaco-editor/react's loader fetches Monaco from cdn.jsdelivr.net unless it
 * is given an instance, which made the editor need the network and run a
 * different Monaco than package.json names. loader.config({ monaco }) hands it
 * the bundled one; Vite builds the two workers the editor uses (the base
 * editor worker and the TypeScript language service) from the ?worker
 * imports. RunPanel imports this module lazily, so Monaco's few megabytes
 * stay out of the first chunk, as they were with the CDN.
 */
import { useCallback, useEffect, useRef } from 'react';
import Editor, { loader, type OnMount } from '@monaco-editor/react';
import * as monaco from 'monaco-editor';
import EditorWorker from 'monaco-editor/editor/editor.worker?worker';
import TsWorker from 'monaco-editor/language/typescript/ts.worker?worker';
import { useTheme } from '../../shared/useTheme';

self.MonacoEnvironment = {
  // Labels are Monaco's language ids; Go and SQL need only the base worker.
  getWorker: (_workerId, label) =>
    label === 'typescript' || label === 'javascript' ? new TsWorker() : new EditorWorker(),
};

const ts = monaco.typescript;
ts.typescriptDefaults.setCompilerOptions({
  ...ts.typescriptDefaults.getCompilerOptions(),
  module: ts.ModuleKind.ESNext,
  // Parse JSX in .tsx models without needing React's types or JSX runtime.
  jsx: ts.JsxEmit.Preserve,
  // Every lesson file is a module, as esbuild and Node treat it. Without
  // this, a file with no import is a script, and `const name` collides with
  // the DOM's global `name`. (3 is ModuleDetectionKind.Force; Monaco doesn't
  // export that enum.)
  moduleDetection: 3,
});
ts.typescriptDefaults.setDiagnosticsOptions({
  ...ts.typescriptDefaults.getDiagnosticsOptions(),
  // "Cannot find module": the editor has no types for the lesson libraries
  // (react, zod, ...) or node: modules, which do resolve when the code runs.
  diagnosticCodesToIgnore: [2307, 2792],
});

loader.config({ monaco });

export default function CodeEditor({ value, onChange, language, path, onRunShortcut }: {
  value: string;
  onChange: (value: string) => void;
  language: 'go' | 'sql' | 'typescript';
  /** A model URI ending in the example's file name, so .tsx parses as TSX. */
  path: string;
  onRunShortcut: () => void;
}) {
  // Monaco registers the command once, at mount; a ref lets it call the
  // latest handler instead of the one from the first render, whose closure
  // still holds the code as it was when the editor opened.
  const runShortcut = useRef(onRunShortcut);
  useEffect(() => {
    runShortcut.current = onRunShortcut;
  }, [onRunShortcut]);

  // Monaco draws its own colours, so it can't follow the page's `dark` class;
  // a new theme prop makes @monaco-editor/react call setTheme, live.
  const { resolved } = useTheme();

  const onMount = useCallback<OnMount>((editor, m) => {
    editor.addCommand(m.KeyMod.CtrlCmd | m.KeyCode.Enter, () => runShortcut.current());
  }, []);

  return (
    <Editor
      height="320px"
      language={language}
      path={path}
      value={value}
      theme={resolved === 'dark' ? 'vs-dark' : 'vs'}
      onChange={(v) => onChange(v ?? '')}
      onMount={onMount}
      options={{
        minimap: { enabled: false },
        fontSize: 13,
        lineNumbers: 'on',
        scrollBeyondLastLine: false,
        automaticLayout: true,
        tabSize: 2,
      }}
    />
  );
}
