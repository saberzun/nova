import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useSearch } from "@tanstack/react-router";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { PublicLayout } from "@/components/layout";
import { Button } from "@/components/ui/button";
import { getDocumentCatalog, getDocument } from "./api";
import { DocumentationView } from "./document-view";
import { sortDocuments } from "./navigation";

export function DocsPage() {
  const { t } = useTranslation();
  const search = useSearch({ strict: false }) as { article?: string };
  const catalog = useQuery({
    queryKey: ["documentation", "catalog"],
    queryFn: getDocumentCatalog,
    staleTime: 0,
  });
  const items = sortDocuments(catalog.data ?? []);
  const slug = search.article || items[0]?.slug;
  const article = useQuery({
    queryKey: ["documentation", "article", slug],
    queryFn: () => getDocument(slug ?? ""),
    enabled: Boolean(slug),
    placeholderData: keepPreviousData,
    staleTime: 0,
  });
  let content: ReactNode;
  if (catalog.isError || article.isError) {
    content = (
      <div className="py-16 text-center" role="alert">
        <p>{t("Document unavailable")}</p>
        <Button
          variant="outline"
          className="mt-4"
          onClick={() => {
            void catalog.refetch();
            if (slug) void article.refetch();
          }}
        >
          {t("Retry")}
        </Button>
      </div>
    );
  } else if (catalog.isPending || (slug && article.isPending)) {
    content = (
      <p className="py-16" role="status">
        {t("Loading...")}
      </p>
    );
  } else if (article.data) {
    content = <DocumentationView article={article.data} catalog={items} />;
  } else {
    content = (
      <p className="py-16 text-center">{t("No published documents yet")}</p>
    );
  }
  return (
    <PublicLayout>
      <div className="mx-auto w-full max-w-7xl px-4 sm:px-6">{content}</div>
    </PublicLayout>
  );
}
