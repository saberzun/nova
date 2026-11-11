# Lightweight documentation — first release

## Operator entry points

- Public: `/docs?article=quick-start` (the default article is selected when no slug is supplied).
- Admin: `/documentation`, also available in the administrator navigation.
- Operators currently need the existing administrator role. This release does not introduce a separate limited editor role.

## Editing and publication

Create an article with an immutable lowercase slug; set title, group, group order and article order. Groups are derived from article metadata, and lower sort values come first. Articles in the same group should use the same group order.

Edit Markdown and save the draft. Preview the saved draft, then publish. Public navigation and content come exclusively from published snapshots. Unsaved changes must be saved before publication actions. A version check rejects stale concurrent edits rather than overwriting them. The UI also warns before switching away from unsaved changes.

Withdraw removes an article from public navigation and makes its public API unavailable. Restore swaps the published snapshot with the previous one; it does not overwrite the draft. Only one previous published snapshot is retained.

## Rendering

The preview and public article share `DocumentationView`. Rendering reuses the existing sanitized GFM Markdown component. Enhancements include heading anchors, desktop table of contents, collapsible mobile navigation, asynchronous syntax highlighting with plain-text fallback, code copy buttons, responsive tables and image enlargement. Raw scripts are not executed.

Image uploads support JPEG, PNG and WebP up to 5 MB. Uploaded images are public assets in a dedicated documentation namespace; never upload private payment evidence or secrets. Existing private corporate-transfer assets are not exposed by the documentation route.

## Persistence and deployment

`documentation_articles` stores draft, current publication and previous publication as text snapshots plus optimistic version and operator timestamps. Public list responses omit Markdown bodies and never return drafts. Article bodies are fetched separately. Public responses use `Cache-Control: no-store` so newly published content is visible on the next request without a build.

The table is additive and supports SQLite, MySQL and PostgreSQL through GORM. The three default Chinese articles are initialized only when the entire table is empty. No data is migrated out of the existing legal documents module.

Images use the existing persisted corporate-transfer storage root, under `documentation/`. Back up that storage with the database.

## Verification

- Model tests cover draft isolation, publication, stale writes, withdrawal, restoration and restart-safe seeding.
- HTTP tests cover unauthenticated admin access and the public publication boundary.
- Frontend tests cover grouped navigation, stable anchors, title escaping, mobile navigation and shared preview rendering.

## Boundaries

No full-text search, approval workflow, scheduled publication, collaborative editing, full version history or content localization in this first release. The UI is translated; operators author article content directly.
