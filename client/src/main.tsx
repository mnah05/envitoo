import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import "./index.css";
import { Providers } from "./providers";
import { App } from "./App";

// Social prerender restore: when the backend redirects a real browser from
// /e/:slug to /?next=/e/:slug, put the original path back before the router
// mounts so bookmarks and share links land on the event page.
const next = new URLSearchParams(window.location.search).get("next");
if (
  next &&
  next.startsWith("/") &&
  !next.startsWith("//") &&
  !next.startsWith("/\\")
) {
  try {
    window.history.replaceState(null, "", next);
  } catch {
    // Malformed/cross-origin next value — silently ignored.
  }
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <Providers>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </Providers>
  </StrictMode>,
);
