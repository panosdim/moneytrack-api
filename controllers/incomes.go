package controllers

import (
	"moneytrack-api/models"
	"moneytrack-api/utils/token"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
)

type GetIncomeInput struct {
	AfterDate time.Time `form:"after_date" time_format:"2006-01-02"`
}

func GetIncomes(c *gin.Context) {
	userId, err := token.ExtractTokenID(c)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var incomes []models.Income
	var input GetIncomeInput

	if c.ShouldBind(&input) == nil {
		//return only expenses that are after requested date
		if err := models.DB.Order("date desc").Find(&incomes, "user_id = ? AND date >= ?", userId, input.AfterDate).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else {
		if err := models.DB.Order("date desc").Find(&incomes, "user_id = ?", userId).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, incomes)
}

func GetIncome(c *gin.Context) {
	userId, err := token.ExtractTokenID(c)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := c.Param("id")
	var income models.Income

	if err := models.DB.Where("user_id = ?", userId).Find(&income, id).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only view your own incomes"})
		return
	}

	c.JSON(http.StatusOK, income)
}

type SaveIncomeInput struct {
	Amount  float64 `json:"amount" binding:"required"`
	Date    string  `json:"date" binding:"required"`
	Comment string  `json:"comment"`
}

func SaveIncome(c *gin.Context) {

	userId, err := token.ExtractTokenID(c)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var input SaveIncomeInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newIncome := models.Income{}

	copier.Copy(&newIncome, &input)
	newIncome.UserID = userId

	if err := models.DB.Create(&newIncome).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, newIncome)
}

type UpdateIncomeInput struct {
	Amount  float64 `json:"amount"`
	Date    string  `json:"date"`
	Comment string  `json:"comment"`
}

func UpdateIncome(c *gin.Context) {

	userId, err := token.ExtractTokenID(c)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var input UpdateIncomeInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := c.Param("id")

	var income models.Income

	if err := models.DB.Where("user_id = ?", userId).Find(&income, id).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only update your own incomes"})
		return
	}

	if err := models.DB.Model(&income).Updates(input).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, income)
}

func DeleteIncome(c *gin.Context) {

	userId, err := token.ExtractTokenID(c)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := c.Param("id")

	var income models.Income

	if err := models.DB.First(&income, id).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if income.UserID != userId {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only delete your own incomes"})
		return
	}

	if err := models.DB.Delete(&income).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
