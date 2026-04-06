package controllers

import (
	"moneytrack-api/models"
	"moneytrack-api/utils/token"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
)

func GetCategories(c *gin.Context) {
	userId, err := token.ExtractTokenID(c)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var categories []models.Category

	if err := models.DB.Find(&categories, "user_id = ?", userId).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, categories)
}

func GetCategory(c *gin.Context) {
	userId, err := token.ExtractTokenID(c)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	id := c.Param("id")
	var category models.Category

	// First check if category exists
	if err := models.DB.First(&category, id).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "category not found or access denied"})
		return
	}

	// Check if it belongs to the user
	if category.UserID != userId {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only view your own categories"})
		return
	}

	c.JSON(http.StatusOK, category)
}

type SaveCategoryInput struct {
	Category string `json:"category" binding:"required"`
}

func SaveCategory(c *gin.Context) {

	userId, err := token.ExtractTokenID(c)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var input SaveCategoryInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if category with same name for the user already exists
	var count int64
	models.DB.Model(models.Category{}).Where("user_id = ? AND category = ?", userId, input.Category).Count(&count)
	if count > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "category with same name already exist"})
		return
	}

	newCategory := models.Category{}

	copier.Copy(&newCategory, &input)
	newCategory.UserID = userId
	newCategory.Count = 0

	if err := models.DB.Create(&newCategory).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, newCategory)
}

type UpdateCategoryInput struct {
	Category string `json:"category"`
	Count    uint   `json:"count"`
}

func UpdateCategory(c *gin.Context) {

	userId, err := token.ExtractTokenID(c)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var input UpdateCategoryInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := c.Param("id")

	var category models.Category

	// First check if category exists
	if err := models.DB.First(&category, id).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "category not found or access denied"})
		return
	}

	// Check if it belongs to the user
	if category.UserID != userId {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only update your own categories"})
		return
	}

	// Check if category with same name already exists (excluding current category)
	var count int64
	models.DB.Model(models.Category{}).Where("category = ? AND id != ?", input.Category, id).Count(&count)
	if count > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "category with same name already exist"})
		return
	}

	if err := models.DB.Model(&category).Updates(input).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, category)
}

func DeleteCategory(c *gin.Context) {

	userId, err := token.ExtractTokenID(c)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	id := c.Param("id")

	var category models.Category

	if err := models.DB.First(&category, id).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if category.UserID != userId {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only delete your own categories"})
		return
	}

	// Check if category is connected with expenses
	var count int64
	models.DB.Model(models.Expense{}).Where("user_id = ? AND category = ?", userId, id).Count(&count)
	if count > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "category is connected with one or more expense and can't be deleted"})
		return
	}

	if err := models.DB.Delete(&category).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
