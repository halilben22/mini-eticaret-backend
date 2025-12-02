package middlewares

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"portProject_development/db"
	"portProject_development/models" // Modül isminle değiştir

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func RequireAuth(c *gin.Context) {
	// 1. Header'dan Token'ı Al
	authHeader := c.GetHeader("Authorization")

	// Header boş mu kontrol et
	if authHeader == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Yetkisiz Erişim! Token gerekli."})
		return
	}

	// 2. Format Kontrolü (Bearer <token>)
	// Tokenlar genelde "Bearer eyJhbGci..." şeklinde gelir. Baştaki "Bearer " kısmını atmalıyız.
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	var blackListed models.TokenBlackList
	if err := db.DB.Where("token=?", tokenString).First(&blackListed).Error; err == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Bu oturum sonlandırılmış. Lütfen tekrar giriş yapın."})
		return
	}
	if tokenString == authHeader {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token formatı hatalı. 'Bearer <token>' şeklinde olmalı."})
		return
	}

	// 3. Token'ı Doğrula ve Çözümle (Parse)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Şifreleme algoritması doğru mu? (HMAC bekliyoruz)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("beklenmeyen imza yöntemi: %v", token.Header["alg"])
		}

		// Gizli anahtarı ver
		return []byte(os.Getenv("API_SECRET_KEY")), nil
	})

	// 4. Token Geçerli mi ve Claim'leri Oku
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		fmt.Println("Buraya girdi")
		// Token süresi dolmuş mu? (exp kontrolü)
		if float64(time.Now().Unix()) > claims["exp"].(float64) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token süresi dolmuş. Lütfen tekrar giriş yapın."})
			return
		}

		// Token içindeki User ID ile veritabanından kullanıcıyı bul
		var user models.User
		db.DB.First(&user, claims["sub"])

		if user.ID == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Kullanıcı bulunamadı."})
			return
		}

		// 5. Kullanıcıyı Context'e Ekle
		// Bu sayede controller içinde "c.Get('user')" diyerek giriş yapan kullanıcıya ulaşabileceğiz.
		c.Set("user", user)

		// Kapıyı Aç, İçeri Girsin
		c.Next()

	} else {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Geçersiz Token: " + err.Error()})
	}
}

//Admin kontrolü

func RequireAdmin(c *gin.Context) {
	userCtx, exists := c.Get("user")
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Kullanıcı bilgisi bulunamadı."})
		return
	}
	user := userCtx.(models.User)
	if user.Role != "admin" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Bu işlem için 'Admin' yetkisi gerekiyor!"})
		return
	}
	c.Next()
}
