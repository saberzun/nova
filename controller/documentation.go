package controller

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetDocumentationCatalog(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	articles, err := model.ListPublishedDocumentation()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, articles)
}
func GetDocumentationArticle(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	a, err := model.GetPublishedDocumentation(c.Param("slug"))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(404, gin.H{"success": false, "message": "Document not found"})
		return
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, a)
}
func AdminListDocumentation(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	a, err := model.ListAdminDocumentation()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, a)
}
func AdminCreateDocumentation(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 256*1024)
	var request struct {
		Slug    string                     `json:"slug"`
		Content model.DocumentationContent `json:"content"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorMsg(c, "Invalid document request")
		return
	}
	a, err := model.CreateDocumentationArticle(c.GetInt("id"), request.Slug, request.Content)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, a)
}
func AdminChangeDocumentation(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 256*1024)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "Invalid document ID")
		return
	}
	var request struct {
		Version int64                      `json:"version"`
		Content model.DocumentationContent `json:"content"`
	}
	if err = c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorMsg(c, "Invalid document request")
		return
	}
	a, err := model.ChangeDocumentationArticle(id, c.GetInt("id"), request.Version, c.Param("action"), request.Content)
	if errors.Is(err, model.ErrDocumentationConflict) {
		c.JSON(409, gin.H{"success": false, "message": "Document changed; reload before saving"})
		return
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, a)
}

var documentationImageKey = regexp.MustCompile(`^[a-zA-Z0-9]{32}\.(png|jpg|webp)$`)

func AdminUploadDocumentationImage(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 6*1024*1024)
	file, err := c.FormFile("file")
	if err != nil {
		common.ApiErrorMsg(c, "Select an image under 5 MB")
		return
	}
	image, err := saveCorporateImage(file, "documentation")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	// Documentation images are public assets; use a dedicated namespace, never
	// expose the private payment-evidence or corporate-collection namespaces.
	name := image.StorageKey[len("documentation/"):]
	common.ApiSuccess(c, gin.H{"url": "/api/docs/images/" + name})
}
func GetDocumentationImage(c *gin.Context) {
	name := c.Param("name")
	if !documentationImageKey.MatchString(name) {
		c.Status(404)
		return
	}
	mime := "image/png"
	if name[len(name)-4:] == ".jpg" {
		mime = "image/jpeg"
	}
	if name[len(name)-5:] == ".webp" {
		mime = "image/webp"
	}
	serveCorporateImage(c, "documentation/"+name, mime, name)
}
