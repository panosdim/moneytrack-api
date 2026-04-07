package controllers

import (
	"moneytrack-api/models"
	"moneytrack-api/utils/token"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type GetExpenseInput struct {
	AfterDate time.Time `form:"after_date" time_format:"2006-01-02"`
}

func GetExpenses(c *gin.Context) {
	userId, err := token.ExtractTokenID(c)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var expenses []models.Expense
	var input GetExpenseInput

	if c.ShouldBind(&input) == nil {
		//return only expenses that are after requested date
		if err := models.DB.Order("date desc").Find(&expenses, "user_id = ? AND date >= ?", userId, input.AfterDate).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else {
		if err := models.DB.Order("date desc").Find(&expenses, "user_id = ?", userId).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, expenses)
}

func GetExpense(c *gin.Context) {
	userId, err := token.ExtractTokenID(c)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	id := c.Param("id")
	var expense models.Expense

	// First check if expense exists
	if err := models.DB.First(&expense, id).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "expense not found or access denied"})
		return
	}

	// Check if it belongs to the user
	if expense.UserID != userId {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only view your own expenses"})
		return
	}

	c.JSON(http.StatusOK, expense)
}

type SaveExpenseInput struct {
	Amount   float64 `json:"amount" binding:"required"`
	Date     string  `json:"date" binding:"required"`
	Category uint    `json:"category" binding:"required"`
	Comment  string  `json:"comment"`
}

func SaveExpense(c *gin.Context) {

	userId, err := token.ExtractTokenID(c)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var input SaveExpenseInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate date format
	parsedDate, err := time.Parse("2006-01-02", input.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
		return
	}

	// Check if Category belong to user
	var category models.Category
	if err := models.DB.First(&category, input.Category).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid category"})
		return
	}
	if category.UserID != userId {
		c.JSON(http.StatusForbidden, gin.H{"error": "category belong to another user"})
		return
	}

	newExpense := models.Expense{
		UserID:   userId,
		Amount:   input.Amount,
		Category: input.Category,
		Comment:  input.Comment,
		Date:     models.Date(parsedDate),
	}

	if err := models.DB.Create(&newExpense).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, newExpense)
}

type UpdateExpenseInput struct {
	Amount   float64 `json:"amount"`
	Date     string  `json:"date"`
	Category uint    `json:"category"`
	Comment  string  `json:"comment"`
}

func UpdateExpense(c *gin.Context) {

	userId, err := token.ExtractTokenID(c)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var input UpdateExpenseInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := c.Param("id")

	var expense models.Expense

	// First check if expense exists
	if err := models.DB.First(&expense, id).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "expense not found or access denied"})
		return
	}

	// Check if it belongs to the user
	if expense.UserID != userId {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only update your own expenses"})
		return
	}

	// Validate date format if provided
	var parsedDate *time.Time
	if input.Date != "" {
		d, err := time.Parse("2006-01-02", input.Date)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
			return
		}
		parsedDate = &d
	}

	// Check if Category belong to user
	if input.Category != 0 {
		var category models.Category
		if err := models.DB.First(&category, input.Category).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "invalid category"})
			return
		}
		if category.UserID != userId {
			c.JSON(http.StatusForbidden, gin.H{"error": "category belong to another user"})
			return
		}
	}

	// Prepare updates
	updates := map[string]interface{}{}
	if input.Amount != 0 {
		updates["amount"] = input.Amount
	}
	if parsedDate != nil {
		updates["date"] = models.Date(*parsedDate)
	}
	if input.Category != 0 {
		updates["category"] = input.Category
	}
	if input.Comment != "" || input.Comment == "" { // allow empty comment
		updates["comment"] = input.Comment
	}

	if len(updates) > 0 {
		if err := models.DB.Model(&expense).Updates(updates).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, expense)
}

func DeleteExpense(c *gin.Context) {

	userId, err := token.ExtractTokenID(c)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	id := c.Param("id")

	var expense models.Expense

	if err := models.DB.First(&expense, id).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if expense.UserID != userId {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only delete your own expenses"})
		return
	}

	if err := models.DB.Delete(&expense).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
