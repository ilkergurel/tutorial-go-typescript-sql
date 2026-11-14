// LEARN(ts.react.routing): the URL owns navigation state; component state dies with the component.
import { useState } from "react";
import { flushSync } from "react-dom";
import { createRoot } from "react-dom/client";
import { Link, MemoryRouter, Route, Routes } from "react-router-dom";
import { useLocation, useNavigate, useParams } from "react-router-dom";
import { useSearchParams } from "react-router-dom";

type Lesson = { id: string; title: string; level: "basic" | "advanced" };
const lessons: Lesson[] = [
  { id: "sablonlar", title: "Şablonlar", level: "basic" },
  { id: "defer", title: "defer", level: "basic" },
  { id: "kanallar", title: "Kanallar", level: "advanced" },
  { id: "sozlukler", title: "Sözlükler", level: "basic" },
];
const collator = new Intl.Collator("tr");

function LessonList() {
  const [params, setParams] = useSearchParams(); // lives in the URL
  const [sorted, setSorted] = useState(false); // lives in this component
  const level = params.get("level"); // string | null: the URL is untyped input
  const shown = lessons.filter((l) => level === null || l.level === level);
  if (sorted) shown.sort((a, b) => collator.compare(a.title, b.title));
  return (
    <nav>
      <button onClick={() => setParams({ level: "basic" })}>Temel</button>
      <button onClick={() => setSorted(true)}>A-Z</button>
      {shown.map((l) => (
        <Link key={l.id} to={`/lessons/${l.id}`}>
          {l.title}
        </Link>
      ))}
    </nav>
  );
}

function LessonPage() {
  const { id } = useParams(); // string | undefined: matched from "/lessons/:id"
  const navigate = useNavigate();
  return <button onClick={() => navigate(-1)}>{`${id}: geri`}</button>;
}

function Url() {
  const { pathname, search } = useLocation();
  return <code>{pathname + search}</code>;
}

const host = document.createElement("div");
const root = createRoot(host);
flushSync(() =>
  root.render(
    // No address bar in this iframe: MemoryRouter keeps history in memory.
    // useTransitions={false}: apply navigations synchronously, for flushSync.
    <MemoryRouter initialEntries={["/lessons"]} useTransitions={false}>
      <Routes>
        <Route path="/lessons" element={<LessonList />} />
        <Route path="/lessons/:id" element={<LessonPage />} />
      </Routes>
      <Url />
    </MemoryRouter>,
  ),
);

function step(label: string) {
  if (label !== "start") {
    const all = [...host.querySelectorAll<HTMLElement>("a, button")];
    flushSync(() => all.find((el) => el.textContent === label)!.click());
  }
  const url = host.querySelector("code")!.textContent!;
  const links = [...host.querySelectorAll("a")].map((a) => a.textContent);
  const shown = links.join(", ") || "(lesson page)";
  console.log(`${label.padEnd(16)}${url.padEnd(22)}${shown}`);
}
["start", "Temel", "A-Z", "Sözlükler", "sozlukler: geri"].forEach(step);
root.unmount();
