package controllers

import (
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"portProject_development/db"
	"portProject_development/enums"
	"portProject_development/models"
	"strings"
)

const adminRole = string(enums.AdminRole)
const customerRole = string(enums.CustomerRole)

func Register(c *gin.Context) {

	var input struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
		FullName string `json:"full_name"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := models.User{
		Email:        input.Email,
		PasswordHash: input.Password,
		FullName:     input.FullName,
		Role:         customerRole,
	}

	if err := db.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Kullanıcı oluşturulamadı (Email kullanılıyor olabilir)"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Kayıt başarılı!", "user_id": user.ID})

}

// /Sonra bu fonksiyon parçalanacak. çünkü token işlemleri de var içinde.Bu solid kurallarına aykırı
func Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := db.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email veya şifre hatalı"})
		return
	}

	//HASHED PSWD CHECK
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email veya şifre hatalı"})
		return
	}

	//CREATE TOKEN
	claims := jwt.MapClaims{
		"sub":  user.ID,
		"role": user.Role,
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	secretKey := os.Getenv("API_SECRET_KEY")

	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// LOG-OUT FUNC
func Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token bulunamadı"})
		return
	}

	//CLEARING BEARER PREFIX
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	//--BLACK-LIST METHOD---
	//SIMPLY,ADDING TOKEN INTO BLACK LIST WHEN USER LOGOUT DUE TO SECURITY
	blacklistedToken := models.TokenBlackList{
		Token:     tokenString,
		ExpiresAt: time.Now().Add(24 * time.Hour), // Örnek olarak 24 saat sonra silinebilir
	}

	if err := db.DB.Create(&blacklistedToken).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Çıkış yapılamadı"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Başarıyla çıkış yapıldı"})
}
