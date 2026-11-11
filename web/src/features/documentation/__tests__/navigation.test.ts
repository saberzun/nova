import assert from "node:assert/strict";
import test from "node:test";
import { sortDocuments, documentHeadingId } from "../navigation.ts";
import type { PublicArticle } from "../types.ts";

const article = (
  slug: string,
  group: string,
  groupOrder: number,
  order: number,
): PublicArticle => ({
  slug,
  group,
  group_order: groupOrder,
  sort_order: order,
  title: slug,
  published_at: 0,
});
test("groups remain together when an article has a different group order", () => {
  const input = [
    article("a2", "A", 8, 2),
    article("b", "B", 2, 1),
    article("a1", "A", 1, 1),
  ];
  assert.deepEqual(
    sortDocuments(input).map((item) => item.slug),
    ["a1", "a2", "b"],
  );
  assert.equal(input[0].slug, "a2");
});
test("empty catalog and duplicate heading labels have stable independent anchors", () => {
  assert.deepEqual(sortDocuments([]), []);
  assert.notEqual(documentHeadingId(0), documentHeadingId(1));
});
