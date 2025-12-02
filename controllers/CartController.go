package controllers

import (
	"net/http"
	"portProject_development/db" // DB değişkeni nerede tanımlıysa orayı import et (models veya db paketi)
	"portProject_development/models"

	"github.com/gin-gonic/gin"
)

func AddToCart(c *gin.Context) {
	userCtx, _ := c.Get("user")
	userID := userCtx.(models.User).ID

	var input struct {
		ProductID uint `json:"product_id" binding:"required"`
		Quantity  int  `json:"quantity" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	database := db.DB

	var product models.Product
	if err := database.First(&product, input.ProductID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if product.StockQuantity < input.Quantity {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Stock Quantity must be greater than or equal to Quantity"})
		return

	}

	//Sepeti var mı yok mu onu kontrol et

	var cart models.Cart
	if err := database.Where("user_id = ?", userID).FirstOrCreate(&cart, models.Cart{UserID: userID}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sepet hatası"})
		return
	}

	//Sepette zaten varsa  ürünün sayısını artır...

	var cartItem models.CartItem
	err := database.Where("cart_id=? AND product_id=?", cart.ID, input.ProductID).First(&cartItem).Error
	if err == nil { //Yani aynı ürün zaten sepette. bu yüzden adedi artır
		cartItem.Quantity += input.Quantity
		database.Save(&cartItem)

	} else {
		cartItem = models.CartItem{
			CartID:    cart.ID,
			ProductID: input.ProductID,
			Quantity:  input.Quantity,
		}
		database.Create(&cartItem)
	}
	c.JSON(http.StatusOK, gin.H{"message": "Sepete eklendi", "item": cartItem})

}

func GetCart(c *gin.Context) {

	userCtx, _ := c.Get("user")
	UserID := userCtx.(models.User).ID
	database := db.DB

	var cart models.Cart

	if err := database.Preload("Items.Product").Where("user_id = ?", UserID).First(&cart).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "message": "Sepetiniz boş"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": cart})
}

/*
func Checkout(c *gin.Context) {

	userCtx, _ := c.Get("user")
	userID := userCtx.(models.User).ID
	database := db.DB

	var input struct {
		ShippingAdress string `json:"shipping_address" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"Teslimat adresi gereklidir!": err.Error()})
		return
	}

	//Sepet içindeki ürünleri getir
	var cart models.Cart
	if err := database.Preload("Items.Product").Where("user_id= ?", userID).First(&cart).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sepetiniz boş!"})
		return
	}

	if len(cart.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sepetiniz boş, sipariş oluşturulamaz. Lütfen ürün seçin."})
		return
	}

	// 3. TRANSACTION BAŞLAT (Hata olursa her şeyi geri almak için)
	tx := database.Begin()

	order := models.Order{
		UserID:          userID,
		ShippingAddress: input.ShippingAdress,
		Status:          "pending",
		TotalAmount:     0,
	}
	if err := tx.Create(&order).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return

	}

	var totalAmount float64 = 0

	//Sepetteki ürünleri order'a taşı

	for _, cartItem := range cart.Items {
		if cartItem.Product.StockQuantity < cartItem.Quantity {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Stok tükenmiş. " + cartItem.Product.Name})
			return
		}

		//Sipariş detayı oluşturma kısmı
		orderItem := models.OrderItem{
			OrderID:   order.ID,
			ProductID: cartItem.Product.ID,
			Quantity:  cartItem.Quantity,
			UnitPrice: cartItem.Product.Price,
		}
		if err := tx.Create(&orderItem).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Sipariş detayı oluşturulamadı"})
			return
		}

		//Stoktan düşme
		//Modeldeki stoğu güncelle

		newStock := cartItem.Product.StockQuantity - cartItem.Quantity
		if err := tx.Model(&cartItem.Product).Update("stock_quantity", newStock).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		//Toplam parayı hesaplama kısmı
		totalAmount += cartItem.Product.Price * float64(cartItem.Quantity)
	}

	//Sipariş'in toplam ederini güncelle

	if err := tx.Model(&order).Update("total_amount", totalAmount).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	//Siparişi oluştur ve sepeti boşalt,sepet kalabilir.
	if err := tx.Where("cart_id=?", cart.ID).Delete(&models.CartItem{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"message": "Sipariş oluşturuldu", "order_id": order.ID, "total": totalAmount})

}
*/

