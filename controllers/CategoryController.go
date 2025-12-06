package controllers

import (
	"net/http"
	"portProject_development/db"
	"portProject_development/models"

	"github.com/gin-gonic/gin"
)

func GetCategories(c *gin.Context) {
	var categories []models.Category
	if err := db.DB.Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kategoriler alınamadı"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": categories})
}
