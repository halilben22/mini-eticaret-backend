package controllers

import (
	"net/http"
	"portProject_development/db" // Kendi modül ismin
	"portProject_development/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

func AddReview(c *gin.Context) {
	//giriş yapan kullanıcı
	userCtx, _ := c.Get("user")
	userID := userCtx.(models.User).ID

	//ürünü bul
	productID, _ := strconv.Atoi(c.Param("id"))

	var input struct {
		Rating  int    `json:"rating" binding:"required"`
		Comment string `json:"comment" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Puan (1-5) ve yorum zorunludur."})
		return
	}

	review := models.Reviews{
		UserID:    userID,
		ProductID: uint(productID),
		Rating:    input.Rating,
		Comment:   input.Comment,
	}

	if err := db.DB.Create(&review).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Yorum eklenemedi"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Yorum eklendi!", "data": review})

}

func GetProductReview(c *gin.Context) {
	productID := c.Param("id")
	var reviews []models.Reviews
	if err := db.DB.Preload("User").Where("product_id = ?", productID).Order("created_at desc").Find(&reviews).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Yorumlar getirilemedi"})
		return
	}

	// Ortalama Puanı hesaplamak
	var totalRating float64 = 0
	for _, r := range reviews {
		totalRating += float64(r.Rating)
	}
	average := 0.0
	if len(reviews) > 0 {
		average = totalRating / float64(len(reviews))
	}

	c.JSON(http.StatusOK, gin.H{
		"reviews":        reviews,
		"average_rating": average,
		"total_reviews":  len(reviews),
	})
}
