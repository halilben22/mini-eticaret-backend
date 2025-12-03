package admin

import (
	"net/http"
	"portProject_development/db" // Kendi modül ismin
	"portProject_development/models"
	"time"

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

// POST /admin/promotions
func CreatePromotion(c *gin.Context) {
	var input models.ShipPromotion
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Varsayılan bitiş tarihi yoksa 1 yıl sonrasını ver
	if input.ExpiresAt.IsZero() {
		input.ExpiresAt = time.Now().AddDate(1, 0, 0)
	}

	if err := db.DB.Create(&input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Promosyon oluşturulamadı"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": input})
}

// 4. PROMOSYONLARI LİSTELE (GET /admin/promotions)
func GetPromotions(c *gin.Context) {
	var promotions []models.ShipPromotion

	// En yeniden en eskiye doğru sırala
	if err := db.DB.Order("created_at desc").Find(&promotions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Promosyonlar getirilemedi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": promotions})
}

// 5. PROMOSYON SİL (DELETE /admin/promotions/:id)
func DeletePromotion(c *gin.Context) {
	id := c.Param("id")

	// Veritabanından sil (Hard delete yapar, istersen Soft Delete için gorm.DeletedAt kullanmış olman gerekirdi)
	if err := db.DB.Delete(&models.Promotion{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Promosyon silinemedi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Promosyon başarıyla silindi"})
}
