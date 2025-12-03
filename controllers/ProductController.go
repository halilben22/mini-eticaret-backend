package controllers

import (
	"fmt"
	"math"
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

// GET /products?page=1&limit=8&name=elma(mesela böyle bir istek)
func FindProducts(c *gin.Context) {
	query := db.DB.Model(&models.Product{}) // Model ile başla

	// 1. FİLTRELERİ UYGULA (Arama, Kategori vb.)
	if name := c.Query("name"); name != "" {
		query = query.Where("name ILIKE ?", "%"+name+"%")
	}
	if categoryID := c.Query("cat"); categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}

	//TOPLAM SAYIYI BUL (Pagination uygulamadan önce!!!)
	var total int64
	query.Count(&total) // Filtrelenmiş sonuçların toplam sayısı

	//  SAYFALAMA AYARLARI
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "8")) // Sayfada 8 ürün gösterelim
	offset := (page - 1) * limit

	//  VERİYİ ÇEK (Limit ve Offsetle beraber)
	var products []models.Product
	if err := query.Limit(limit).Offset(offset).Order("id desc").Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ürünler getirilemedi"})
		return
	}

	// TOPLAM SAYFA HESABI
	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	c.JSON(http.StatusOK, gin.H{
		"data": products,
		"meta": gin.H{
			"total_items": total,
			"total_pages": totalPages,
			"page":        page,
			"limit":       limit,
		},
	})
}

func FindProductById(c *gin.Context) {
	var product models.Product

	// URL'den gelen ID'yi al (Örnek=/products/1)
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
	//  AVG(rating) -> Puan Ortalaması
	//  COUNT(id)   -> Yorum Sayısı
	// ORDER BY average_rating DESC, review_count DESC
	//   (Önce puana bak, puanlar eşitse kimin çok yorumu varsa onu öne al)

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
