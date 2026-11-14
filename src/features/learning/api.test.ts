import { describe, expect, it } from 'vitest';
import { backendReachable } from './api';

describe('backendReachable', () => {
  const answering = (status: number): typeof fetch => async () => new Response('', { status });

  it('is true when /healthz answers OK', async () => {
    expect(await backendReachable(answering(200))).toBe(true);
  });

  // With the backend down, Vite's dev proxy still answers, with a 500: the
  // localStorage fallback must kick in then too.
  it('is false when the proxy answers with an error status', async () => {
    expect(await backendReachable(answering(502))).toBe(false);
    expect(await backendReachable(answering(500))).toBe(false);
  });

  it('is false when the request fails outright', async () => {
    const failing: typeof fetch = async () => {
      throw new TypeError('fetch failed');
    };
    expect(await backendReachable(failing)).toBe(false);
  });
});
