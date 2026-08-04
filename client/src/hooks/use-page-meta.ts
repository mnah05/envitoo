import { useEffect } from "react";

export const DEFAULT_OG_IMAGE = "/og-default.png";

interface PageMetaOptions {
  title: string;
  description?: string;
  image?: string;
  url?: string;
  type?: string;
}

function setMeta(attr: "name" | "property", key: string, content: string) {
  const selector = `meta[${attr}="${key}"]`;
  let el = document.head.querySelector<HTMLMetaElement>(selector);
  if (!el) {
    el = document.createElement("meta");
    el.setAttribute(attr, key);
    document.head.appendChild(el);
  }
  el.setAttribute("content", content);
}

function absoluteUrl(value: string): string {
  if (/^https?:\/\//.test(value)) return value;
  return `${window.location.origin}${value.startsWith("/") ? value : `/${value}`}`;
}

/**
 * Sets document.title plus Open Graph / Twitter card meta tags for the
 * current page. Defaults the social image to the site-wide og-default so
 * every page (not just events) previews cleanly when shared.
 */
export function usePageMeta({ title, description, image, url, type }: PageMetaOptions) {
  useEffect(() => {
    document.title = title;

    const desc = description ?? "Host and discover events people actually show up for.";
    const socialImage = image ? absoluteUrl(image) : absoluteUrl(DEFAULT_OG_IMAGE);
    const canonical = url ? absoluteUrl(url) : window.location.href;

    setMeta("property", "og:site_name", "EnviToo");
    setMeta("property", "og:title", title);
    setMeta("property", "og:description", desc);
    setMeta("property", "og:type", type ?? "website");
    setMeta("property", "og:url", canonical);
    setMeta("property", "og:image", socialImage);
    setMeta("name", "description", desc);
    setMeta("name", "twitter:card", "summary_large_image");
    setMeta("name", "twitter:title", title);
    setMeta("name", "twitter:description", desc);
    setMeta("name", "twitter:image", socialImage);
  }, [title, description, image, url, type]);
}
