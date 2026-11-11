export interface DocumentContent {
  title: string;
  group: string;
  group_order: number;
  sort_order: number;
  body?: string;
}
export interface PublicArticle extends DocumentContent {
  slug: string;
  published_at: number;
}
export interface AdminArticle {
  id: number;
  slug: string;
  draft: string;
  published: string;
  previous: string;
  version: number;
  updated_by: number;
  published_by: number;
  updated_at: number;
  published_at: number;
}
