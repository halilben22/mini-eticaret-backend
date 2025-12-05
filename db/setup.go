package db

import (
	"fmt"
	"log"
	"os"
	"portProject_development/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase() {
	dsn := os.Getenv("DATABASE_URL")

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Could not connect to database!,ERROR: " + err.Error())
	}
	err = database.AutoMigrate(&models.User{}, &models.Product{}, &models.Order{}, &models.OrderItem{}, &models.Cart{}, &models.CartItem{}, models.TokenBlackList{}, models.Reviews{}, models.Promotion{})

	if err != nil {
		log.Fatal("Migration error :", err)
	}
	DB = database
	fmt.Println("Migration and database connection successfull!")

}
