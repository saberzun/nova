import type { PublicArticle } from "./types";

export function sortDocuments(articles: PublicArticle[]): PublicArticle[] {
  // A group's lowest configured order determines its position consistently.
  const orders = new Map<string, number>();
  for (const item of articles)
    {orders.set(
      item.group,
      Math.min(orders.get(item.group) ?? Infinity, item.group_order),
    );}
  return [...articles].sort(
    (a, b) =>
      (orders.get(a.group) ?? 0) - (orders.get(b.group) ?? 0) ||
      a.group.localeCompare(b.group) ||
      a.sort_order - b.sort_order ||
      a.slug.localeCompare(b.slug),
  );
}
export function documentHeadingId(index: number): string {
  return `document-heading-${index}`;
}
