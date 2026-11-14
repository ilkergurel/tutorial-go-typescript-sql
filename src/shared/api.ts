/**
 * Backend API base, for every feature's client (runner, learning).
 *
 * Empty by default: requests stay same-origin and Vite's dev proxy forwards
 * them. Set VITE_API_URL only for a backend that sends CORS headers (it
 * currently does not). It lives in shared/ because it is transport
 * configuration, not part of any one feature; it used to sit in
 * features/runner/types.ts, which made learning import runner.
 */
export const API_BASE: string =
  (import.meta.env.VITE_API_URL as string | undefined) ?? '';
