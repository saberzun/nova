package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentationHTTPPublicationBoundary(t *testing.T) {
	db := setupManageUserTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.DocumentationArticle{}))
	router := gin.New()
	router.GET("/docs", GetDocumentationCatalog)
	router.GET("/docs/:slug", GetDocumentationArticle)
	admin := router.Group("/admin")
	admin.Use(middleware.AdminAuth())
	admin.GET("", AdminListDocumentation)
	admin.POST("", AdminCreateDocumentation)
	a, err := model.CreateDocumentationArticle(1, "guide", model.DocumentationContent{Title: "Draft title", Group: "Guide", Body: "PRIVATE DRAFT"})
	require.NoError(t, err)
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(method, "/admin", strings.NewReader(`{}`)))
		assert.Equal(t, 401, w.Code)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/docs/guide", nil))
	assert.Equal(t, 404, w.Code)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/docs", nil))
	assert.NotContains(t, w.Body.String(), "PRIVATE DRAFT")
	a, err = model.ChangeDocumentationArticle(a.Id, 1, a.Version, "publish", model.DocumentationContent{})
	require.NoError(t, err)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/docs/guide", nil))
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "PRIVATE DRAFT")
	var response struct {
		Success bool                      `json:"success"`
		Data    model.DocumentationPublic `json:"data"`
	}
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
	assert.True(t, response.Success)
	_, err = model.ChangeDocumentationArticle(a.Id, 1, a.Version, "withdraw", model.DocumentationContent{})
	require.NoError(t, err)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/docs/guide", nil))
	assert.Equal(t, 404, w.Code)
}
func TestDocumentationImageRouteRejectsForeignNamespaces(t *testing.T) {
	router := gin.New()
	router.GET("/images/:name", GetDocumentationImage)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/images/evidence.png", nil))
	assert.Equal(t, 404, w.Code)
}
