/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  // `dark:` variants follow a `dark` class on <html>, not the OS setting
  // alone: the theme is the user's choice (dark, light or system), set by
  // src/shared/theme.ts and, before the first paint, index.html.
  darkMode: 'class',
  theme: {
    extend: {},
  },
  plugins: [],
};
