import js from '@eslint/js';
import globals from 'globals';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';
import tseslint from 'typescript-eslint';

export default tseslint.config(
  // Lint the app only: lesson examples (content/) show buggy code on purpose
  // and are checked by verify:lessons; lesson-libs and .lesson-runs are
  // generated.
  { ignores: ['dist', 'content', 'public/lesson-libs', '.lesson-runs', '.bin'] },
  {
    extends: [js.configs.recommended, ...tseslint.configs.recommended],
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    plugins: {
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      'react-refresh/only-export-components': [
        'warn',
        { allowConstantExport: true },
      ],
    },
  },
  // Layering: app -> features -> shared (lesson ts.architecture.folders).
  // Features meet in app/, where one feature's slot or callback is filled with
  // another's component or function; shared/ knows no feature. The patterns
  // match import strings, so "../x/" from a flat feature folder is a sibling.
  {
    files: ['src/features/**/*.{ts,tsx}'],
    rules: {
      'no-restricted-imports': ['error', { patterns: [
        { regex: '^\\.\\./(?!\\.\\.)', message: "Features don't import each other: compose them in app/." },
        { regex: '(^|/)app/', message: 'features/ must not depend on app/.' },
        { regex: '^@/(features|app)/', message: 'Import your own feature relatively; other features meet in app/.' },
      ] }],
    },
  },
  {
    files: ['src/shared/**/*.{ts,tsx}'],
    rules: {
      'no-restricted-imports': ['error', { patterns: [
        { regex: '(^|/)(features|app)/', message: 'shared/ is the bottom layer: take the value as a prop.' },
      ] }],
    },
  }
);
