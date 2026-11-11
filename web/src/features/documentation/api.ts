import { api } from "@/lib/api";
import type { AdminArticle, DocumentContent, PublicArticle } from "./types";

interface Response<T> {
  success: boolean;
  message?: string;
  data: T;
}
async function read<T>(response: Response<T>): Promise<T> {
  if (!response.success)
    {throw new Error(response.message || "Operation failed");}
  return response.data;
}
export async function getDocumentCatalog(): Promise<PublicArticle[]> {
  return read((await api.get("/api/docs")).data);
}
export async function getDocument(slug: string): Promise<PublicArticle> {
  return read(
    (await api.get(`/api/docs/articles/${encodeURIComponent(slug)}`)).data,
  );
}
export async function getAdminDocuments(): Promise<AdminArticle[]> {
  return read((await api.get("/api/admin/documentation")).data);
}
export async function createDocument(
  slug: string,
  content: DocumentContent,
): Promise<AdminArticle> {
  return read(
    (await api.post("/api/admin/documentation", { slug, content })).data,
  );
}
export async function changeDocument(
  article: AdminArticle,
  action: string,
  content?: DocumentContent,
): Promise<AdminArticle> {
  return read(
    (
      await api.post(`/api/admin/documentation/${article.id}/${action}`, {
        version: article.version,
        content,
      })
    ).data,
  );
}
export async function uploadDocumentImage(
  file: File,
): Promise<{ url: string }> {
  const form = new FormData();
  form.append("file", file);
  return read((await api.post("/api/admin/documentation/images", form)).data);
}
