// @vitest-environment jsdom
import { afterEach, describe, expect, it } from 'vitest';
import { IFRAME_SHIM, iframeDoc } from './browserTs';

// The shim and the document are browser code; jsdom runs the parts that don't
// need ES modules. What needs a real browser (the lesson module, import(),
// CSP) is covered by npm run verify:lessons in headless Edge.

describe('iframeDoc', () => {
  const parse = (html: string) => new DOMParser().parseFromString(html, 'text/html');

  it('leaves <body> empty, so page code sees only what it adds', () => {
    const doc = parse(iframeDoc('console.log(document.body.textContent);', '{"imports":{}}', 'main.ts'));
    expect(doc.body.childNodes).toHaveLength(0);
    expect(doc.body.textContent).toBe('');
  });

  it('puts the CSP, import map, shim, lesson and end watcher in <head>, in that order', () => {
    const doc = parse(iframeDoc('console.log(1);', '{"imports":{}}', 'main.ts'));
    const order = [...doc.head.children].map((el) =>
      el instanceof HTMLScriptElement ? `script:${el.type || 'classic'}${el.src ? ':src' : ''}` : el.localName
    );
    expect(order).toEqual(['meta', 'meta', 'script:importmap', 'script:classic', 'script:module:src', 'script:module']);
    expect(doc.head.querySelector('meta[http-equiv]')?.getAttribute('content')).toMatch(/^default-src 'none';/);
    expect(doc.head.querySelectorAll('script')[1].textContent).toBe(IFRAME_SHIM);
  });

  it('runs the lesson from a data: URL that the watcher imports again', () => {
    // Characters that would break an attribute, a script element or a URL.
    const js = `const s = "</script><b a='1'>&amp;</b> %41 # İı";\nconsole.log(s);`;
    const doc = parse(iframeDoc(js, null, 'my lesson.ts'));
    const [, lesson, watcher] = doc.head.querySelectorAll('script');
    const url = lesson.getAttribute('src') ?? '';
    const prefix = 'data:text/javascript;charset=utf-8,';
    expect(url.startsWith(prefix)).toBe(true);
    expect(decodeURIComponent(url.slice(prefix.length))).toBe(`${js}\n//# sourceURL=my_lesson.ts`);
    // The same URL is the same module: import() settles when the lesson ends.
    expect(watcher.textContent).toContain(`import(${JSON.stringify(url)})`);
  });

  it('adds no import map when the lesson imports no library', () => {
    const doc = parse(iframeDoc('console.log(1);', null, 'main.ts'));
    expect(doc.head.querySelector('script[type="importmap"]')).toBeNull();
  });
});

describe('IFRAME_SHIM', () => {
  type Posted = { type: string; args: string[] };
  type FrameWindow = Window & typeof globalThis & { __stacklabMainDone(): void };
  const cleanups: (() => void)[] = [];
  afterEach(() => cleanups.splice(0).forEach((f) => f()));

  /** A frame running the shim; the parent collects what it posts. */
  function shimFrame() {
    const frame = document.createElement('iframe');
    document.body.append(frame);
    const win = frame.contentWindow as FrameWindow;
    const posted: Posted[] = [];
    const onMessage = (e: MessageEvent<{ __stacklab_run?: true } & Posted>) => {
      if (e.data?.__stacklab_run) posted.push({ type: e.data.type, args: e.data.args });
    };
    window.addEventListener('message', onMessage);
    cleanups.push(() => {
      window.removeEventListener('message', onMessage);
      frame.remove();
    });
    // The shim also passes each call on to the real console: keep it quiet.
    for (const k of ['log', 'info', 'warn', 'error', 'debug'] as const) win.console[k] = () => {};
    win.eval(IFRAME_SHIM);
    // jsdom logs errors nobody handled; the shim's listener has seen them.
    win.addEventListener('error', (e) => e.preventDefault());
    // Code runs inside the frame, as a lesson does: its errors are the
    // frame's own Error objects.
    const run = (code: string) => void win.eval(code);
    return { win, posted, run };
  }

  const settle = (ms: number) => new Promise((r) => setTimeout(r, ms));
  async function until(check: () => boolean) {
    for (let i = 0; i < 100 && !check(); i++) await settle(10);
    expect(check()).toBe(true);
  }
  const has = (posted: Posted[], type: string, text?: string) =>
    posted.some((p) => p.type === type && (text === undefined || p.args.join(' ') === text));

  it('reports an error thrown in a timer callback without ending the run', async () => {
    const { win, posted, run } = shimFrame();
    run(`setTimeout(function () { throw new Error('timer boom'); }, 0);`);
    await until(() => has(posted, 'uncaught', 'Uncaught Error: timer boom'));
    await settle(50);
    expect(has(posted, 'done')).toBe(false); // the module hasn't finished
    win.__stacklabMainDone();
    await until(() => has(posted, 'done'));
  });

  it('reports an error thrown in a listener; the dispatching code goes on', async () => {
    const { win, posted, run } = shimFrame();
    run(`
      var button = document.createElement('button');
      button.addEventListener('click', function () { throw new Error('listener boom'); });
      button.click();
      console.log('after click');`);
    await until(() => has(posted, 'log', 'after click'));
    expect(has(posted, 'uncaught', 'Uncaught Error: listener boom')).toBe(true);
    await settle(50);
    expect(has(posted, 'done')).toBe(false);
    win.__stacklabMainDone();
    await until(() => has(posted, 'done'));
  });

  it('ends the run once the main part is over and no timer is pending', async () => {
    const { posted, run } = shimFrame();
    run(`setTimeout(function () { console.log('late'); }, 30); __stacklabMainDone();`);
    await until(() => has(posted, 'done'));
    expect(posted.map((p) => p.type)).toEqual(['log', 'done']);
  });

  it('ends the run after a timer error when the main part is over', async () => {
    const { posted, run } = shimFrame();
    run(`setTimeout(function () { throw new Error('late boom'); }, 10); __stacklabMainDone();`);
    await until(() => has(posted, 'done'));
    expect(posted.map((p) => p.type)).toEqual(['uncaught', 'done']);
  });

  it('ignores an error React replays in its invokeguardedcallback event', async () => {
    const { posted, run } = shimFrame();
    run(`
      var fake = document.createElement('react');
      fake.addEventListener('react-invokeguardedcallback', function () { throw new Error('replayed'); });
      fake.dispatchEvent(new Event('react-invokeguardedcallback'));
      console.log('after replay');`);
    await until(() => has(posted, 'log', 'after replay'));
    await settle(20);
    expect(has(posted, 'uncaught')).toBe(false);
  });
});
