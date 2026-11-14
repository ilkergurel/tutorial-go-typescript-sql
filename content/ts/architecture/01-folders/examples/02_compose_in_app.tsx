// LEARN(ts.architecture.folders): layers inside a feature; features meet only in app/.
import { useEffect, useState, type ReactNode } from "react";
import { flushSync } from "react-dom";
import { createRoot } from "react-dom/client";

// shared/api/http.ts: the only module that knows the transport.
type Get = (path: string) => Promise<unknown>;

// features/progress/api.ts: turns HTTP into the feature's own types.
type Progress = { status: "viewed" | "completed" };
const progressApi = (get: Get) => ({
  // Real code validates here (lesson ts.services.zod); the cast is the shortcut.
  status: async (lessonId: string) => (await get(`/api/progress/${lessonId}`)) as Progress,
});
type ProgressApi = ReturnType<typeof progressApi>;

// features/progress/useProgress.ts: state. Knows the API client, not the UI.
function useProgress(api: ProgressApi, lessonId: string): Progress | null {
  const [progress, setProgress] = useState<Progress | null>(null);
  useEffect(() => {
    let live = true;
    void api.status(lessonId).then((p) => {
      if (live) setProgress(p);
    });
    return () => {
      live = false;
    };
  }, [api, lessonId]);
  return progress;
}

// features/progress/ProgressBadge.tsx: UI. Knows the hook, not HTTP.
function ProgressBadge({ api, lessonId }: { api: ProgressApi; lessonId: string }) {
  const p = useProgress(api, lessonId);
  if (p === null) return <small>…</small>;
  return <small>{p.status === "completed" ? "tamamlandı" : "okundu"}</small>;
}

// features/lessons/LessonView.tsx: knows nothing about progress; it leaves a slot.
function LessonView({ title, badge }: { title: string; badge: ReactNode }) {
  return <h1>{title} {badge}</h1>;
}

// app/routes/LessonPage.tsx: the only module that imports both features.
function LessonPage({ api }: { api: ProgressApi }) {
  const badge = <ProgressBadge api={api} lessonId="ts.architecture.folders" />;
  return <LessonView title="Klasörler" badge={badge} />;
}

// main.tsx passes the real transport; a fake stands in here, as in a test.
const fakeGet: Get = async (path) => {
  console.log(`GET ${path}`);
  return { status: "completed" };
};
const host = document.createElement("div");
const root = createRoot(host);
flushSync(() => root.render(<LessonPage api={progressApi(fakeGet)} />));
console.log(host.textContent);
while (!host.textContent!.includes("tamam")) await new Promise((r) => setTimeout(r, 0));
console.log(host.textContent);
root.unmount();
