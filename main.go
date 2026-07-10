package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type URLData struct {
	OriginalURL string    `json:"original_url" bson:"original_url"`
	ShortCode   string    `json:"short_code" bson:"short_code"`
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
	Clicks      int       `json:"clicks" bson:"clicks"`
}

const (
	letters        = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	mongoURL       = "mongodb://admin:secret123@localhost:27017"
	mDB            = "url_shortener"
	collection_url = "urls"
	port           = ":8090"
	baseURL        = "http://localhost" + port
)

var (
	collection *mongo.Collection
	ctx        = context.TODO()
)

func main() {
	rand.Seed(time.Now().UnixNano())

	InitDB()

	r := gin.Default()

	r.GET("/", Home)
	r.NoRoute(NotFound)
	r.POST("/api/shorten", SetURL)
	r.GET("/:code", GetURL)

	log.Println("server is running on " + baseURL)
	r.Run(port)
}

func GenerateShortCode(n int) string {
	b := make([]byte, n)

	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func InitDB() {
	clientOptions := options.Client().ApplyURI(mongoURL)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal("could not connect to MongoDB: ", err)
	}

	collection = client.Database(mDB).Collection(collection_url)
	log.Println("Successfully connected to MongoDB")
}

func Home(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Welcome to the URL Shortener API!",
		"endpoints": gin.H{
			"shorten_url": "POST /api/shorten",
			"redirect":    "GET /:code",
		},
		"status": "Server is running smoothly",
	})
}

func NotFound(c *gin.Context) {
	c.Redirect(http.StatusPermanentRedirect, "/")
}

func SetURL(c *gin.Context) {
	var reqBody struct {
		OriginalURL string `json:"original_url"`
	}

	if err := c.BindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	shortCode := GenerateShortCode(6)

	newURL := URLData{
		OriginalURL: reqBody.OriginalURL,
		ShortCode:   shortCode,
		CreatedAt:   time.Now(),
		Clicks:      0,
	}

	_, err := collection.InsertOne(ctx, newURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save URL"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"short_url":    baseURL + "/" + shortCode,
		"short_code":   shortCode,
		"original_url": reqBody.OriginalURL,
	})
}

func GetURL(c *gin.Context) {
	shortCode := c.Param("code")

	var result URLData

	filter := bson.M{"short_code": shortCode}

	update := bson.M{"$inc": bson.M{"clicks": 1}}

	err := collection.FindOneAndUpdate(ctx, filter, update).Decode(&result)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		return
	}

	c.Redirect(http.StatusFound, result.OriginalURL)
}
