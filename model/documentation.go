package model

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

var ErrDocumentationConflict = errors.New("document changed, reload before saving")
var documentationSlug = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Published and previous snapshots contain all display metadata, so draft edits
// never change public navigation before an explicit publication.
type DocumentationContent struct {
	Title      string `json:"title"`
	Group      string `json:"group"`
	GroupOrder int    `json:"group_order"`
	SortOrder  int    `json:"sort_order"`
	Body       string `json:"body,omitempty"`
}
type DocumentationArticle struct {
	Id          int    `json:"id" gorm:"primaryKey"`
	Slug        string `json:"slug" gorm:"size:160;uniqueIndex;not null"`
	Draft       string `json:"draft" gorm:"size:1048576;not null"`
	Published   string `json:"published" gorm:"size:1048576;not null"`
	Previous    string `json:"previous" gorm:"size:1048576;not null"`
	Version     int64  `json:"version" gorm:"bigint;not null"`
	UpdatedBy   int    `json:"updated_by"`
	PublishedBy int    `json:"published_by"`
	UpdatedAt   int64  `json:"updated_at" gorm:"bigint"`
	PublishedAt int64  `json:"published_at" gorm:"bigint"`
}
type DocumentationPublic struct {
	DocumentationContent
	Slug        string `json:"slug"`
	PublishedAt int64  `json:"published_at"`
}

func validateDocumentationContent(c DocumentationContent) error {
	if strings.TrimSpace(c.Title) == "" || utf8.RuneCountInString(c.Title) > 160 || strings.TrimSpace(c.Group) == "" || utf8.RuneCountInString(c.Group) > 80 {
		return errors.New("title and group are required (maximum 160 and 80 characters)")
	}
	if len(c.Body) > 200000 || c.SortOrder < -100000 || c.SortOrder > 100000 || c.GroupOrder < -100000 || c.GroupOrder > 100000 {
		return errors.New("document content or ordering exceeds allowed limits")
	}
	return nil
}
func CreateDocumentationArticle(operator int, slug string, content DocumentationContent) (*DocumentationArticle, error) {
	if !documentationSlug.MatchString(slug) || len(slug) > 160 {
		return nil, errors.New("use lowercase letters, numbers and hyphens for the document path")
	}
	if err := validateDocumentationContent(content); err != nil {
		return nil, err
	}
	raw, err := common.Marshal(content)
	if err != nil {
		return nil, err
	}
	a := DocumentationArticle{Slug: slug, Draft: string(raw), Version: 1, UpdatedBy: operator, UpdatedAt: time.Now().Unix()}
	if err = DB.Create(&a).Error; err != nil {
		return nil, errors.New("document path already exists or document could not be created")
	}
	return &a, nil
}
func ListAdminDocumentation() ([]DocumentationArticle, error) {
	a := []DocumentationArticle{}
	err := DB.Order("id asc").Find(&a).Error
	return a, err
}
func ListPublishedDocumentation() ([]DocumentationPublic, error) {
	var articles []DocumentationArticle
	if err := DB.Select("slug", "published", "published_at").Where("published <> ?", "").Find(&articles).Error; err != nil {
		return nil, err
	}
	result := make([]DocumentationPublic, 0, len(articles))
	for _, a := range articles {
		var content DocumentationContent
		if err := common.UnmarshalJsonStr(a.Published, &content); err != nil {
			return nil, err
		}
		content.Body = ""
		result = append(result, DocumentationPublic{DocumentationContent: content, Slug: a.Slug, PublishedAt: a.PublishedAt})
	}
	return result, nil
}
func GetPublishedDocumentation(slug string) (*DocumentationPublic, error) {
	var a DocumentationArticle
	if err := DB.Select("slug", "published", "published_at").Where("slug = ? AND published <> ?", slug, "").First(&a).Error; err != nil {
		return nil, err
	}
	var content DocumentationContent
	if err := common.UnmarshalJsonStr(a.Published, &content); err != nil {
		return nil, err
	}
	return &DocumentationPublic{DocumentationContent: content, Slug: a.Slug, PublishedAt: a.PublishedAt}, nil
}
func ChangeDocumentationArticle(id, operator int, version int64, action string, content DocumentationContent) (*DocumentationArticle, error) {
	var a DocumentationArticle
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).First(&a, id).Error; err != nil {
			return err
		}
		if a.Version != version {
			return ErrDocumentationConflict
		}
		changes := map[string]interface{}{"version": version + 1, "updated_by": operator, "updated_at": time.Now().Unix()}
		switch action {
		case "save":
			if err := validateDocumentationContent(content); err != nil {
				return err
			}
			raw, err := common.Marshal(content)
			if err != nil {
				return err
			}
			changes["draft"] = string(raw)
		case "publish":
			var draft DocumentationContent
			if err := common.UnmarshalJsonStr(a.Draft, &draft); err != nil {
				return err
			}
			if err := validateDocumentationContent(draft); err != nil {
				return err
			}
			if strings.TrimSpace(draft.Body) == "" {
				return errors.New("document body is required for publication")
			}
			if a.Published != "" && a.Published != a.Draft {
				changes["previous"] = a.Published
			}
			changes["published"] = a.Draft
			changes["published_at"] = time.Now().Unix()
			changes["published_by"] = operator
		case "withdraw":
			if a.Published == "" {
				return errors.New("document is not published")
			}
			changes["previous"] = a.Published
			changes["published"] = ""
			changes["published_at"] = int64(0)
		case "restore":
			if a.Previous == "" {
				return errors.New("no previous published version is available")
			}
			changes["published"] = a.Previous
			changes["previous"] = a.Published
			changes["published_at"] = time.Now().Unix()
			changes["published_by"] = operator
		default:
			return errors.New("unsupported document action")
		}
		result := tx.Model(&DocumentationArticle{}).Where("id = ? AND version = ?", id, version).Updates(changes)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrDocumentationConflict
		}
		return tx.First(&a, id).Error
	})
	return &a, err
}

