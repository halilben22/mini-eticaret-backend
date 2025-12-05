package helper // Sabit Kargo Ücreti (Bunu veritabanından veya env'den de çekebilirsin)
import (
	"portProject_development/models"
	"time"

	"gorm.io/gorm"
)

const STANDARD_SHIPPING_FEE = 39.90

// Hesaplama Sonucu Dönüş Tipi
type OrderCalculation struct {
	SubTotal       float64
	ShippingFee    float64
	DiscountAmount float64
	TotalAmount    float64
	PromoCode      string
}

func CalculateOrderTotal(cartItems []models.CartItem, promoCode string, db *gorm.DB) OrderCalculation {
	var subTotal float64 = 0

	//  Ürünlerin Toplamını Bul
	for _, item := range cartItems {
		subTotal += item.Product.Price * float64(item.Quantity)
	}

	// Varsayılan Değerler
	shippingFee := STANDARD_SHIPPING_FEE
	discountAmount := 0.0

	// Promosyon Kontrolü
	if promoCode != "" {
		var promo models.Promotion
		// Kodu bul, aktif mi ve süresi dolmamış mı bak
		if err := db.Where("code = ? AND is_active = ? AND expires_at > ?", promoCode, true, time.Now()).First(&promo).Error; err == nil {

			// Sepet alt limitini karşılıyor mu? (Örn: 500 TL üzeri)
			if subTotal >= promo.MinOrderAmount {

				switch promo.DiscountType {
				case "free_shipping":
					discountAmount = shippingFee // İndirim kargo ücreti kadar olur
					shippingFee = 0              // Kargo sıfırlanır

				case "percentage":
					var discount = (subTotal * promo.DiscountValue) / 100
					discountAmount = discount

				case "fixed_amount":
					discountAmount = promo.DiscountValue
				}
			}
		}
	} else {
		// Eğer kod yoksa ama sepet 1000 TL üstüyse otomatik kargo bedava olsun (İsteğe bağlı)
		if subTotal >= 1000 {
			shippingFee = 0
		}
	}

	// 3. Genel Toplam
	// (SubTotal + Shipping) - Discount
	// Not: Kargo indirimi zaten shippingFee'yi 0 yaparak uygulandı, o yüzden discount'u sadece ürün indirimleri için düşüyoruz.
	// Ancak 'free_shipping' tipinde discountAmount'u sadece loglamak için tutuyoruz, total'den tekrar düşmemeliyiz.

	var finalTotal float64
	if shippingFee == 0 && discountAmount == STANDARD_SHIPPING_FEE {
		// Kargo bedava kampanyasıysa
		finalTotal = subTotal
	} else {
		finalTotal = subTotal + shippingFee - discountAmount
	}

	return OrderCalculation{
		SubTotal:       subTotal,
		ShippingFee:    shippingFee,
		DiscountAmount: discountAmount,
		TotalAmount:    finalTotal,
		PromoCode:      promoCode,
	}
}
