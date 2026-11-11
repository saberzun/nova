import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import test from "node:test";
import vm from "node:vm";
import { transformSync } from "esbuild";
import React from "react";
import { renderToStaticMarkup } from "react-dom/server";

const require = createRequire(import.meta.url);
const load = (path, dependencies) => {
  const source = readFileSync(new URL(path, import.meta.url), "utf8");
  const code = transformSync(source, {
    loader: "tsx",
    format: "cjs",
    jsx: "automatic",
  }).code;
  const module = { exports: {} };
  vm.runInNewContext(code, {
    module,
    exports: module.exports,
    require: (name) => dependencies[name] ?? require(name),
  });
  return module.exports;
};
const navigation = load("../navigation.ts", {});
const box = (props) => React.createElement("div", null, props.children);
const documentLinks = [];
const { DocumentationView } = load("../document-view.tsx", {
  "@tanstack/react-router": {
    Link: (props) => {
      documentLinks.push(props);
      return React.createElement(
        "a",
        {
          href: props.to + "?article=" + props.search.article,
          "aria-current": props["aria-current"],
        },
        props.children,
      );
    },
  },
  "react-i18next": { useTranslation: () => ({ t: (key) => key }) },
  "./navigation": navigation,
  "@/components/ui/markdown": {
    Markdown: (props) => React.createElement("div", null, props.children),
  },
  "@/components/ui/dialog": {
    Dialog: () => null,
    DialogContent: box,
    DialogTitle: box,
    DialogDescription: box,
  },
});
const article = {
  slug: "quick-start",
  title: "<script>long title</script>",
  group: "Guides",
  group_order: 0,
  sort_order: 0,
  body: "# Body",
  published_at: 0,
};
test("public document has mobile navigation, desktop table of contents and escaped titles", () => {
  const html = renderToStaticMarkup(
    React.createElement(DocumentationView, { article, catalog: [article] }),
  );
  assert.match(html, /<details/);
  assert.match(html, /aria-label="Documentation navigation"/);
  assert.match(html, /aria-label="On this page"/);
  assert.match(html, /aria-current="page"/);
  assert.match(html, /href="\/docs\?article=quick-start"/);
  assert.doesNotMatch(html, /<script>/);
});
test("draft preview uses the same article view without public navigation links", () => {
  const html = renderToStaticMarkup(
    React.createElement(DocumentationView, {
      article,
      catalog: [article],
      preview: true,
    }),
  );
  assert.match(html, /# Body/);
  assert.doesNotMatch(html, /href="\/docs\?article=/);
});

test("document navigation delegates article switching to the SPA router", () => {
  documentLinks.length = 0;
  renderToStaticMarkup(
    React.createElement(DocumentationView, { article, catalog: [article] }),
  );
  assert.equal(documentLinks.length, 2);
  assert.equal(documentLinks[0].to, "/docs");
  assert.equal(documentLinks[0].search.article, "quick-start");
});
