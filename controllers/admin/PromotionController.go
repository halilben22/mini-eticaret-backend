package admin

import (
	"errors"
	"net/http"
	"portProject_development/db"
	"portProject_development/enums"
	"portProject_development/models"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Promotion(c *gin.Context) {
	productId := c.Param("id")
	productIdInt, err := strconv.Atoi(productId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz ID"})
		return
	}

	var input struct {
		PromotionType  string  `json:"promotion_type" binding:"required"`
		DiscountRate   float64 `json:"discount_rate"`
		ShipPriceLimit float64 `json:"ship_price_limit"`
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
		//  eğer kayıt yoksa yenisini oluştur
		promotion = models.Promotion{
			ProductID:     product.ID,
			PromotionType: targetType,
		}
		if err := db.DB.Create(&promotion).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Promosyon başlığı oluşturulamadı"})
			return
		}
	} else {
		// Veritabanı hatası
		c.JSON(http.StatusInternalServerError, gin.H{"error": errProm.Error()})
		return
	}

	if targetType == enums.PromotionTypeDiscount {
		DiscountPromotion(c, input.DiscountRate, productIdInt, promotion.ID)
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

func ShipPromotion(c *gin.Context, shipPriceLimit float64, productId int, promotionId uint) {

}
