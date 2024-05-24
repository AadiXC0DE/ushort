package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	DB    *gorm.DB
	Cache *redis.Client
)

type TinyURL struct {
	ID          uint   `gorm:"primaryKey"`
	OriginalURL string `gorm:"uniqueIndex"`
	ShortURL    string `gorm:"uniqueIndex"`
}

func main() {
	var err error

	//database connection
	dsn := "host=localhost user=user password=password dbname=ushort_db port=5432 sslmode-disable"
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect to db: %v", err)
	}

	//migrations
	DB.AutoMigrate(&TinyURL{})

	//redis connection
	Cache = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	//create gin router
	r := gin.Default()

	//routes
	r.POST("/shorten", shortenURL)
	r.GET("/:shortURL", redirectURL)

	if err := r.Run(); err != nil {
		log.Fatal("failed to run server: %v", err)
	}
}

func shortenURL(c *gin.Context) {
	var json struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tinyURL := TinyURL{OriginalURL: json.URL}
	DB.Create(&tinyURL)

	tinyURL.ShortURL = encode(tinyURL.ID)
	DB.Save(&tinyURL)

	c.JSON(http.StatusOK, gin.H{"shortURL": tinyURL.ShortURL})
}

func redirectURL(c *gin.Context) {
	shortURL := c.Param("shortURL")

	var tinyURL TinyURL
	if err := DB.Where("short_url=?", shortURL).First(&tinyURL).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		return
	}
	c.Redirect(http.StatusFound, tinyURL.OriginalURL)
}

func encode(n uint) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var encoded string
	base := uint(len(alphabet))
	for n > 0 {
		encoded = string(alphabet[n%base]) + encoded
		n /= base
	}
	return encoded
}
