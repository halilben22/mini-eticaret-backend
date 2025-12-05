package admin

import (
	"fmt"
	"net/http"
	"path/filepath"
	"portProject_development/db"
	"portProject_development/models"
	"strconv"
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

func CreateProduct(c *gin.Context) {

	name := c.PostForm("name")
	description := c.PostForm("description")
	priceStr := c.PostForm("price")
	stockStr := c.PostForm("stock_quantity")
	categoryIDStr := c.PostForm("category_id")

	price, _ := strconv.ParseFloat(priceStr, 64)
	stock, _ := strconv.Atoi(stockStr)
	categoryID, _ := strconv.Atoi(categoryIDStr)

	//Dosya yükleme
	file, err := c.FormFile("image")
	var imagePath string

	if err == nil {
		filename := fmt.Sprintf("%d_%s", time.Now().Unix(), filepath.Base(file.Filename))
		uploadPath := "uploads/" + filename

		//Dosyayı sunucuya kaydet
		if err := c.SaveUploadedFile(file, uploadPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Dosya kaydedilemedi: " + err.Error()})

			return
		}

		// Veritabanına kaydedilecek yol
		// Yayına aldığımda domain eklerim
		imagePath = "/uploads/" + filename

	} else {
		// Dosya yüklenmediyse varsayılan bir resim koyabiliriz
		imagePath = ""
	}

	product := models.Product{
		Name:          name,
		Description:   description,
		Price:         price,
		StockQuantity: stock,
		CategoryID:    uint(categoryID),
		ImageURL:      imagePath,
		IsActive:      true,
	}

	if err := db.DB.Create(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ürün veritabanına eklenemedi"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Ürün ve resim başarıyla yüklendi",
		"data":    product,
	})

}

// PRODUCT UPDATE FUNC
func UpdateProduct(c *gin.Context) {
	id := c.Param("id")

	// ürünü getir
	var product models.Product
	if err := db.DB.First(&product, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ürün bulunamadı"})
		return
	}

	// GET FORM DATA
	name := c.PostForm("name")
	description := c.PostForm("description")
	priceStr := c.PostForm("price")
	stockStr := c.PostForm("stock_quantity")
	categoryIDStr := c.PostForm("category_id")

	// IF NOT EMPTY,THEN UPDATE
	if name != "" {
		product.Name = name
	}
	if description != "" {
		product.Description = description
	}

	if priceStr != "" {
		if p, err := strconv.ParseFloat(priceStr, 64); err == nil {
			product.Price = p
		}
	}
	if stockStr != "" {
		if s, err := strconv.Atoi(stockStr); err == nil {
			product.StockQuantity = s
		}
	}
	if categoryIDStr != "" {
		if cat, err := strconv.Atoi(categoryIDStr); err == nil {
			product.CategoryID = uint(cat)
		}
	}

	// IMAGE FILE UPDATE
	file, err := c.FormFile("image")
	if err == nil {
		// Yeni resim geldiyse eskisini silmek (os.Remove) iyi olur ama şimdilik üzerine yazalım
		filename := fmt.Sprintf("%d_%s", time.Now().Unix(), filepath.Base(file.Filename))
		uploadPath := "uploads/" + filename

		if err := c.SaveUploadedFile(file, uploadPath); err == nil {
			product.ImageURL = "/uploads/" + filename
		}
	}

	// Save
	if err := db.DB.Save(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Güncelleme başarısız"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Ürün güncellendi", "data": product})
}

// DELETING PRODUCT BY "SOFT DELETE"(NOT EXACTLY DELETING ON DB BUT SWITCHING IS_ACTIVE CASE)
func DeleteProduct(c *gin.Context) {
	id := c.Param("id")

	//IT'LL LOOKS LIKE DELETED ON FRONT-END BUT WE CAN KEEP RECORDS OF ORDER'S PAST DATA
	if err := db.DB.Model(&models.Product{}).Where("id = ?", id).Update("is_active", false).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ürün arşivlenemedi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Ürün başarıyla silindi (arşivlendi)"})
}
