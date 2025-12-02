package admin

import (
	"errors"
	"net/http"
	"portProject_development/db" // Kendi modül ismin
	"portProject_development/enums"
	"portProject_development/models"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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

func Promotion(c *gin.Context) {
	productId := c.Param("id")
	productIdInt, err := strconv.Atoi(productId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz ID"})
		return
	}

	var input struct {
		PromotionType string  `json:"promotion_type" binding:"required"`
		DiscountRate  float64 `json:"discount_rate"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Ürün kontrolü
	var product models.Product
	if err := db.DB.First(&product, productId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ürün bulunamadı"})
		return
	}

	targetType := enums.PromotionType(input.PromotionType)
	var promotion models.Promotion

	// Bu ürün için ve bu tipte (Örn: Discount) zaten açık bir kampanya başlığı var mı?
	errProm := db.DB.Where("product_id = ? AND promotion_type = ?", product.ID, targetType).First(&promotion).Error

	if errProm == nil {

	} else if errors.Is(errProm, gorm.ErrRecordNotFound) {
		// DURUM 2: Kayıt yok, o zaman yeni oluşturuyoruz.
		promotion = models.Promotion{
			ProductID:     product.ID,
			PromotionType: targetType,
		}
		if err := db.DB.Create(&promotion).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Promosyon başlığı oluşturulamadı"})
			return
		}
	} else {
		// DURUM 3: Veritabanı hatası
		c.JSON(http.StatusInternalServerError, gin.H{"error": errProm.Error()})
		return
	}

	if targetType == enums.PromotionTypeDiscount {
		DiscountPromotion(c, input.DiscountRate, productIdInt, promotion.ID)
		return
	}

	if targetType == enums.PromotionTypeShip {
		ShipPromotion(c) // Gerekirse buna da ID gönderin
		return
	}
}

func DiscountPromotion(c *gin.Context, discountRate float64, productId int, promotionId uint) {

	//promosyon yüzdesi ver

	if discountRate <= 0 || discountRate > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lütfen indirim değerlerini kontrol ediniz..."})
		return
	}

	var product models.Product
	//ürünü getir
	if err := db.DB.First(&product, productId).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	var discountPromos []models.Discount

	if err := db.DB.
		Joins("JOIN promotions ON promotions.id = discounts.promotion_id").
		Where("discounts.product_id = ? AND promotions.promotion_type = ?", productId, enums.PromotionTypeDiscount).
		Find(&discountPromos).Error; err == nil {
		if len(discountPromos) > 0 {
			db.DB.Where("product_id = ?", productId).Delete(&models.Discount{})

		}
	}
	//promosyonlu fiyatı hesapla
	var discountPrice = product.Price - (((discountRate) / 100) * product.Price)

	discount := models.Discount{
		ProductID:     product.ID, // ⚠ FK hatasını engellemek için zorunlu
		PromotionID:   promotionId,
		DiscountRate:  discountRate,
		DiscountPrice: discountPrice,
	}

	if err := db.DB.Create(&discount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
}

func ShipPromotion(c *gin.Context) {

}
