package controllers

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

func CreateProduct(c *gin.Context) {

	name := c.PostForm("name")
	description := c.PostForm("description")
	priceStr := c.PostForm("price")
	stockStr := c.PostForm("stock_quantity")
	categoryIDStr := c.PostForm("category_id")

	price, _ := strconv.ParseFloat(priceStr, 64)
	stock, _ := strconv.Atoi(stockStr)
	categoryID, _ := strconv.Atoi(categoryIDStr)

	//Dosya yükle
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

		// Veritabanına kaydedilecek yol (URL)
		// Gerçek hayatta buraya tam domain de eklenebilir (http://localhost:8080/...)
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

func FindProducts(c *gin.Context) {
	// 1. Veritabanı bağlantısını al (Zincirleme sorgu için değişkene atıyoruz)
	// models.DB yerine query değişkenini kullanacağız.
	query := db.DB

	//filtrele (isim ile arama)
	if name := c.Query("name"); name != "" {
		query = query.Where("name ILIKE ?", "%"+name+"%")
	}

	//filtrele(kategori ile)

	if categoryID := c.Query("cat"); categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}

	// filtrele minimum fiyat (min=100)
	if minPrice := c.Query("min"); minPrice != "" {
		query = query.Where("price >= ?", minPrice)
	}

	// filtrele  maksimum Fiyat (max=5000)
	if maxPrice := c.Query("max"); maxPrice != "" {
		query = query.Where("price <= ?", maxPrice)
	}

	var products []models.Product

	if err := query.Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(products), // Kaç ürün bulunduğunu gösterelim
		"data":  products,
	})

}

func FindProductById(c *gin.Context) {
	var product models.Product

	// URL'den gelen ID'yi al (Örn: /products/1)
	id := c.Param("id")

	if err := db.DB.First(&product, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ürün bulunamadı!"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": product})
}

// GET /products/top-rated
// GET /products/top-rated
func GetTopRatedProducts(c *gin.Context) {
	// Hem Ürün verisini, hem ortalamayı, hem de yorum sayısını tutacak yapı
	type Result struct {
		models.Product
		AverageRating float64 `json:"average_rating"`
		ReviewCount   int64   `json:"review_count"` // YENİ: Yorum Sayısı
	}

	var results []Result

	// SQL MANTIĞI:
	// 1. AVG(rating) -> Puan Ortalaması
	// 2. COUNT(id)   -> Yorum Sayısı
	// 3. ORDER BY average_rating DESC, review_count DESC
	//    (Önce puana bak, puanlar eşitse kimin çok yorumu varsa onu öne al)

	err := db.DB.Table("products").
		Select("products.*, AVG(reviews.rating) as average_rating, COUNT(reviews.id) as review_count").
		Joins("LEFT JOIN reviews ON reviews.product_id = products.id").
		Group("products.id").
		Having("AVG(reviews.rating) IS NOT NULL").
		Order("average_rating DESC, review_count DESC"). // <-- KRİTİK NOKTA BURASI
		Limit(5).                                        // İlk 3 yerine 5 yapalım, slider daha zengin olsun
		Scan(&results).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "En iyiler getirilemedi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": results})
}
