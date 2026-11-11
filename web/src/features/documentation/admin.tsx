import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";

import { SectionPageLayout } from "@/components/layout";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import {
  getAdminDocuments,
  createDocument,
  changeDocument,
  uploadDocumentImage,
} from "./api";
import { DocumentationView } from "./document-view";
import type { AdminArticle, DocumentContent, PublicArticle } from "./types";

const schema = z.object({
  slug: z
    .string()
    .min(1)
    .max(160)
    .regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/),
  title: z.string().trim().min(1).max(160),
  group: z.string().trim().min(1).max(80),
  group_order: z.number().int().min(-100000).max(100000),
  sort_order: z.number().int().min(-100000).max(100000),
  body: z.string().max(200000),
});
type Form = z.infer<typeof schema>;
type Action = "publish" | "withdraw" | "restore";
const empty: Form = {
  slug: "",
  title: "",
  group: "",
  group_order: 0,
  sort_order: 0,
  body: "",
};

export function DocumentationAdmin() {
  const { t } = useTranslation();
  const client = useQueryClient();
  const query = useQuery({
    queryKey: ["documentation", "admin"],
    queryFn: getAdminDocuments,
    staleTime: Infinity,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [dirty, setDirty] = useState(false);
  const [nextSelection, setNextSelection] = useState<{
    id: number | null;
  } | null>(null);
  const article = query.data?.find((item) => item.id === selectedId);
  const select = (id: number | null) => {
    if (id === selectedId) return;
    if (dirty) setNextSelection({ id });
    else setSelectedId(id);
  };
  const saved = (updated: AdminArticle) => {
    client.setQueryData<AdminArticle[]>(
      ["documentation", "admin"],
      (current) => {
        const list = current ?? [];
        return [...list.filter((item) => item.id !== updated.id), updated].sort(
          (a, b) => a.id - b.id,
        );
      },
    );
    void client.invalidateQueries({ queryKey: ["documentation", "catalog"] });
    void client.invalidateQueries({ queryKey: ["documentation", "article"] });
    setSelectedId(updated.id);
    setDirty(false);
  };
  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t("Documentation management")}
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <p className="mb-5 text-sm text-muted-foreground">
          {t(
            "Edit Markdown, save a draft, preview, then publish. Public pages only show published content.",
          )}
        </p>
        {query.isPending && <p>{t("Loading...")}</p>}
        {query.isError && (
          <div role="alert">
            <p>{t("Loading failed")}</p>
            <Button onClick={() => void query.refetch()}>{t("Retry")}</Button>
          </div>
        )}
        {!query.isPending && !query.isError && (
          <div className="grid min-w-0 gap-6 lg:grid-cols-[240px_minmax(0,1fr)]">
            <aside className="space-y-3">
              <Button className="w-full" onClick={() => select(null)}>
                {t("New document")}
              </Button>
              <nav
                aria-label={t("Documents")}
                className="max-h-[65vh] space-y-2 overflow-y-auto"
              >
                {(query.data ?? []).map((item) => {
                  const content = JSON.parse(item.draft) as DocumentContent;
                  return (
                    <button
                      key={item.id}
                      type="button"
                      onClick={() => select(item.id)}
                      aria-current={selectedId === item.id ? "page" : undefined}
                      className={`block w-full rounded-lg border p-3 text-left ${selectedId === item.id ? "border-primary bg-primary/5" : "hover:bg-accent"}`}
                    >
                      <span className="block break-words font-medium">
                        {content.title}
                      </span>
                      <span className="mt-1 block text-xs text-muted-foreground">
                        {content.group} ·{" "}
                        {item.published ? t("Published") : t("Unpublished")}
                        {item.published && item.draft !== item.published
                          ? ` · ${t("Unpublished changes")}`
                          : ""}
                      </span>
                    </button>
                  );
                })}
              </nav>
            </aside>
            <DocumentEditor
              key={`${selectedId ?? "new"}-${article?.version ?? 0}`}
              article={article}
              catalog={query.data ?? []}
              onSaved={saved}
              onDirty={setDirty}
            />
          </div>
        )}
        <ConfirmDialog
          open={Boolean(nextSelection)}
          onOpenChange={(open) => {
            if (!open) setNextSelection(null);
          }}
          title={t("Unsaved changes")}
          desc={t("Discard unsaved changes and switch documents?")}
          handleConfirm={() => {
            if (nextSelection) {
              setSelectedId(nextSelection.id);
              setDirty(false);
              setNextSelection(null);
            }
          }}
        />
      </SectionPageLayout.Content>
    </SectionPageLayout>
  );
}

