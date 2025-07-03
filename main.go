package main

import (
	"fmt"
	"log"
	"moneytrack-api/controllers"
	"moneytrack-api/middlewares"
	"moneytrack-api/models"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {

	models.ConnectDataBase()

	r := gin.Default()
	r.Use(middlewares.CORSMiddleware())

	public := r.Group("/api")

	public.POST("/login", controllers.Login)

	public.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": "1.0"})
	})

	private := r.Group("/api")

	private.Use(middlewares.JwtAuthMiddleware())
	{
		// User Info
		private.GET("/user", controllers.CurrentUser)

		// Years
		private.GET("/years", controllers.Years)

		// Category API
		private.GET("/category", controllers.GetCategories)
		private.GET("/category/:id", controllers.GetCategory)
		private.POST("/category", controllers.SaveCategory)
		private.PUT("/category/:id", controllers.UpdateCategory)
		private.DELETE("/category/:id", controllers.DeleteCategory)

		// Income API
		private.GET("/income", controllers.GetIncomes)
		private.GET("/income/:id", controllers.GetIncome)
		private.POST("/income", controllers.SaveIncome)
		private.PUT("/income/:id", controllers.UpdateIncome)
		private.DELETE("/income/:id", controllers.DeleteIncome)

		// Expense API
		private.GET("/expense", controllers.GetExpenses)
		private.GET("/expense/:id", controllers.GetExpense)
		private.POST("/expense", controllers.SaveExpense)
		private.PUT("/expense/:id", controllers.UpdateExpense)
		private.DELETE("/expense/:id", controllers.DeleteExpense)
	}

	if err := r.Run(fmt.Sprintf(":%s", os.Getenv("PORT"))); err != nil {
		log.Fatalf("Error starting server")
	}
}
