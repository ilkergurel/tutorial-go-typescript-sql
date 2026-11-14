// LEARN(ts.react.effects-hooks): an effect synchronizes with an external system; cleanup undoes setup.
import { StrictMode, useEffect } from "react";
import { flushSync } from "react-dom";
import { createRoot } from "react-dom/client";

// Stands in for a WebSocket or a third-party widget: not managed by React.
function connect(room: string) {
  console.log(`  connect #${room}`);
  return { close: () => console.log(`  close #${room}`) };
}

function ChatRoom({ room }: { room: string }) {
  useEffect(() => {
    const connection = connect(room);
    return () => connection.close();
  }, [room]);
  return <h1>#{room}</h1>;
}

const host = document.createElement("div");
const root = createRoot(host);
function show(room: string) {
  flushSync(() =>
    root.render(
      <StrictMode>
        <ChatRoom room={room} />
      </StrictMode>,
    ),
  );
}

console.log("mount #genel");
show("genel");
console.log("re-render, same room");
show("genel");
console.log("switch to #duyurular");
show("duyurular");
console.log("unmount");
root.unmount();