function DocumentEditor(props: {
  article?: AdminArticle;
  catalog: AdminArticle[];
  onSaved: (article: AdminArticle) => void;
  onDirty: (dirty: boolean) => void;
}) {
  const { t } = useTranslation();
  const initial = props.article
    ? {
        ...(JSON.parse(props.article.draft) as DocumentContent),
        slug: props.article.slug,
      }
    : empty;
  const form = useForm<Form>({
    resolver: zodResolver(schema),
    defaultValues: { ...initial, body: initial.body ?? "" },
  });
  const values = useWatch({ control: form.control });
  const [preview, setPreview] = useState(false);
  const [previewed, setPreviewed] = useState<string | null>(null);
  const [action, setAction] = useState<Action | null>(null);
  const [uploading, setUploading] = useState(false);
  const dirty = form.formState.isDirty;
  const reportDirty = props.onDirty;
  useEffect(() => {
    reportDirty(dirty);
  }, [dirty, reportDirty]);
  useEffect(() => {
    if (!dirty) return;
    const warn = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [dirty]);
  const mutation = useMutation({
    mutationFn: async (operation: "save" | Action) => {
      if (operation === "save") {
        const content = form.getValues();
        if (!props.article) return createDocument(content.slug, content);
        return changeDocument(props.article, "save", content);
      }
      if (!props.article) throw new Error(t("Save the draft first"));
      return changeDocument(props.article, operation);
    },
    onSuccess: (updated) => {
      toast.success(t("Operation completed"));
      setAction(null);
      props.onSaved(updated);
    },
    onError: (error) => toast.error(t(error.message)),
  });
  const previewArticle: PublicArticle = {
    title: values.title || t("Untitled document"),
    group: values.group || t("Documents"),
    group_order: values.group_order ?? 0,
    sort_order: values.sort_order ?? 0,
    body: values.body ?? "",
    slug: values.slug || "preview",
    published_at: 0,
  };
  const previewCatalog: PublicArticle[] = props.catalog
    .filter((item) => item.id !== props.article?.id && item.published)
    .map((item) => ({
      ...(JSON.parse(item.published) as DocumentContent),
      slug: item.slug,
      published_at: item.published_at,
    }));
  previewCatalog.push(previewArticle);
  const errors = Object.keys(form.formState.errors);
  const disabled = mutation.isPending || uploading;
  let actionTitle = t("Restore previous publication");
  if (action === "publish") actionTitle = t("Publish draft");
  else if (action === "withdraw") actionTitle = t("Withdraw document");
  return (
    <section className="min-w-0 rounded-xl border p-4 sm:p-6">
      <form
        onSubmit={form.handleSubmit(() => mutation.mutate("save"))}
        className="space-y-5"
      >
        <fieldset disabled={disabled} className="grid gap-4 sm:grid-cols-2">
          <label className="space-y-2 text-sm">
            <span>{t("Document title")}</span>
            <Input {...form.register("title")} maxLength={160} />
          </label>
          <label className="space-y-2 text-sm">
            <span>{t("Document path")}</span>
            <Input
              {...form.register("slug")}
              readOnly={Boolean(props.article)}
              placeholder="quick-start"
            />
            <span className="block text-xs text-muted-foreground">
              {t(
                "Lowercase letters, numbers and hyphens. The path is fixed after creation.",
              )}
            </span>
          </label>
          <label className="space-y-2 text-sm">
            <span>{t("Document group")}</span>
            <Input
              {...form.register("group")}
              list="documentation-groups"
              maxLength={80}
            />
          </label>
          <datalist id="documentation-groups">
            {[
              ...new Set(
                props.catalog.map(
                  (item) => (JSON.parse(item.draft) as DocumentContent).group,
                ),
              ),
            ].map((group) => (
              <option key={group} value={group} />
            ))}
          </datalist>
          <div className="grid grid-cols-2 gap-3">
            <label className="space-y-2 text-sm">
              <span>{t("Group order")}</span>
              <Input
                type="number"
                {...form.register("group_order", { valueAsNumber: true })}
              />
            </label>
            <label className="space-y-2 text-sm">
              <span>{t("Article order")}</span>
              <Input
                type="number"
                {...form.register("sort_order", { valueAsNumber: true })}
              />
            </label>
          </div>
        </fieldset>
        <label className="block space-y-2 text-sm">
          <span>{t("Markdown content")}</span>
          <Textarea
            {...form.register("body")}
            disabled={disabled}
            className="min-h-[400px] resize-y font-mono text-sm leading-6"
            spellCheck={false}
          />
        </label>
        {errors.length > 0 && (
          <p role="alert" className="text-sm text-destructive">
            {t("Check the title, path, group, ordering and content length.")}
          </p>
        )}
        <p className="text-xs text-muted-foreground">
          {t(
            "Markdown supports headings, code blocks, tables, images and blockquotes. Images uploaded here are public.",
          )}
        </p>
        <div className="flex flex-wrap items-center gap-3">
          <Button type="submit" disabled={disabled}>
            {mutation.isPending ? t("Saving...") : t("Save draft")}
          </Button>
          <Button
            type="button"
            variant="outline"
            disabled={disabled}
            onClick={() => {
              setPreview(true);
              setPreviewed(JSON.stringify(form.getValues()));
            }}
          >
            {t("Preview document")}
          </Button>
          <label className="cursor-pointer text-sm text-primary">
            {uploading ? t("Uploading...") : t("Upload image")}
            <input
              type="file"
              aria-label={t("Upload image")}
              accept="image/png,image/jpeg,image/webp"
              disabled={disabled}
              className="sr-only"
              onChange={async (event) => {
                const file = event.target.files?.[0];
                event.target.value = "";
                if (!file) return;
                if (file.size > 5 * 1024 * 1024) {
                  toast.error(t("Image must be under 5 MB"));
                  return;
                }
                setUploading(true);
                try {
                  const result = await uploadDocumentImage(file);
                  form.setValue(
                    "body",
                    `${form.getValues("body")}\n\n![${t("Image")}](${result.url})\n`,
                    { shouldDirty: true },
                  );
                } catch (error) {
                  toast.error(
                    error instanceof Error
                      ? t(error.message)
                      : t("Upload failed"),
                  );
                } finally {
                  setUploading(false);
                }
              }}
            />
          </label>
        </div>
        <div className="flex flex-wrap items-center gap-3 border-t pt-4">
          <Button
            type="button"
            disabled={
              disabled ||
              !props.article ||
              dirty ||
              !values.body?.trim() ||
              previewed !== JSON.stringify(form.getValues())
            }
            onClick={() => setAction("publish")}
          >
            {t("Publish draft")}
          </Button>
          <Button
            type="button"
            variant="outline"
            disabled={disabled || dirty || !props.article?.published}
            onClick={() => setAction("withdraw")}
          >
            {t("Withdraw document")}
          </Button>
          <Button
            type="button"
            variant="outline"
            disabled={disabled || dirty || !props.article?.previous}
            onClick={() => setAction("restore")}
          >
            {t("Restore previous publication")}
          </Button>
          <a
            className="text-sm text-primary underline"
            href={`/docs?article=${encodeURIComponent(props.article?.slug ?? "")}`}
            target="_blank"
            rel="noreferrer"
          >
            {t("Open public docs")}
          </a>
        </div>
        <p className="text-xs text-muted-foreground">
          {t(
            "Save and preview the current draft before publishing. Restoring a publication does not overwrite your draft.",
          )}
        </p>
        {props.article && (
          <p className="text-xs text-muted-foreground">
            {t("Last edited by")}: #{props.article.updated_by} ·{" "}
            {new Date(props.article.updated_at * 1000).toLocaleString()}
            {props.article.published_at > 0
              ? ` · ${t("Published at")}: ${new Date(props.article.published_at * 1000).toLocaleString()}`
              : ""}
          </p>
        )}
      </form>
      <Dialog open={preview} onOpenChange={setPreview}>
        <DialogContent className="max-h-[95vh] overflow-auto sm:max-w-[95vw]">
          <DialogTitle>{t("Draft preview")}</DialogTitle>
          <DialogDescription>
            {t("This preview is not public until you publish the saved draft.")}
          </DialogDescription>
          <DocumentationView
            article={previewArticle}
            catalog={previewCatalog}
            preview
          />
        </DialogContent>
      </Dialog>
      <ConfirmDialog
        open={Boolean(action)}
        onOpenChange={(open) => {
          if (!open && !mutation.isPending) setAction(null);
        }}
        title={actionTitle}
        desc={t(
          "This action changes the public documentation immediately. Continue?",
        )}
        isLoading={mutation.isPending}
        handleConfirm={() => {
          if (action) mutation.mutate(action);
        }}
      />
    </section>
  );
}