// 1. SİPARİŞ OLUŞTURMA
func CreateOrderBeforePayment(c *gin.Context) {
	userCtx, _ := c.Get("user")
	userID := userCtx.(models.User).ID
	database := db.DB

	var input struct {
		ShippingAddress string `json:"shipping_address" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Adres gerekli"})
		return
	}

	// Sepeti Getir
	var cart models.Cart
	if err := database.Preload("Items.Product").Where("user_id = ?", userID).First(&cart).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sepet boş"})
		return
	}
	if len(cart.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sepetiniz boş!"})
		return
	}

	tx := database.Begin()

	//eski siparişi silme kısmı
	var existingOrder models.Order
	// Kullanıcının "waiting_payment" durumundaki siparişini bul
	if err := tx.Preload("OrderItems").Where("user_id = ? AND status = ?", userID, "waiting_payment").First(&existingOrder).Error; err == nil {
		// Eğer böyle bir sipariş VARSA (Hata yoksa):

		// siparişi silmeden stokları geri yükle
		for _, oldItem := range existingOrder.OrderItems {
			var product models.Product
			if err := tx.First(&product, oldItem.ProductID).Error; err == nil {
				product.StockQuantity += oldItem.Quantity
				tx.Save(&product)
			}
		}

		// eski siparişin detaylarını sil
		tx.Where("order_id = ?", existingOrder.ID).Delete(&models.OrderItem{})

		// 3. Eski Siparişin Kendisini Sil
		tx.Delete(&existingOrder)

	}

	//Yeni Sipariş'in dbye kaydı
	order := models.Order{
		UserID:          userID,
		ShippingAddress: input.ShippingAddress,
		Status:          "waiting_payment",
		TotalAmount:     0,
	}
	if err := tx.Create(&order).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sipariş oluşturulamadı"})
		return
	}

	var totalAmount float64 = 0

	for _, cartItem := range cart.Items {
		// ... Stok Kontrolü ...
		if cartItem.Product.StockQuantity < cartItem.Quantity {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Stok yetersiz: " + cartItem.Product.Name})
			return
		}

		// ... OrderItem Oluşturma ...
		orderItem := models.OrderItem{
			OrderID:   order.ID,
			ProductID: cartItem.ProductID,
			Quantity:  cartItem.Quantity,
			UnitPrice: cartItem.Product.Price,
		}
		if err := tx.Create(&orderItem).Error; err != nil {
			tx.Rollback()
			return
		}

		// ... Stoktan Düşme ...
		newStock := cartItem.Product.StockQuantity - cartItem.Quantity
		tx.Model(&cartItem.Product).Update("stock_quantity", newStock)

		totalAmount += cartItem.Product.Price * float64(cartItem.Quantity)
	}

	// ... Toplam Güncelleme ve Commit ...
	if err := tx.Model(&order).Update("total_amount", totalAmount).Error; err != nil {
		tx.Rollback()
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message":  "Sipariş oluşturuldu, ödeme bekleniyor.",
		"order_id": order.ID,
		"total":    totalAmount,
	})
}

// Ödeme onayı
func ConfirmPayment(c *gin.Context) {
	userCtx, _ := c.Get("user")
	userID := userCtx.(models.User).ID
	database := db.DB

	var input struct {
		OrderID       uint   `json:"order_id" binding:"required"`
		PaymentMethod string `json:"payment_method" binding:"required"` // Kredi Kartı / Havale
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Eksik veri"})
		return
	}

	tx := database.Begin()

	// A. Siparişi Bul
	var order models.Order
	if err := tx.Where("id = ? AND user_id = ?", input.OrderID, userID).First(&order).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Sipariş bulunamadı"})
		return
	}

	// B. Zaten ödenmiş mi?
	if order.Status == "paid" || order.Status == "shipped" {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bu sipariş zaten ödenmiş"})
		return
	}

	// C. Durumu Güncelle
	order.Status = "paid" // Veya "preparing"
	if err := tx.Save(&order).Where("order_id", order.ID).Error; err != nil {
		tx.Rollback()
		return
	}

	var cart models.Cart
	tx.Where("user_id = ?", userID).First(&cart)
	tx.Where("cart_id = ?", cart.ID).Delete(&models.CartItem{})

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"message": "Ödeme alındı, sipariş hazırlanıyor!"})

}

// 4. SEPET ÜRÜNÜNÜ GÜNCELLE (PUT /cart)
func UpdateCartItem(c *gin.Context) {
	userCtx, _ := c.Get("user")
	userID := userCtx.(models.User).ID
	database := db.DB

	var input struct {
		ProductID uint `json:"product_id" binding:"required"`
		Quantity  int  `json:"quantity" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Sepeti Bul
	var cart models.Cart
	if err := database.Where("user_id = ?", userID).First(&cart).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sepet bulunamadı"})
		return
	}

	// Ürünü Bul
	var cartItem models.CartItem
	if err := database.Where("cart_id = ? AND product_id = ?", cart.ID, input.ProductID).First(&cartItem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ürün sepette yok"})
		return
	}

	// Stok Kontrolü (Yeni girilen miktar stoktan fazla mı?)
	var product models.Product
	database.First(&product, input.ProductID)
	if product.StockQuantity < input.Quantity {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Stok yetersiz!"})
		return
	}

	// --- GÜNCELLEME İŞLEMİ (=) ---
	cartItem.Quantity = input.Quantity
	database.Save(&cartItem)

	c.JSON(http.StatusOK, gin.H{"message": "Sepet güncellendi", "item": cartItem})
}