// Only seeds an empty installation. Unpublished articles are never re-published on restart.
func EnsureDocumentationDefaults() error {
	var count int64
	if err := DB.Model(&DocumentationArticle{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	defaults := []struct {
		slug    string
		content DocumentationContent
	}{
		{"quick-start", DocumentationContent{Title: "快速开始", Group: "接入指南", Body: "## 创建 API 密钥\n\n在控制台创建 API 密钥，并选择允许使用的分组。请妥善保管密钥，不要在公开页面或代码仓库中暴露它。\n\n## 发送第一个请求\n\n```bash\ncurl https://api.itokenify.com/v1/chat/completions \\\n  -H \"Authorization: Bearer YOUR_API_KEY\" \\\n  -H \"Content-Type: application/json\" \\\n  -d '{\"model\":\"YOUR_MODEL\",\"messages\":[{\"role\":\"user\",\"content\":\"Hello\"}]}'\n```\n\n> 请将 YOUR_MODEL 替换为当前分组实际可用的模型。\n\n## 查询模型列表\n\n```bash\ncurl https://api.itokenify.com/v1/models \\\n  -H \"Authorization: Bearer YOUR_API_KEY\"\n```"}},
		{"quota-and-groups", DocumentationContent{Title: "额度与分组", Group: "使用说明", GroupOrder: 1, Body: "## 两种资金来源\n\n充值额度与订阅额度分别管理，每种额度只能用于其允许的分组。\n\n## 分组使用顺序\n\nAPI 密钥包含多个分组时，系统按配置顺序选择可用分组。不可用分组可能仍显示，但无法选择。\n\n## 查看消耗\n\n使用记录会展示实际使用的分组、资金来源以及计费详情。具体可用额度、有效期和每日限额以所购商品及账户权益为准。"}},
		{"client-setup", DocumentationContent{Title: "客户端配置", Group: "接入指南", SortOrder: 1, Body: "## OpenAI 兼容客户端\n\n将 Base URL 设置为：\n\n```text\nhttps://api.itokenify.com/v1\n```\n\n使用控制台创建的 API 密钥。可用模型取决于密钥分组和上游可用性。\n\n## 排查连接问题\n\n| 响应 | 检查方向 |\n| --- | --- |\n| 401 | 检查 API 密钥 |\n| 429 | 降低请求频率，稍后重试 |\n| 模型不可用 | 核对模型名称及分组权限 |"}},
	}
	for _, d := range defaults {
		raw, err := common.Marshal(d.content)
		if err != nil {
			return err
		}
		a := DocumentationArticle{Slug: d.slug, Draft: string(raw), Published: string(raw), Version: 1, UpdatedAt: time.Now().Unix(), PublishedAt: time.Now().Unix()}
		// Unique slug protects simultaneous initializers on different application nodes.
		if err := DB.Where("slug = ?", d.slug).FirstOrCreate(&a).Error; err != nil {
			return err
		}
	}
	return nil
}
