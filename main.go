package main

import (
	"log"
	"parking-system/config"
	"parking-system/handlers"
	"parking-system/middleware"
	"parking-system/models"
	"parking-system/services"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	db, err := models.InitDB(cfg)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	models.InitParkingSpaces(db, cfg.Parking.TotalSpaces)

	r := gin.Default()

	r.Use(middleware.CORS())

	authHandler := handlers.NewAuthHandler(db, cfg)
	parkingSvc := services.NewParkingService(cfg)
	parkingHandler := handlers.NewParkingHandler(db, parkingSvc)
	spaceHandler := handlers.NewSpaceHandler(db, cfg)
	cardHandler := handlers.NewCardHandler(db, cfg)
	billHandler := handlers.NewBillHandler(db, cfg)

	api := r.Group("/api/v1")
	{
		api.POST("/auth/login", authHandler.Login)
		api.POST("/auth/register", authHandler.Register)

		auth := api.Group("")
		auth.Use(middleware.JWTAuth(cfg.JWT.Secret))
		{
			auth.GET("/auth/me", authHandler.Me)

			admin := auth.Group("")
			admin.Use(middleware.RequireRole("admin"))
			{
				admin.POST("/users", authHandler.CreateUser)
				admin.GET("/users", authHandler.ListUsers)
				admin.DELETE("/users/:id", authHandler.DeleteUser)
			}

			auth.POST("/parking/entry", parkingHandler.Entry)
			auth.POST("/parking/exit", parkingHandler.Exit)
			auth.GET("/parking/records", parkingHandler.ListRecords)
			auth.GET("/parking/record/:id", parkingHandler.GetRecord)

			auth.GET("/spaces/status", spaceHandler.Status)
			auth.GET("/spaces/list", spaceHandler.List)
			admin.POST("/spaces/add", spaceHandler.AddSpaces)

			auth.POST("/cards", middleware.RequireRole("admin"), cardHandler.CreateCard)
			auth.POST("/cards/renew", middleware.RequireRole("admin"), cardHandler.RenewCard)
			auth.GET("/cards", cardHandler.ListCards)
			auth.GET("/cards/:id", cardHandler.GetCard)
			auth.GET("/cards/plate/:plate", cardHandler.GetCardByPlate)
			auth.DELETE("/cards/:id", middleware.RequireRole("admin"), cardHandler.DeleteCard)

			auth.GET("/bills/daily", billHandler.DailyStats)
			auth.GET("/bills/monthly", billHandler.MonthlyStats)
			auth.GET("/bills/export", billHandler.Export)
			auth.GET("/bills/list", billHandler.ListBills)
		}
	}

	log.Printf("服务启动于端口 %d", cfg.Server.Port)
	if err := r.Run(cfg.Server.Addr()); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
