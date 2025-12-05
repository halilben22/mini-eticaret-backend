// POST /admin/promotions
package admin

import (
	"net/http"
	"portProject_development/db"
	"portProject_development/models"
	"time"

	"github.com/gin-gonic/gin"
)

func CreatePromotion(c *gin.Context) {
	var input models.Promotion
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

// GET PROMOTIONS
func GetPromotions(c *gin.Context) {
	var promotions []models.Promotion

	// En yeniden en eskiye doğru sırala
	if err := db.DB.Order("created_at desc").Find(&promotions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Promosyonlar getirilemedi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": promotions})
}

// DELETE PROMOTION(DELETE /admin/promotions/:id)
func DeletePromotion(c *gin.Context) {
	id := c.Param("id")

	// DELETE BY "HARD DELETE" (DELETING PERMANANTLY)
	if err := db.DB.Delete(&models.Promotion{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Promosyon silinemedi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Promosyon başarıyla silindi"})
}
