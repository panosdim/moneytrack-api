package controllers

import (
	"moneytrack-api/models"
	"moneytrack-api/utils/token"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Years(c *gin.Context) {

	userId, err := token.ExtractTokenID(c)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var years []int16

	if err := models.DB.Model(&models.Expense{}).Select("distinct YEAR(date) as years").Order("years desc").Where("user_id = ?", userId).Pluck("years", &years).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, years)
}
