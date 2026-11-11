package model

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"testing"
)

func resetDocumentation(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&DocumentationArticle{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&DocumentationArticle{}).Error)
	t.Cleanup(func() { DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&DocumentationArticle{}) })
}

func TestDocumentationDraftPublishRestoreAndWithdraw(t *testing.T) {
	resetDocumentation(t)
	content := DocumentationContent{Title: "快速开始", Group: "入门", Body: "## 认证\n使用 API Key"}
	article, err := CreateDocumentationArticle(1, "quick-start", content)
	require.NoError(t, err)
	list, err := ListPublishedDocumentation()
	require.NoError(t, err)
	assert.Empty(t, list)
	_, err = GetPublishedDocumentation("quick-start")
	assert.Error(t, err)
	article, err = ChangeDocumentationArticle(article.Id, 1, article.Version, "publish", DocumentationContent{})
	require.NoError(t, err)
	public, err := GetPublishedDocumentation("quick-start")
	require.NoError(t, err)
	assert.Equal(t, content.Body, public.Body)
	content.Body = "草稿内容"
	article, err = ChangeDocumentationArticle(article.Id, 2, article.Version, "save", content)
	require.NoError(t, err)
	public, err = GetPublishedDocumentation("quick-start")
	require.NoError(t, err)
	assert.NotEqual(t, content.Body, public.Body)
	article, err = ChangeDocumentationArticle(article.Id, 2, article.Version, "publish", DocumentationContent{})
	require.NoError(t, err)
	assert.NotEmpty(t, article.Previous)
	article, err = ChangeDocumentationArticle(article.Id, 2, article.Version, "restore", DocumentationContent{})
	require.NoError(t, err)
	public, err = GetPublishedDocumentation("quick-start")
	require.NoError(t, err)
	assert.Equal(t, "## 认证\n使用 API Key", public.Body)
	_, err = ChangeDocumentationArticle(article.Id, 2, article.Version, "withdraw", DocumentationContent{})
	require.NoError(t, err)
	_, err = GetPublishedDocumentation("quick-start")
	assert.Error(t, err)
}

func TestDocumentationRejectsStaleWriteAndInvalidContent(t *testing.T) {
	resetDocumentation(t)
	content := DocumentationContent{Title: "Title", Group: "Group", Body: "Body"}
	a, err := CreateDocumentationArticle(1, "valid-slug", content)
	require.NoError(t, err)
	_, err = CreateDocumentationArticle(1, "../bad", content)
	assert.Error(t, err)
	_, err = CreateDocumentationArticle(1, "valid-slug", content)
	assert.Error(t, err)
	_, err = ChangeDocumentationArticle(a.Id, 1, a.Version-1, "save", content)
	assert.ErrorIs(t, err, ErrDocumentationConflict)
	content.Title = ""
	_, err = ChangeDocumentationArticle(a.Id, 1, a.Version, "save", content)
	assert.Error(t, err)
	_, err = ChangeDocumentationArticle(a.Id, 1, a.Version, "restore", DocumentationContent{})
	assert.Error(t, err)
}

func TestDocumentationPublicCatalogDoesNotExposeDrafts(t *testing.T) {
	resetDocumentation(t)
	c := DocumentationContent{Title: "Published", Group: "Guide", Body: "public"}
	a, err := CreateDocumentationArticle(1, "guide", c)
	require.NoError(t, err)
	a, err = ChangeDocumentationArticle(a.Id, 1, a.Version, "publish", DocumentationContent{})
	require.NoError(t, err)
	c.Title = "Secret draft"
	c.Body = "unpublished secret"
	_, err = ChangeDocumentationArticle(a.Id, 1, a.Version, "save", c)
	require.NoError(t, err)
	list, err := ListPublishedDocumentation()
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Published", list[0].Title)
	assert.Empty(t, list[0].Body)
}

func TestDocumentationDefaultsDoNotOverwriteOperatorChanges(t *testing.T) {
	resetDocumentation(t)
	require.NoError(t, EnsureDocumentationDefaults())
	list, err := ListAdminDocumentation()
	require.NoError(t, err)
	require.Len(t, list, 3)
	a := list[0]
	_, err = ChangeDocumentationArticle(a.Id, 1, a.Version, "withdraw", DocumentationContent{})
	require.NoError(t, err)
	require.NoError(t, EnsureDocumentationDefaults())
	_, err = GetPublishedDocumentation(a.Slug)
	assert.Error(t, err)
}
