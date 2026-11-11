import { Link } from "@tanstack/react-router";
import DOMPurify from "dompurify";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";

import { Markdown } from "@/components/ui/markdown";
import {
  Dialog,
  DialogContent,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { documentHeadingId, sortDocuments } from "./navigation";
import type { PublicArticle } from "./types";

interface Heading {
  id: string;
  text: string;
  level: number;
}

export function DocumentationView(props: {
  article: PublicArticle;
  catalog: PublicArticle[];
  preview?: boolean;
}) {
  const { t } = useTranslation();
  const content = useRef<HTMLDivElement>(null);
  const [headings, setHeadings] = useState<Heading[]>([]);
  const [image, setImage] = useState<{ src: string; alt: string } | null>(null);
  const articles = sortDocuments(props.catalog);
  const groups = [...new Set(articles.map((item) => item.group))];

  useEffect(() => {
    const root = content.current;
    if (!root) return;
    let cancelled = false;
    const cleanup: Array<() => void> = [];
    setHeadings(
      [...root.querySelectorAll("h1,h2,h3")].map((element, index) => {
        element.id = documentHeadingId(index);
        element.classList.add("scroll-mt-24");
        return {
          id: element.id,
          text: element.textContent ?? "",
          level: Number(element.tagName.slice(1)),
        };
      }),
    );
    for (const pre of root.querySelectorAll("pre")) {
      const code = pre.querySelector("code");
      if (!code) continue;
      const source = code.textContent ?? "";
      const button = document.createElement("button");
      button.type = "button";
      button.textContent = t("Copy code");
      button.className =
        "mb-3 block rounded border px-2 py-1 text-xs hover:bg-accent";
      button.setAttribute("aria-label", t("Copy code"));
      const copy = () => {
        void navigator.clipboard.writeText(source).then(
          () => toast.success(t("Copied")),
          () => toast.error(t("Copy failed")),
        );
      };
      button.addEventListener("click", copy);
      pre.prepend(button);
      cleanup.push(() => {
        button.removeEventListener("click", copy);
        button.remove();
      });
      const language = code.className.replace("language-", "").trim() || "text";
      void import("shiki/bundle/web")
        .then(async ({ codeToHtml }) => {
          const html = await codeToHtml(source, {
            lang: language,
            theme: "github-light",
          });
          if (cancelled || !code.isConnected) return;
          const template = document.createElement("template");
          template.innerHTML = DOMPurify.sanitize(html);
          const highlighted = template.content.querySelector("code");
          if (highlighted) code.replaceChildren(...highlighted.childNodes);
        })
        .catch(() => {
          /* Unknown languages retain safely rendered plain text. */
        });
    }
    for (const img of root.querySelectorAll("img")) {
      img.loading = "lazy";
      img.classList.add("max-w-full", "rounded-lg", "cursor-zoom-in");
      const zoom = () => setImage({ src: img.src, alt: img.alt });
      const button = document.createElement("button");
      button.type = "button";
      button.className = "block max-w-full";
      button.setAttribute("aria-label", `${t("Enlarge image")}: ${img.alt}`);
      img.replaceWith(button);
      button.append(img);
      button.addEventListener("click", zoom);
      cleanup.push(() => {
        button.removeEventListener("click", zoom);
        if (button.isConnected) button.replaceWith(img);
      });
    }
    return () => {
      cancelled = true;
      cleanup.forEach((fn) => fn());
    };
  }, [props.article.body, t]);

  const navigation = (
    <nav aria-label={t("Documentation navigation")} className="space-y-6">
      {groups.map((group) => (
        <div key={group}>
          <p className="mb-2 text-xs font-semibold tracking-wide text-muted-foreground">
            {group}
          </p>
          <ul className="space-y-1">
            {articles
              .filter((item) => item.group === group)
              .map((item) => (
                <li key={item.slug}>
                  {props.preview ? (
                    <span className="block rounded-lg px-3 py-2 text-sm">
                      {item.title}
                    </span>
                  ) : (
                    <Link
                      to="/docs"
                      search={{ article: item.slug }}
                      aria-current={
                        item.slug === props.article.slug ? "page" : undefined
                      }
                      className={`block rounded-lg px-3 py-2 text-sm ${item.slug === props.article.slug ? "bg-primary/10 font-medium text-primary" : "text-muted-foreground hover:bg-accent"}`}
                    >
                      {item.title}
                    </Link>
                  )}
                </li>
              ))}
          </ul>
        </div>
      ))}
    </nav>
  );
  return (
    <div className="grid min-w-0 gap-8 py-8 lg:grid-cols-[220px_minmax(0,1fr)] xl:grid-cols-[220px_minmax(0,1fr)_190px]">
      <aside className="hidden lg:block">
        <div className="sticky top-24 max-h-[calc(100vh-8rem)] overflow-y-auto pr-3">
          {navigation}
        </div>
      </aside>
      <main className="min-w-0">
        <details className="mb-6 rounded-lg border p-3 lg:hidden">
          <summary className="cursor-pointer font-medium">
            {t("Documentation navigation")}
          </summary>
          <div className="mt-4">{navigation}</div>
        </details>
        <p className="mb-3 text-sm font-medium text-primary">
          {props.article.group}
        </p>
        <h1 className="text-3xl font-bold tracking-tight md:text-4xl">
          {props.article.title}
        </h1>
        {props.article.published_at > 0 && (
          <p className="mt-3 text-xs text-muted-foreground">
            {t("Published at")}:{" "}
            {new Date(props.article.published_at * 1000).toLocaleString()}
          </p>
        )}
        <div
          ref={content}
          className="mt-8 min-w-0 [&_blockquote]:border-primary/40 [&_blockquote]:bg-primary/5 [&_blockquote]:rounded-r-lg [&_blockquote]:py-3 [&_blockquote]:pr-4 [&_h2]:mt-10 [&_h2]:border-b [&_h2]:pb-3 [&_pre]:bg-white [&_pre]:text-slate-900"
        >
          <Markdown>{props.article.body ?? ""}</Markdown>
        </div>
      </main>
      <aside className="hidden xl:block">
        <nav
          aria-label={t("On this page")}
          className="sticky top-24 max-h-[calc(100vh-8rem)] space-y-3 overflow-y-auto border-l pl-4 text-sm"
        >
          <p className="font-semibold">{t("On this page")}</p>
          {headings.map((heading) => (
            <a
              key={heading.id}
              href={`#${heading.id}`}
              className={`block text-muted-foreground hover:text-primary ${heading.level === 3 ? "pl-3" : ""}`}
            >
              {heading.text}
            </a>
          ))}
        </nav>
      </aside>
      <Dialog
        open={Boolean(image)}
        onOpenChange={(open) => {
          if (!open) setImage(null);
        }}
      >
        <DialogContent className="max-h-[95vh] overflow-auto sm:max-w-[90vw]">
          <DialogTitle>{image?.alt || t("Enlarge image")}</DialogTitle>
          <DialogDescription className="sr-only">
            {t("Image preview")}
          </DialogDescription>
          {image && (
            <img
              src={image.src}
              alt={image.alt}
              className="mx-auto max-h-[80vh] max-w-full object-contain"
            />
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
