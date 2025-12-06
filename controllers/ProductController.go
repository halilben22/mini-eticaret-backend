package controllers

import (
	"math"
	"net/http"
	"portProject_development/db"
	"portProject_development/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ProductFilterParams struct {
	Name       string
	CategoryID string
	MinPrice   float64
	MaxPrice   float64
	MinRating  float64
	SortBy     string // "price_asc", "price_desc", "rating", "reviews", "newest"
	Page       int
	Limit      int
}

type ProductWithStats struct {
	models.Product
	AverageRating float64 `json:"average_rating"`
	ReviewCount   int64   `json:"review_count"`
}

// GET PRODUCTS BY PAGE FILTER(PAGINATION PART) /products?page=1&limit=8&name=apple
// GET /products

func FindProducts(c *gin.Context) {
	// 1. URL Parametrelerini Al
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "8"))
	minPrice, _ := strconv.ParseFloat(c.Query("min_price"), 64)
	maxPrice, _ := strconv.ParseFloat(c.Query("max_price"), 64)
	minRating, _ := strconv.ParseFloat(c.Query("min_rating"), 64)

	// 2. Parametreleri Yapılandır
	params := ProductFilterParams{
		Name:       c.Query("name"),
		CategoryID: c.Query("cat"),
		SortBy:     c.Query("sort"), // YENİ: ?sort=price_asc gibi
		MinPrice:   minPrice,
		MaxPrice:   maxPrice,
		MinRating:  minRating,
		Page:       page,
		Limit:      limit,
	}

	// 3. İşçi Fonksiyonu Çağır
	products, total, err := getFilteredProducts(params)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ürünler getirilemedi"})
		return
	}

	// 4. Cevabı Dön
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

// Yardımcı Fonksiyon: Filtreleme ve Sıralama Motoru
func getFilteredProducts(params ProductFilterParams) ([]ProductWithStats, int64, error) {
	var products []ProductWithStats
	var total int64

	// 1. Temel Sorgu (Joins ve Select)
	query := db.DB.Table("products").
		Select("products.*, COALESCE(AVG(reviews.rating), 0) as average_rating, COUNT(reviews.id) as review_count").
		Joins("LEFT JOIN reviews ON reviews.product_id = products.id").
		Group("products.id")

	// 2. Filtreleri Uygula
	if params.Name != "" {
		query = query.Where("products.name ILIKE ?", "%"+params.Name+"%") // Contains araması
	}
	if params.CategoryID != "" {
		query = query.Where("products.category_id = ?", params.CategoryID)
	}
	if params.MinPrice > 0 {
		query = query.Where("products.price >= ?", params.MinPrice)
	}
	if params.MaxPrice > 0 {
		query = query.Where("products.price <= ?", params.MaxPrice)
	}
	// Puan Filtresi (Aggregate olduğu için HAVING)
	if params.MinRating > 0 {
		query = query.Having("COALESCE(AVG(reviews.rating), 0) >= ?", params.MinRating)
	}

	// 3. Toplam Sayıyı Bul (Pagination için)
	// Not: Group By varken Count almak biraz tricky olabilir, GORM bunu genelde halleder.
	// Daha performanslı olması için burada ayrı bir count query yazılabilir ama şimdilik bu yeterli.
	db.DB.Table("(?) as subquery", query).Count(&total)

	// 4. Sıralama (Sorting) Mantığı
	switch params.SortBy {
	case "price_asc": // Fiyat Artan
		query = query.Order("products.price ASC")
	case "price_desc": // Fiyat Azalan
		query = query.Order("products.price DESC")
	case "rating": // En Yüksek Puan
		query = query.Order("average_rating DESC")
	case "reviews": // En Çok Değerlendirme
		query = query.Order("review_count DESC")
	case "newest": // En Yeniler (Varsayılan)
		query = query.Order("products.created_at DESC")
	default:
		query = query.Order("products.id DESC") // Varsayılan sıralama
	}

	// 5. Sayfalama ve Çalıştırma
	offset := (params.Page - 1) * params.Limit
	err := query.Preload("Category").Limit(params.Limit).Offset(offset).Scan(&products).Error

	return products, total, err
}
func FindProductById(c *gin.Context) {
	var product models.Product

	id := c.Param("id")

	if err := db.DB.First(&product, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ürün bulunamadı!"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": product})
}

// FUNCTION FOR TOP RATED PRODUCTS
func GetTopRatedProducts(c *gin.Context) {

	//STRUCT TO KEEP PRODUCT DATA,AVERAGE AND REVIEW COUNT
	type Result struct {
		models.Product
		AverageRating float64 `json:"average_rating"`
		ReviewCount   int64   `json:"review_count"` // YENİ: Yorum Sayısı
	}

	var results []Result

	// ORDER BY average_rating DESC, review_count DESC
	//FIRST LOOK FOR STAR POINT THEN LOOK FOR REVIEW COUNT

	err := db.DB.Table("products").
		Select("products.*, AVG(reviews.rating) as average_rating, COUNT(reviews.id) as review_count").
		Joins("LEFT JOIN reviews ON reviews.product_id = products.id").
		Group("products.id").
		Having("AVG(reviews.rating) IS NOT NULL").
		Order("average_rating DESC, review_count DESC").
		Limit(5). // GET FIRST 5
		Scan(&results).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "En iyiler getirilemedi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": results})
}
