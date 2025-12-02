package models

import (
	"portProject_development/enums"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Email        string    `gorm:"unique" json:"email"`
	PasswordHash string    `json:"-"`
	FullName     string    `json:"full_name"`
	Role         string    `gorm:"default:customer" json:"role"`
	Orders       []Order   `json:"orders,omitempty"` // Bir kullanıcının çok siparişi olabilir
	CreatedAt    time.Time `json:"created_at"`
}

type Product struct {
	ID          uint    `gorm:"primaryKey" json:"id"`
	CategoryID  uint    `json:"category_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`

	StockQuantity int       `json:"stock_quantity"`
	ImageURL      string    `json:"image_url" `
	IsActive      bool      `gorm:"default:true" json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
}

type Order struct {
	ID              uint        `gorm:"primaryKey" json:"id"`
	UserID          uint        `json:"user_id"`
	User            User        `gorm:"foreignKey:UserID" json:"-"` // İlişki
	Status          string      `gorm:"default:pending" json:"status"`
	TotalAmount     float64     `json:"total_amount"`
	ShippingAddress string      `json:"shipping_address"`
	OrderItems      []OrderItem `json:"order_items"`
	CreatedAt       time.Time   `json:"created_at"`
}

type OrderItem struct {
	ID        uint    `gorm:"primaryKey" json:"id"`
	OrderID   uint    `json:"order_id"`
	ProductID uint    `json:"product_id"`
	Product   Product `gorm:"foreignKey:ProductID" json:"product"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

type Cart struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	UserID    uint       `json:"user_id"`
	User      User       `gorm:"foreignKey:UserID" json:"-"`
	Items     []CartItem `json:"items"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type CartItem struct {
	ID        uint    `gorm:"primaryKey" json:"id"`
	CartID    uint    `json:"cart_id"`
	ProductID uint    `json:"product_id"`
	Product   Product `gorm:"foreignKey:ProductID" json:"product"`
	Quantity  int     `json:"quantity"`
}

type Reviews struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `json:"user_id"`
	User      User      `gorm:"foreignKey:UserID" json:"-"`
	ProductID uint      `json:"product_id"`
	Rating    int       `json:"rating"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
}

type TokenBlackList struct {
	ID        uint64    `gorm:"primaryKey"`
	Token     string    `gorm:"uniqueIndex;not null"` //Token
	ExpiresAt time.Time `json:"expires_at"`           //Token'ın normal geçerlilik süresi
	CreatedAt time.Time
}

type Promotion struct {
	ID            uint                `gorm:"primaryKey" json:"id"`
	ProductID     uint                `json:"product_id"`
	PromotionType enums.PromotionType `json:"promotion_type"`
	Discount      Discount            `gorm:"foreignKey:PromotionID" json:"discount"`
	//Discount      float64             `json:"discount"`
	//DiscountPrice float64             `json:"discount_price"`
}

type Discount struct {
	ID            uint    `gorm:"primaryKey" json:"id"`
	PromotionID   uint    `json:"promotion_id"`
	ProductID     uint    `json:"product_id"`
	DiscountRate  float64 `json:"discount_rate"`
	DiscountPrice float64 `json:"discount_price"`
}

type ShipDiscount struct {
	ID             uint    `gorm:"primaryKey" json:"id"`
	ProductID      uint    `json:"product_id"`
	PromotionID    uint    `json:"promotion_id"`
	ShipPriceLimit float64 `json:"ship_price_limit"`
	Discount       float64 `json:"discount"`
	DiscountPrice  float64 `json:"discount_price"`
}

func (u *User) BeforeSave(tx *gorm.DB) error {

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.PasswordHash), bcrypt.DefaultCost)

	if err != nil {
		return err
	}
	u.PasswordHash = string(hashedPassword)
	return nil

}
