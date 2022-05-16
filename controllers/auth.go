package controllers

import (
	"moneytrack-api/models"
	"moneytrack-api/utils/token"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func CurrentUser(c *gin.Context) {

	userId, err := token.ExtractTokenID(c)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var u models.User

	if err := models.DB.First(&u, userId).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, u)
}

type LoginInput struct {
	Email    string `form:"email" binding:"required,email"`
	Password string `form:"password" binding:"required"`
}

func VerifyPassword(password, hashedPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

func Login(c *gin.Context) {

	var input LoginInput

	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u := models.User{}

	if err := models.DB.Model(models.User{}).Where("email = ?", input.Email).Take(&u).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email or password is incorrect"})
		return
	}

	if err := VerifyPassword(input.Password, u.Password); err != nil && err == bcrypt.ErrMismatchedHashAndPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email or password is incorrect"})
		return
	}

	jwt, err := token.GenerateToken(u.ID)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": jwt})

}
