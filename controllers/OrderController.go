package controllers

import (
	"net/http"
	"portProject_development/db"
	"portProject_development/enums"
	"portProject_development/models" // Kendi modül adın

	"github.com/gin-gonic/gin"
)

// ORDER STATUS ENUM
const pendingStatus = string(enums.OrderStatusPending)

type OrderInput struct {
	ShippingAddress string `json:"shipping_address" binding:"required"`
	Items           []struct {
		ProductID uint `json:"product_id" binding:"required"`
		Quantity  int  `json:"quantity" binding:"required"`
	} `json:"items" binding:"required"`
}

func CreateOrder(c *gin.Context) {
	userCtx, _ := c.Get("user")
	user := userCtx.(*models.User)

	var input OrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order := models.Order{
		UserID:          user.ID,
		ShippingAddress: input.ShippingAddress,
		Status:          pendingStatus,
		TotalAmount:     0,
	}

	//TRANSACTION BEGINS
	tx := db.DB.Begin()

	if err := tx.Create(&order).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sipariş oluşturulamadı"})
		return
	}
	var totalAmount float64 = 0
	for _, itemInput := range input.Items {
		var product models.Product

		//FIND ITEM
		if err := tx.First(&product, itemInput.ProductID).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Ürün bulunamadı ID: " + string(rune(itemInput.ProductID))})
			return
		}
		//STOCK AND ITEM INPUT QUANTITY COMPARE
		if product.StockQuantity < itemInput.Quantity {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Stok yetersiz: " + product.Name})
			return
		}

		orderItem := models.OrderItem{
			OrderID:   order.ID,
			ProductID: product.ID,
			Quantity:  itemInput.Quantity,
			UnitPrice: product.Price,
		}

		if err := tx.Create(&orderItem).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Sipariş detayı eklenemedi"})
			return
		}
		product.StockQuantity -= itemInput.Quantity //DECREASE STOCK

		if err := tx.Save(&product).Error; err != nil {
			tx.Rollback()
			return
		}

		totalAmount += product.Price * float64(itemInput.Quantity)
	}

	order.TotalAmount = totalAmount
	tx.Save(&order)
	tx.Commit()

	c.JSON(http.StatusCreated, gin.H{"message": "Sipariş alındı!", "order_id": order.ID, "total": totalAmount})
}

// GET ALL ORDERS
func GetMyOrders(c *gin.Context) {
	userCtx, _ := c.Get("user")
	user := userCtx.(models.User)

	var orders []models.Order
	//WE USED "PRELOAD" BECAUSE WE NEED PRODUCT DETAILS
	db.DB.Preload("OrderItems.Product").Where("user_id = ?", user.ID).Find(&orders)
	c.JSON(http.StatusOK, gin.H{"data": orders})
}
