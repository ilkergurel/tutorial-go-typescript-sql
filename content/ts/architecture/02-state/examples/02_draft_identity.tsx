// LEARN(ts.architecture.state): a draft is a deliberate copy; give it the identity of what it edits.
import { useEffect, useState } from "react";
import { flushSync } from "react-dom";
import { createRoot } from "react-dom/client";

const saved: Record<string, string> = { folders: "app → features → shared", state: "URL mi, sunucu mu?" };
// Answers after 20 ms, like a request to the backend.
const loadNote = (id: string) =>
  new Promise<string>((resolve) => setTimeout(() => resolve(saved[id] ?? ""), 20));

// The shape of this app's LessonNotes: load into local state when the id changes.
function Notes({ lessonId }: { lessonId: string }) {
  const [body, setBody] = useState("");
  const [loaded, setLoaded] = useState(false);
  useEffect(() => {
    let live = true;
    void loadNote(lessonId).then((text) => {
      if (!live) return;
      setBody(text);
      setLoaded(true);
    });
    return () => {
      live = false;
    };
  }, [lessonId]);
  return <textarea value={body} disabled={!loaded} onChange={(e) => setBody(e.target.value)} />;
}

// The fix is one attribute: a new key is a new component with fresh state.
function Page({ lessonId, keyed }: { lessonId: string; keyed: boolean }) {
  return keyed ? <Notes key={lessonId} lessonId={lessonId} /> : <Notes lessonId={lessonId} />;
}

const tick = () => new Promise((r) => setTimeout(r, 0));
for (const keyed of [false, true]) {
  const host = document.createElement("div");
  const root = createRoot(host);
  const area = () => host.querySelector("textarea")!;
  const show = (when: string) =>
    console.log(`${keyed ? "key   " : "no key"} ${when.padEnd(13)} ${area().disabled ? "disabled" : "editable"} "${area().value}"`);
  const until = async (id: string) => {
    while (area().disabled || area().value !== saved[id]) await tick();
  };

  flushSync(() => root.render(<Page lessonId="folders" keyed={keyed} />));
  await until("folders");
  show("on folders");
  flushSync(() => root.render(<Page lessonId="state" keyed={keyed} />));
  show("just switched");
  await until("state");
  show("loaded");
  root.unmount();
}
