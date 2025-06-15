package main

import (
	"log"
	"os"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "github.com/yangirxd/goods-service/docs"
	"github.com/yangirxd/goods-service/internal/cache"
	"github.com/yangirxd/goods-service/internal/db"
	"github.com/yangirxd/goods-service/internal/handler"
	"github.com/yangirxd/goods-service/internal/queue"
	"github.com/yangirxd/goods-service/internal/repository"
)

// @title           Goods Service API
// @version         1.0
// @description     Service for managing goods with caching and event logging.
// @host           localhost:8080
// @BasePath       /
func main() {
	pgDSN := getEnv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/testdb?sslmode=disable")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	natsURL := getEnv("NATS_URL", "nats://localhost:4222")
	clickhouseURL := getEnv("CLICKHOUSE_URL", "tcp://localhost:9000?database=logs")

	// Подключение к Postgres
	pg, err := db.NewPostgres(pgDSN)
	if err != nil {
		log.Fatalf("Ошибка подключения к Postgres: %v", err)
	}
	defer pg.Close()

	// Запуск миграций
	if err := db.RunMigrations(pg, "migrations/postgres"); err != nil {
		log.Fatalf("Ошибка миграций: %v", err)
	}

	// Подключение к Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	defer redisClient.Close()

	// Подключение к NATS
	logger, err := queue.NewLogger(natsURL)
	if err != nil {
		log.Fatalf("Ошибка подключения к NATS: %v", err)
	}
	defer logger.Close()

	// Создание и запуск потребителя логов
	logConsumer, err := queue.NewLogConsumer(natsURL, clickhouseURL)
	if err != nil {
		log.Fatalf("Ошибка создания потребителя логов: %v", err)
	}
	defer logConsumer.Close()

	// Запускаем обработку логов в отдельной горутине
	go func() {
		if err := logConsumer.Start(); err != nil {
			log.Printf("Ошибка запуска потребителя логов: %v", err)
		}
	}()

	goodsRepo := repository.NewGoodsRepository(pg)
	goodsCache := cache.NewGoodsCache(redisClient)
	goodsHandler := handler.NewGoodsHandler(goodsRepo, goodsCache, logger)

	r := gin.Default()

	// Swagger документация
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	goods := r.Group("/goods")
	{
		goods.POST("/create", goodsHandler.Create)
		goods.GET("/get/:id", goodsHandler.Get)
		goods.PATCH("/update/:id", goodsHandler.Update)
		goods.DELETE("/remove/:id", goodsHandler.Delete)
		goods.GET("/list", goodsHandler.List)
		goods.PATCH("/reprioritize", goodsHandler.Reprioritize)
	}

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
