package controllers

import (
	"net/http"
	"portProject_development/db"
	"portProject_development/enums"
	"portProject_development/helper"
	"portProject_development/models"

	"github.com/gin-gonic/gin"
)

// STATUS ENUMS
const orderStatusPaid = string(enums.OrderStatusPaid)
const orderStatusShipped = string(enums.OrderStatusShipped)
const orderStatusWaitingPayment = string(enums.OrderStatusWaitingPayment)

// ADDING PRODUCTS INTO CART
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

	//STOCK AND INPUT COMPARING BECAUSE STOCK CAN'T BE LOWER THAN INPUT QUANTITY
	if product.StockQuantity < input.Quantity {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Stock Quantity must be greater than or equal to Quantity"})
		return

	}

	//CHECKING IF CART ALREADY EXIST OR NOT IF NOT,CREATE AN EMPTY ONE(PERMANENTLY)
	var cart models.Cart
	if err := database.Where("user_id = ?", userID).FirstOrCreate(&cart, models.Cart{UserID: userID}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sepet hatası"})
		return
	}

	//IF THAT ITEM WHICH WANTED TO BE ADDED BY USER IS ALREADY EXITS,SIMPLY INCREASE...
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

// GET ALL CART DATA
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

// CREATING ORDER
func CreateOrderBeforePayment(c *gin.Context) {
	userCtx, _ := c.Get("user")
	userID := userCtx.(models.User).ID
	database := db.DB

	var input struct {
		ShippingAddress string `json:"shipping_address" binding:"required"`
		PromoCode       string `json:"promo_code"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Adres gerekli"})
		return
	}

	// GET CART
	var cart models.Cart
	if err := database.Preload("Items.Product").Where("user_id = ?", userID).First(&cart).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sepet boş"})
		return
	}
	if len(cart.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sepetiniz boş!"})
		return
	}

	//CALCULATION HELPER FUNC FOR CALCULATING TOTAL PAYMENT
	calc := helper.CalculateOrderTotal(cart.Items, input.PromoCode, db.DB)
	//TRANSACTION BEGIN HERE
	tx := database.Begin()

	//DELETING OLD ORDER
	//WHY WE SHOULD DO THIS? IF USER DOESN'T WANT TO PAY IN PAYMENT SCREEN AND BACK TO CART AGAIN AND IF CHANGE SOME
	//ITEMS ON THERE,WE MUST DELETE THE OLD ORDER SO WE CAN ADD NEW ONE IN DB
	var existingOrder models.Order
	// FIND USER'S ORDER ON THE "WAITING_PAYMENT" STATUS
	if err := tx.Preload("OrderItems").Where("user_id = ? AND status = ?", userID, orderStatusWaitingPayment).First(&existingOrder).Error; err == nil {
		// IF THAT ORDER EXISTS AND NO ERROR

		// FIRST,GET THE ITEMS ON THE ORDER
		for _, oldItem := range existingOrder.OrderItems {
			var product models.Product
			if err := tx.First(&product, oldItem.ProductID).Error; err == nil {
				//RESTORE THE STOCK OF OLD ITEMS
				product.StockQuantity += oldItem.Quantity
				tx.Save(&product)
			}
		}

		// DELETE
		tx.Where("order_id = ?", existingOrder.ID).Delete(&models.OrderItem{})

		// DELETE OLD ORDER'S DETAILS
		tx.Delete(&existingOrder)

	}

	//SAVING OF NEW ORDER'S RECORDS
	order := models.Order{
		UserID:           userID,
		ShippingAddress:  input.ShippingAddress,
		Status:           orderStatusWaitingPayment,
		SubTotal:         calc.SubTotal,
		ShippingFee:      calc.ShippingFee,
		DiscountAmount:   calc.DiscountAmount,
		TotalAmount:      calc.TotalAmount,
		AppliedPromoCode: calc.PromoCode,
	}
	if err := tx.Create(&order).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sipariş oluşturulamadı"})
		return
	}

	for _, cartItem := range cart.Items {
		// WE SHOULD CHECK THE STOCK AGAIN BECAUSE SOMEONE MIGHT ALREADY BUY THAT SPECIFIC ITEM
		if cartItem.Product.StockQuantity < cartItem.Quantity {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Stok yetersiz: " + cartItem.Product.Name})
			return
		}

		// CREATING ORDER ITEM DETAILS
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

		// DECREASING STOCK
		newStock := cartItem.Product.StockQuantity - cartItem.Quantity
		tx.Model(&cartItem.Product).Update("stock_quantity", newStock)

	}

	// TOTAL AMOUNT UPDATING AND COMMITING TRANSACTION
	if err := tx.Model(&order).Update("total_amount", calc.TotalAmount).Error; err != nil {
		tx.Rollback()
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message":  "Sipariş oluşturuldu",
		"order_id": order.ID,
		"total":    order.TotalAmount,
		"shipping": order.ShippingFee,
	})
}

// CONFIRM PAYMENT
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

	// FIND THE ORDER
	var order models.Order
	if err := tx.Where("id = ? AND user_id = ?", input.OrderID, userID).First(&order).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Sipariş bulunamadı"})
		return
	}

	// CHECKING IF ALREADY PAID OR NOT
	if order.Status == orderStatusPaid || order.Status == orderStatusShipped {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bu sipariş zaten ödenmiş"})
		return
	}

	// UPDATE THE STATUS
	order.Status = orderStatusPaid // OR "PREPARING"
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

// UPDATE CART PRODUCT
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

	// FIND THE CART
	var cart models.Cart
	if err := database.Where("user_id = ?", userID).First(&cart).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sepet bulunamadı"})
		return
	}

	// FIND THE PRODUCT IN THE CART
	var cartItem models.CartItem
	if err := database.Where("cart_id = ? AND product_id = ?", cart.ID, input.ProductID).First(&cartItem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ürün sepette yok"})
		return
	}

	// STOCK CHECK
	var product models.Product
	database.First(&product, input.ProductID)
	if product.StockQuantity < input.Quantity {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Stok yetersiz!"})
		return
	}

	// UPDATING PART
	cartItem.Quantity = input.Quantity
	database.Save(&cartItem)

	c.JSON(http.StatusOK, gin.H{"message": "Sepet güncellendi", "item": cartItem})
}

// DELETE PRODUCT FROM THE CART
func RemoveFromCart(c *gin.Context) {
	userCtx, _ := c.Get("user")
	userID := userCtx.(models.User).ID
	database := db.DB

	// GET ID FROM CONTEXT
	productID := c.Param("id")

	// GET CART
	var cart models.Cart
	if err := database.Where("user_id = ?", userID).First(&cart).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sepet bulunamadı"})
		return
	}

	//DELETING PART
	result := database.Where("cart_id = ? AND product_id = ?", cart.ID, productID).Delete(&models.CartItem{})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Silme işlemi başarısız"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Ürün sepetten silindi"})
}
