package main

import (
	"log"
	"portProject_development/controllers"
	"portProject_development/controllers/admin"
	"portProject_development/db"
	"portProject_development/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/gin-contrib/cors"
)

//TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(".env dosyası okunamadı")
	}
	db.ConnectDatabase()
	r := gin.Default()

	//Cors ayarları
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true                                                               // Şimdilik herkese izin ver (Test için)
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"} // Token başlığına izin ver
	r.Use(cors.New(config))

	//Resim yüklemek için
	r.Static("/uploads", "./uploads")

	//Public Rotalar
	r.GET("/products", controllers.FindProducts)
	r.POST("/admin/promotion/:id", admin.Promotion)
	r.GET("/products/:id", controllers.FindProductById)

	r.POST("/register", controllers.Register)
	r.POST("/login", controllers.Login)

	r.GET("/products/:id/reviews", controllers.GetProductReview)

	// YENİ: En Çok Puan Alanlar
	r.GET("/products/top-rated", controllers.GetTopRatedProducts)

	//MIDDLEWARES***************************************************
	//Korunmuş rotalar
	//Middleware ile çalışacak

	customerGroup := r.Group("/")
	customerGroup.Use(middlewares.RequireAuth)
	{
		customerGroup.POST("/logout", controllers.Logout)

		customerGroup.POST("/orders", controllers.CreateOrder)
		customerGroup.GET("/orders", controllers.GetMyOrders)

		customerGroup.POST("/cart", controllers.AddToCart)     // Sepete Ekle
		customerGroup.GET("/cart", controllers.GetCart)        //sepeti getir
		customerGroup.PUT("/cart", controllers.UpdateCartItem) //sepeti güncelle

		// SİPARİŞ OLUŞTUR (Sepet kalır)
		customerGroup.POST("/create-order", controllers.CreateOrderBeforePayment)

		// ÖDEME YAP (Sepet silinir)
		customerGroup.POST("/payment", controllers.ConfirmPayment)

		//Yorum yazma
		customerGroup.POST("/products/:id/reviews", controllers.AddReview)

		//sepet silme fonksiyonu
		customerGroup.DELETE("/cart/:id", controllers.RemoveFromCart)

	}

	adminGroup := r.Group("/")
	adminGroup.Use(middlewares.RequireAuth)
	adminGroup.Use(middlewares.RequireAdmin)
	{

		adminGroup.POST("/admin/add-product", controllers.CreateProduct)

		adminGroup.GET("/admin/stats", admin.GetDashboardStats)      // İstatistik
		adminGroup.GET("/admin/orders", admin.GetAllOrders)          // Tüm siparişler
		adminGroup.PUT("/admin/orders/:id", admin.UpdateOrderStatus) // Durum güncelleme

		// Promosyon Yönetimi
		adminGroup.POST("/admin/promotions", admin.CreatePromotion)       // (Bunu daha önce eklemiştik)
		adminGroup.GET("/admin/promotions", admin.GetPromotions)          // <--- YENİ
		adminGroup.DELETE("/admin/promotions/:id", admin.DeletePromotion) // <--- YENİ
	}

	r.Run(":8080")

}
