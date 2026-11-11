package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSKUDeletionListingAndRestorePreserveHistoricalOrders(t *testing.T) {
	db := setupManageUserTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Product{}, &model.ProductSKU{}, &model.ProductOrder{}, &model.ProductOrderItem{}, &model.Entitlement{}, &model.EntitlementType{}, &model.EntitlementTypeGroup{}))
	product := model.Product{Code: "history", Name: "History", Category: model.ProductCategorySubscription, Status: model.ProductStatusActive}
	require.NoError(t, db.Create(&product).Error)
	sku := model.ProductSKU{Code: "history", Name: "History", ProductId: product.Id, Status: model.ProductStatusActive}
	require.NoError(t, db.Create(&sku).Error)
	order := model.ProductOrder{OrderNo: "history", Status: model.ProductOrderStatusCancelled}
	require.NoError(t, db.Create(&order).Error)
	require.NoError(t, db.Create(&model.ProductOrderItem{OrderId: order.Id, SKUId: sku.Id}).Error)
	router := gin.New()
	router.GET("/products", AdminListProducts)
	router.GET("/store", ListStoreProducts)
	router.GET("/skus/:id/references", AdminGetSKUReferences)
	router.DELETE("/skus/:id", AdminDeleteProductSKU)
	router.POST("/skus/:id/restore", AdminRestoreProductSKU)
	router.PUT("/skus/:id", AdminUpdateProductSKU)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodDelete, "/skus/1", nil))
	var deletion struct {
		Success bool `json:"success"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &deletion))
	require.True(t, deletion.Success, response.Body.String())

	for _, test := range []struct {
		path  string
		count int
	}{
		{"/products", 0}, {"/products?include_deleted=true", 1}, {"/store", 0},
	} {
		t.Run(test.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
			var result struct {
				Success bool            `json:"success"`
				Data    []model.Product `json:"data"`
			}
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
			require.True(t, result.Success)
			require.Len(t, result.Data, 1)
			assert.Len(t, result.Data[0].SKUs, test.count)
		})
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/skus/1", strings.NewReader(`{"status":"active"}`)))
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &deletion))
	assert.False(t, deletion.Success)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/skus/1/restore", nil))
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &deletion))
	assert.True(t, deletion.Success, response.Body.String())
	require.NoError(t, db.First(&sku, sku.Id).Error)
	assert.Zero(t, sku.DeletedAt)
	assert.Equal(t, model.ProductStatusArchived, sku.Status)
	require.NoError(t, db.First(&order, order.Id).Error)
	assert.Equal(t, model.ProductOrderStatusCancelled, order.Status)
}
