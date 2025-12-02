package admin

import (
	"net/http"
	"portProject_development/db" // Kendi modül ismin
	"portProject_development/models"

	"github.com/gin-gonic/gin"
)

func GetDashboardStats(c *gin.Context) {

	database := db.DB

	var totalOrders int64
	var totalRevenue float64

	//Toplam sipariş sayısı
	database.Model(&models.Order{}).Count(&totalOrders)

	database.Model(&models.Order{}).
		Where("status IN ?", []string{"paid", "shipped", "delivered"}).
		Select("COALESCE(SUM(total_amount), 0)").
		Scan(&totalRevenue)

	c.JSON(http.StatusOK, gin.H{
		"total_orders":  totalOrders,
		"total_revenue": totalRevenue,
	})
}

func GetAllOrders(c *gin.Context) {
	var orders []models.Order
	if err := db.DB.Preload("OrderItems.Product").Order("created_at desc").Find(&orders).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": orders})

}

func UpdateOrderStatus(c *gin.Context) {
	id := c.Param("id")
	var input struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Durum bilgisi gerekli"})
		return
	}

	var order models.Order
	if err := db.DB.Where("id = ?", id).First(&order).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sipariş bulunamadı" + err.Error()})
		return
	}

	if err := db.DB.Model(&order).Where("id = ?", id).Update("status", input.Status).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Durum bilgisi güncellenemedi" + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "Sipariş durumu güncellendi", "status ": input.Status})

}

func ProductDiscount(c *gin.Context) {
	productId := c.Param("id")
	//promosyon yüzdesi ver
	var input struct {
		DiscountRate float64 `json:"discount_rate" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil || input.DiscountRate <= 0 || input.DiscountRate > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lütfen indirim değerlerini kontrol ediniz..." + err.Error()})
		return
	}

	var product models.Product
	//ürünü getir
	if err := db.DB.First(&product, productId).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	var promotions []models.Promotion

	if err := db.DB.Where("product_id = ?", productId).Find(&promotions).Error; err == nil {
		if len(promotions) > 0 {
			db.DB.Where("product_id = ?", productId).Delete(&models.Promotion{})
		}
	}
	//promosyonlu fiyatı hesapla
	var discountPrice = product.Price - (((input.DiscountRate) / 100) * product.Price)

	promotion := models.Promotion{
		ProductID:     product.ID,
		Discount:      input.DiscountRate,
		DiscountPrice: discountPrice,
	}
	//promosyon 100 geçmesin
	//bir üründe sadece 1 promosyon olsun

	db.DB.Save(&promotion)

}
