package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"moneytrack-api/models"
	"moneytrack-api/utils/token"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func findRepoRoot() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("unable to get caller information")
	}

	cwd := filepath.Dir(filename)
	for {
		if _, err := os.Stat(filepath.Join(cwd, "go.mod")); err == nil {
			return cwd, nil
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			return "", fmt.Errorf("could not find go.mod in parent directories")
		}
		cwd = parent
	}
}

func setupTestDB(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	for _, filename := range []string{".env", ".env.local"} {
		path := filepath.Join(repoRoot, filename)
		if _, err := os.Stat(path); err == nil {
			_ = godotenv.Load(path)
		}
	}

	setTokenEnv(t)
	if os.Getenv("DB_DRIVER") == "" {
		if err := os.Setenv("DB_DRIVER", "mysql"); err != nil {
			t.Fatal(err)
		}
	}
	if os.Getenv("DB_HOST") == "" {
		if err := os.Setenv("DB_HOST", "127.0.0.1"); err != nil {
			t.Fatal(err)
		}
	}
	if os.Getenv("DB_PORT") == "" {
		if err := os.Setenv("DB_PORT", "3306"); err != nil {
			t.Fatal(err)
		}
	}
	if os.Getenv("DB_USER") == "" || os.Getenv("DB_PASSWORD") == "" {
		t.Skip("MySQL tests skipped: set DB_USER and DB_PASSWORD environment variables")
	}
	if err := os.Setenv("DB_NAME", "moneytrack_api_test"); err != nil {
		t.Fatal(err)
	}

	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	rootDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local", user, password, host, port)
	sqlDB, err := sql.Open("mysql", rootDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	if _, err := sqlDB.Exec("DROP DATABASE IF EXISTS `" + dbName + "`"); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec("CREATE DATABASE `" + dbName + "`"); err != nil {
		t.Fatal(err)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, password, host, port, dbName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(&models.User{}, &models.Category{}, &models.Expense{}, &models.Income{}); err != nil {
		t.Fatal(err)
	}

	models.DB = db
}

func testContext(method, path, body string, headers map[string]string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	ctx.Request = req
	return ctx, recorder
}

func createTestUser(t *testing.T, email, password string) models.User {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	user := models.User{Email: email, Password: string(hash), FirstName: "Test", LastName: "User"}
	if err := models.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	return user
}

func createTestCategory(t *testing.T, userID uint, name string) models.Category {
	category := models.Category{UserID: userID, Category: name, Count: 0}
	if err := models.DB.Create(&category).Error; err != nil {
		t.Fatal(err)
	}
	return category
}

func createTestExpense(t *testing.T, userID uint, category uint, amount float64, date string) models.Expense {
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		t.Fatal(err)
	}
	expense := models.Expense{UserID: userID, Category: category, Amount: amount, Date: models.Date(parsedDate), Comment: "test expense"}
	if err := models.DB.Create(&expense).Error; err != nil {
		t.Fatal(err)
	}
	return expense
}

func createTestIncome(t *testing.T, userID uint, amount float64, date string) models.Income {
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		t.Fatal(err)
	}
	income := models.Income{UserID: userID, Amount: amount, Date: parsedDate.Format("2006-01-02"), Comment: "test income"}
	if err := models.DB.Create(&income).Error; err != nil {
		t.Fatal(err)
	}
	return income
}

func setTokenEnv(t *testing.T) {
	if err := os.Setenv("TOKEN_HOUR_LIFESPAN", "24"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("API_SECRET", "test-secret"); err != nil {
		t.Fatal(err)
	}
}

func authHeaderForUser(t *testing.T, userID uint) string {
	setTokenEnv(t)
	tokenString, err := token.GenerateToken(userID)
	if err != nil {
		t.Fatal(err)
	}
	return "Bearer " + tokenString
}

func TestVerifyPassword(t *testing.T) {
	password := "secret123"
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}

	if err := VerifyPassword(password, string(hashed)); err != nil {
		t.Fatalf("expected password to verify, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	setupTestDB(t)
	setTokenEnv(t)
	createTestUser(t, "login@example.com", "password123")

	ctx, recorder := testContext(http.MethodPost, "/login", "email=login@example.com&password=password123", map[string]string{"Content-Type": "application/x-www-form-urlencoded"})

	Login(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["token"] == "" {
		t.Fatal("expected token in response")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	setupTestDB(t)
	createTestUser(t, "login2@example.com", "password123")

	ctx, recorder := testContext(http.MethodPost, "/login", "email=login2@example.com&password=wrongpass", map[string]string{"Content-Type": "application/x-www-form-urlencoded"})

	Login(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}

func TestCurrentUser_ReturnsUser(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "current@example.com", "password123")
	header := authHeaderForUser(t, user.ID)

	ctx, recorder := testContext(http.MethodGet, "/current", "", map[string]string{"Authorization": header})

	CurrentUser(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var returned models.User
	if err := json.Unmarshal(recorder.Body.Bytes(), &returned); err != nil {
		t.Fatal(err)
	}
	if returned.Email != user.Email {
		t.Fatalf("expected email %s, got %s", user.Email, returned.Email)
	}
}

func TestSaveCategoryAndGetCategory(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "category@example.com", "password123")
	header := authHeaderForUser(t, user.ID)

	payload := `{"category":"Food"}`
	ctx, recorder := testContext(http.MethodPost, "/categories", payload, map[string]string{"Content-Type": "application/json", "Authorization": header})
	SaveCategory(ctx)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", recorder.Code)
	}

	var category models.Category
	if err := json.Unmarshal(recorder.Body.Bytes(), &category); err != nil {
		t.Fatal(err)
	}
	if category.Category != "Food" {
		t.Fatalf("expected category name Food, got %s", category.Category)
	}

	ctx, recorder = testContext(http.MethodGet, "/categories/1", "", map[string]string{"Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(category.ID))}}
	GetCategory(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestSaveCategory_Duplicate(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "dup@example.com", "password123")
	header := authHeaderForUser(t, user.ID)

	createTestCategory(t, user.ID, "Health")
	ctx, recorder := testContext(http.MethodPost, "/categories", `{"category":"Health"}`, map[string]string{"Content-Type": "application/json", "Authorization": header})
	SaveCategory(ctx)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", recorder.Code)
	}
}

func TestUpdateCategory(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "updatecat@example.com", "password123")
	category := createTestCategory(t, user.ID, "Books")
	header := authHeaderForUser(t, user.ID)

	ctx, recorder := testContext(http.MethodPut, "/categories/1", `{"category":"Books Updated","count":10}`, map[string]string{"Content-Type": "application/json", "Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(category.ID))}}
	UpdateCategory(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestDeleteCategory(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "deletecat@example.com", "password123")
	category := createTestCategory(t, user.ID, "Travel")
	header := authHeaderForUser(t, user.ID)

	ctx, recorder := testContext(http.MethodDelete, "/categories/1", "", map[string]string{"Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(category.ID))}}
	DeleteCategory(ctx)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", recorder.Code)
	}
}

func TestSaveExpenseAndDeleteExpense(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "expense@example.com", "password123")
	category := createTestCategory(t, user.ID, "Food")
	header := authHeaderForUser(t, user.ID)

	payload := `{"amount":42.25,"date":"2025-01-01","category":` + strconv.Itoa(int(category.ID)) + `,"comment":"sample"}`
	ctx, recorder := testContext(http.MethodPost, "/expenses", payload, map[string]string{"Content-Type": "application/json", "Authorization": header})
	SaveExpense(ctx)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", recorder.Code)
	}

	var expense models.Expense
	if err := json.Unmarshal(recorder.Body.Bytes(), &expense); err != nil {
		t.Fatal(err)
	}

	ctx, recorder = testContext(http.MethodDelete, "/expenses/1", "", map[string]string{"Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(expense.ID))}}
	DeleteExpense(ctx)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", recorder.Code)
	}
}

func TestSaveIncomeAndGetIncome(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "income@example.com", "password123")
	header := authHeaderForUser(t, user.ID)

	payload := `{"amount":120.50,"date":"2025-03-01","comment":"salary"}`
	ctx, recorder := testContext(http.MethodPost, "/incomes", payload, map[string]string{"Content-Type": "application/json", "Authorization": header})
	SaveIncome(ctx)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", recorder.Code)
	}

	var income models.Income
	if err := json.Unmarshal(recorder.Body.Bytes(), &income); err != nil {
		t.Fatal(err)
	}

	ctx, recorder = testContext(http.MethodGet, "/incomes/1", "", map[string]string{"Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(income.ID))}}
	GetIncome(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestGetExpenses_WithDateFilter(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "filter@example.com", "password123")
	category := createTestCategory(t, user.ID, "Travel")
	header := authHeaderForUser(t, user.ID)

	createTestExpense(t, user.ID, category.ID, 50.0, "2024-01-01")
	createTestExpense(t, user.ID, category.ID, 80.0, "2025-02-01")

	ctx, recorder := testContext(http.MethodGet, "/expenses?after_date=2025-01-01", "", map[string]string{"Authorization": header})
	GetExpenses(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var expenses []models.Expense
	if err := json.Unmarshal(recorder.Body.Bytes(), &expenses); err != nil {
		t.Fatal(err)
	}

	if len(expenses) != 1 {
		t.Fatalf("expected 1 expense after date filter, got %d", len(expenses))
	}
}

func authHeaderForExpiredUser(t *testing.T, userID uint) string {
	os.Setenv("TOKEN_HOUR_LIFESPAN", "-1") // Expired
	os.Setenv("API_SECRET", "test-secret")
	defer func() {
		os.Setenv("TOKEN_HOUR_LIFESPAN", "24")
	}()
	tokenString, err := token.GenerateToken(userID)
	if err != nil {
		t.Fatal(err)
	}
	return "Bearer " + tokenString
}

func createAnotherUser(t *testing.T) models.User {
	return createTestUser(t, "other@example.com", "password456")
}

func TestLogin_MissingRequiredParameter(t *testing.T) {
	setupTestDB(t)
	createTestUser(t, "missing@example.com", "password123")

	// Missing password
	ctx, recorder := testContext(http.MethodPost, "/login", "email=missing@example.com", map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	Login(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}

	// Missing email
	ctx, recorder = testContext(http.MethodPost, "/login", "password=password123", map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	Login(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestLogin_WrongEmail(t *testing.T) {
	setupTestDB(t)

	ctx, recorder := testContext(http.MethodPost, "/login", "email=wrong@example.com&password=password123", map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	Login(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestCurrentUser_ExpiredJWT(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "expired@example.com", "password123")
	header := authHeaderForExpiredUser(t, user.ID)

	ctx, recorder := testContext(http.MethodGet, "/current", "", map[string]string{"Authorization": header})
	CurrentUser(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestCurrentUser_NoJWT(t *testing.T) {
	setupTestDB(t)

	ctx, recorder := testContext(http.MethodGet, "/current", "", map[string]string{})
	CurrentUser(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestSaveCategory_UnicodeName(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "unicode@example.com", "password123")
	header := authHeaderForUser(t, user.ID)

	payload := `{"category":"🧳Unicode"}`
	ctx, recorder := testContext(http.MethodPost, "/categories", payload, map[string]string{"Content-Type": "application/json", "Authorization": header})
	SaveCategory(ctx)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", recorder.Code)
	}
}

func TestSaveCategory_NameExistsForAnotherUser(t *testing.T) {
	setupTestDB(t)
	user1 := createTestUser(t, "user1@example.com", "password123")
	user2 := createAnotherUser(t)
	header := authHeaderForUser(t, user2.ID)

	createTestCategory(t, user1.ID, "SharedName")

	ctx, recorder := testContext(http.MethodPost, "/categories", `{"category":"SharedName"}`, map[string]string{"Content-Type": "application/json", "Authorization": header})
	SaveCategory(ctx)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", recorder.Code)
	}
}

func TestSaveCategory_MissingRequiredParameter(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "missingcat@example.com", "password123")
	header := authHeaderForUser(t, user.ID)

	ctx, recorder := testContext(http.MethodPost, "/categories", `{}`, map[string]string{"Content-Type": "application/json", "Authorization": header})
	SaveCategory(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestGetCategories_ExpiredJWT(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "expiredcat@example.com", "password123")
	header := authHeaderForExpiredUser(t, user.ID)

	ctx, recorder := testContext(http.MethodGet, "/categories", "", map[string]string{"Authorization": header})
	GetCategories(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestGetCategories_NoJWT(t *testing.T) {
	setupTestDB(t)

	ctx, recorder := testContext(http.MethodGet, "/categories", "", map[string]string{})
	GetCategories(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestGetCategory_BelongsToAnotherUser(t *testing.T) {
	setupTestDB(t)
	user1 := createTestUser(t, "user1cat@example.com", "password123")
	user2 := createAnotherUser(t)
	category := createTestCategory(t, user1.ID, "Private")
	header := authHeaderForUser(t, user2.ID)

	ctx, recorder := testContext(http.MethodGet, "/categories/1", "", map[string]string{"Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(category.ID))}}
	GetCategory(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestGetCategory_NotExist(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "notexistcat@example.com", "password123")
	header := authHeaderForUser(t, user.ID)

	ctx, recorder := testContext(http.MethodGet, "/categories/999", "", map[string]string{"Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: "999"}}
	GetCategory(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestUpdateCategory_NameAlreadyExists(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "updatecatdup@example.com", "password123")
	category1 := createTestCategory(t, user.ID, "First")
	createTestCategory(t, user.ID, "Second")
	header := authHeaderForUser(t, user.ID)

	ctx, recorder := testContext(http.MethodPut, "/categories/1", `{"category":"Second"}`, map[string]string{"Content-Type": "application/json", "Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(category1.ID))}}
	UpdateCategory(ctx)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", recorder.Code)
	}
}

func TestUpdateCategory_NameExistsForAnotherUser(t *testing.T) {
	setupTestDB(t)
	user1 := createTestUser(t, "user1update@example.com", "password123")
	user2 := createAnotherUser(t)
	category := createTestCategory(t, user1.ID, "MyCat")
	createTestCategory(t, user2.ID, "OtherCat")
	header := authHeaderForUser(t, user1.ID)

	ctx, recorder := testContext(http.MethodPut, "/categories/1", `{"category":"OtherCat"}`, map[string]string{"Content-Type": "application/json", "Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(category.ID))}}
	UpdateCategory(ctx)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", recorder.Code)
	}
}

func TestUpdateCategory_BelongsToOtherUser(t *testing.T) {
	setupTestDB(t)
	user1 := createTestUser(t, "user1updateother@example.com", "password123")
	user2 := createAnotherUser(t)
	category := createTestCategory(t, user1.ID, "PrivateCat")
	header := authHeaderForUser(t, user2.ID)

	ctx, recorder := testContext(http.MethodPut, "/categories/1", `{"category":"Updated"}`, map[string]string{"Content-Type": "application/json", "Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(category.ID))}}
	UpdateCategory(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestUpdateCategory_NotExist(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "updatenotexist@example.com", "password123")
	header := authHeaderForUser(t, user.ID)

	ctx, recorder := testContext(http.MethodPut, "/categories/999", `{"category":"Updated"}`, map[string]string{"Content-Type": "application/json", "Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: "999"}}
	UpdateCategory(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestDeleteCategory_ConnectedWithExpense(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "deleteconnected@example.com", "password123")
	category := createTestCategory(t, user.ID, "Connected")
	createTestExpense(t, user.ID, category.ID, 10.0, "2025-01-01")
	header := authHeaderForUser(t, user.ID)

	ctx, recorder := testContext(http.MethodDelete, "/categories/1", "", map[string]string{"Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(category.ID))}}
	DeleteCategory(ctx)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", recorder.Code)
	}
}

func TestDeleteCategory_NotExist(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "deletenotexist@example.com", "password123")
	header := authHeaderForUser(t, user.ID)

	ctx, recorder := testContext(http.MethodDelete, "/categories/999", "", map[string]string{"Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: "999"}}
	DeleteCategory(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestDeleteCategory_BelongsToOtherUser(t *testing.T) {
	setupTestDB(t)
	user1 := createTestUser(t, "user1deleteother@example.com", "password123")
	user2 := createAnotherUser(t)
	category := createTestCategory(t, user1.ID, "PrivateDelete")
	header := authHeaderForUser(t, user2.ID)

	ctx, recorder := testContext(http.MethodDelete, "/categories/1", "", map[string]string{"Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(category.ID))}}
	DeleteCategory(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestSaveIncome_MissingRequiredParameter(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "missingincome@example.com", "password123")
	header := authHeaderForUser(t, user.ID)

	ctx, recorder := testContext(http.MethodPost, "/incomes", `{"amount":100}`, map[string]string{"Content-Type": "application/json", "Authorization": header})
	SaveIncome(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestSaveIncome_InvalidDate(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "invaliddateincome@example.com", "password123")
	header := authHeaderForUser(t, user.ID)

	payload := `{"amount":100,"date":"2025-13-01","comment":"invalid"}`
	ctx, recorder := testContext(http.MethodPost, "/incomes", payload, map[string]string{"Content-Type": "application/json", "Authorization": header})
	SaveIncome(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestSaveIncome_InvalidDateFormat(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "invalidformatincome@example.com", "password123")
	header := authHeaderForUser(t, user.ID)

	payload := `{"amount":100,"date":"31-05-2025","comment":"invalid"}`
	ctx, recorder := testContext(http.MethodPost, "/incomes", payload, map[string]string{"Content-Type": "application/json", "Authorization": header})
	SaveIncome(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestSaveIncome_InvalidAmount(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "invalidamountincome@example.com", "password123")
	header := authHeaderForUser(t, user.ID)

	payload := `{"amount":"notnumber","date":"2025-01-01","comment":"invalid"}`
	ctx, recorder := testContext(http.MethodPost, "/incomes", payload, map[string]string{"Content-Type": "application/json", "Authorization": header})
	SaveIncome(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestGetIncomes_ExpiredJWT(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "expiredincome@example.com", "password123")
	header := authHeaderForExpiredUser(t, user.ID)

	ctx, recorder := testContext(http.MethodGet, "/incomes", "", map[string]string{"Authorization": header})
	GetIncomes(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestGetIncomes_NoJWT(t *testing.T) {
	setupTestDB(t)

	ctx, recorder := testContext(http.MethodGet, "/incomes", "", map[string]string{})
	GetIncomes(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestGetIncome_BelongsToAnotherUser(t *testing.T) {
	setupTestDB(t)
	user1 := createTestUser(t, "user1income@example.com", "password123")
	user2 := createAnotherUser(t)
	income := createTestIncome(t, user1.ID, 100.0, "2025-01-01")
	header := authHeaderForUser(t, user2.ID)

	ctx, recorder := testContext(http.MethodGet, "/incomes/1", "", map[string]string{"Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(income.ID))}}
	GetIncome(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestGetIncome_NotExist(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "notexistincome@example.com", "password123")
	header := authHeaderForUser(t, user.ID)

	ctx, recorder := testContext(http.MethodGet, "/incomes/999", "", map[string]string{"Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: "999"}}
	GetIncome(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestUpdateIncome_BelongsToOtherUser(t *testing.T) {
	setupTestDB(t)
	user1 := createTestUser(t, "user1updateincome@example.com", "password123")
	user2 := createAnotherUser(t)
	income := createTestIncome(t, user1.ID, 100.0, "2025-01-01")
	header := authHeaderForUser(t, user2.ID)

	ctx, recorder := testContext(http.MethodPut, "/incomes/1", `{"amount":200}`, map[string]string{"Content-Type": "application/json", "Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(income.ID))}}
	UpdateIncome(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestUpdateIncome_NotExist(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "updatenotexistincome@example.com", "password123")
	header := authHeaderForUser(t, user.ID)

	ctx, recorder := testContext(http.MethodPut, "/incomes/999", `{"amount":200}`, map[string]string{"Content-Type": "application/json", "Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: "999"}}
	UpdateIncome(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestUpdateIncome_InvalidDate(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "updateinvaliddateincome@example.com", "password123")
	income := createTestIncome(t, user.ID, 100.0, "2025-01-01")
	header := authHeaderForUser(t, user.ID)

	ctx, recorder := testContext(http.MethodPut, "/incomes/1", `{"date":"2025-13-01"}`, map[string]string{"Content-Type": "application/json", "Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(income.ID))}}
	UpdateIncome(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestUpdateIncome_InvalidDateFormat(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "updateinvalidformatincome@example.com", "password123")
	income := createTestIncome(t, user.ID, 100.0, "2025-01-01")
	header := authHeaderForUser(t, user.ID)

	ctx, recorder := testContext(http.MethodPut, "/incomes/1", `{"date":"31-05-2025"}`, map[string]string{"Content-Type": "application/json", "Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(income.ID))}}
	UpdateIncome(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestUpdateIncome_InvalidAmount(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "updateinvalidamountincome@example.com", "password123")
	income := createTestIncome(t, user.ID, 100.0, "2025-01-01")
	header := authHeaderForUser(t, user.ID)

	ctx, recorder := testContext(http.MethodPut, "/incomes/1", `{"amount":"notnumber"}`, map[string]string{"Content-Type": "application/json", "Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(income.ID))}}
	UpdateIncome(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestDeleteIncome_NotExist(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "deletenotexistincome@example.com", "password123")
	header := authHeaderForUser(t, user.ID)

	ctx, recorder := testContext(http.MethodDelete, "/incomes/999", "", map[string]string{"Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: "999"}}
	DeleteIncome(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestDeleteIncome_BelongsToOtherUser(t *testing.T) {
	setupTestDB(t)
	user1 := createTestUser(t, "user1deleteincome@example.com", "password123")
	user2 := createAnotherUser(t)
	income := createTestIncome(t, user1.ID, 100.0, "2025-01-01")
	header := authHeaderForUser(t, user2.ID)

	ctx, recorder := testContext(http.MethodDelete, "/incomes/1", "", map[string]string{"Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(income.ID))}}
	DeleteIncome(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestSaveExpense_MissingRequiredParameter(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "missingexpense@example.com", "password123")
	header := authHeaderForUser(t, user.ID)

	ctx, recorder := testContext(http.MethodPost, "/expenses", `{"amount":50}`, map[string]string{"Content-Type": "application/json", "Authorization": header})
	SaveExpense(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestSaveExpense_InvalidDate(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "invaliddateexpense@example.com", "password123")
	category := createTestCategory(t, user.ID, "TestCat")
	header := authHeaderForUser(t, user.ID)

	payload := fmt.Sprintf(`{"amount":50,"date":"2025-13-01","category":%d}`, category.ID)
	ctx, recorder := testContext(http.MethodPost, "/expenses", payload, map[string]string{"Content-Type": "application/json", "Authorization": header})
	SaveExpense(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestSaveExpense_InvalidDateFormat(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "invalidformatexpense@example.com", "password123")
	category := createTestCategory(t, user.ID, "TestCat")
	header := authHeaderForUser(t, user.ID)

	payload := fmt.Sprintf(`{"amount":50,"date":"31-05-2025","category":%d}`, category.ID)
	ctx, recorder := testContext(http.MethodPost, "/expenses", payload, map[string]string{"Content-Type": "application/json", "Authorization": header})
	SaveExpense(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestSaveExpense_InvalidAmount(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "invalidamountexpense@example.com", "password123")
	category := createTestCategory(t, user.ID, "TestCat")
	header := authHeaderForUser(t, user.ID)

	payload := fmt.Sprintf(`{"amount":"notnumber","date":"2025-01-01","category":%d}`, category.ID)
	ctx, recorder := testContext(http.MethodPost, "/expenses", payload, map[string]string{"Content-Type": "application/json", "Authorization": header})
	SaveExpense(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestSaveExpense_InvalidCategory(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "invalidcategoryexpense@example.com", "password123")
	header := authHeaderForUser(t, user.ID)

	payload := `{"amount":50,"date":"2025-01-01","category":999}`
	ctx, recorder := testContext(http.MethodPost, "/expenses", payload, map[string]string{"Content-Type": "application/json", "Authorization": header})
	SaveExpense(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestSaveExpense_CategoryBelongsToOtherUser(t *testing.T) {
	setupTestDB(t)
	user1 := createTestUser(t, "user1expense@example.com", "password123")
	user2 := createAnotherUser(t)
	category := createTestCategory(t, user1.ID, "PrivateCat")
	header := authHeaderForUser(t, user2.ID)

	payload := fmt.Sprintf(`{"amount":50,"date":"2025-01-01","category":%d}`, category.ID)
	ctx, recorder := testContext(http.MethodPost, "/expenses", payload, map[string]string{"Content-Type": "application/json", "Authorization": header})
	SaveExpense(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestGetExpenses_ExpiredJWT(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "expiredexpense@example.com", "password123")
	header := authHeaderForExpiredUser(t, user.ID)

	ctx, recorder := testContext(http.MethodGet, "/expenses", "", map[string]string{"Authorization": header})
	GetExpenses(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestGetExpenses_NoJWT(t *testing.T) {
	setupTestDB(t)

	ctx, recorder := testContext(http.MethodGet, "/expenses", "", map[string]string{})
	GetExpenses(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestGetExpense_BelongsToAnotherUser(t *testing.T) {
	setupTestDB(t)
	user1 := createTestUser(t, "user1expenseget@example.com", "password123")
	user2 := createAnotherUser(t)
	category := createTestCategory(t, user1.ID, "TestCat")
	expense := createTestExpense(t, user1.ID, category.ID, 50.0, "2025-01-01")
	header := authHeaderForUser(t, user2.ID)

	ctx, recorder := testContext(http.MethodGet, "/expenses/1", "", map[string]string{"Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(expense.ID))}}
	GetExpense(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestGetExpense_NotExist(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "notexistexpense@example.com", "password123")
	header := authHeaderForUser(t, user.ID)

	ctx, recorder := testContext(http.MethodGet, "/expenses/999", "", map[string]string{"Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: "999"}}
	GetExpense(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestUpdateExpense_BelongsToOtherUser(t *testing.T) {
	setupTestDB(t)
	user1 := createTestUser(t, "user1updateexpense@example.com", "password123")
	user2 := createAnotherUser(t)
	category := createTestCategory(t, user1.ID, "TestCat")
	expense := createTestExpense(t, user1.ID, category.ID, 50.0, "2025-01-01")
	header := authHeaderForUser(t, user2.ID)

	ctx, recorder := testContext(http.MethodPut, "/expenses/1", `{"amount":100}`, map[string]string{"Content-Type": "application/json", "Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(expense.ID))}}
	UpdateExpense(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestUpdateExpense_NotExist(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "updatenotexistexpense@example.com", "password123")
	header := authHeaderForUser(t, user.ID)

	ctx, recorder := testContext(http.MethodPut, "/expenses/999", `{"amount":100}`, map[string]string{"Content-Type": "application/json", "Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: "999"}}
	UpdateExpense(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestUpdateExpense_InvalidDate(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "updateinvaliddateexpense@example.com", "password123")
	category := createTestCategory(t, user.ID, "TestCat")
	expense := createTestExpense(t, user.ID, category.ID, 50.0, "2025-01-01")
	header := authHeaderForUser(t, user.ID)

	ctx, recorder := testContext(http.MethodPut, "/expenses/1", `{"date":"2025-13-01"}`, map[string]string{"Content-Type": "application/json", "Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(expense.ID))}}
	UpdateExpense(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestUpdateExpense_InvalidDateFormat(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "updateinvalidformatexpense@example.com", "password123")
	category := createTestCategory(t, user.ID, "TestCat")
	expense := createTestExpense(t, user.ID, category.ID, 50.0, "2025-01-01")
	header := authHeaderForUser(t, user.ID)

	ctx, recorder := testContext(http.MethodPut, "/expenses/1", `{"date":"31-05-2025"}`, map[string]string{"Content-Type": "application/json", "Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(expense.ID))}}
	UpdateExpense(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestUpdateExpense_InvalidAmount(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "updateinvalidamountexpense@example.com", "password123")
	category := createTestCategory(t, user.ID, "TestCat")
	expense := createTestExpense(t, user.ID, category.ID, 50.0, "2025-01-01")
	header := authHeaderForUser(t, user.ID)

	ctx, recorder := testContext(http.MethodPut, "/expenses/1", `{"amount":"notnumber"}`, map[string]string{"Content-Type": "application/json", "Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(expense.ID))}}
	UpdateExpense(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestUpdateExpense_InvalidCategory(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "updateinvalidcategoryexpense@example.com", "password123")
	category := createTestCategory(t, user.ID, "TestCat")
	expense := createTestExpense(t, user.ID, category.ID, 50.0, "2025-01-01")
	header := authHeaderForUser(t, user.ID)

	ctx, recorder := testContext(http.MethodPut, "/expenses/1", `{"category":999}`, map[string]string{"Content-Type": "application/json", "Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(expense.ID))}}
	UpdateExpense(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestUpdateExpense_CategoryBelongsToOtherUser(t *testing.T) {
	setupTestDB(t)
	user1 := createTestUser(t, "user1updateexpensecat@example.com", "password123")
	user2 := createAnotherUser(t)
	category1 := createTestCategory(t, user1.ID, "TestCat1")
	category2 := createTestCategory(t, user2.ID, "TestCat2")
	expense := createTestExpense(t, user1.ID, category1.ID, 50.0, "2025-01-01")
	header := authHeaderForUser(t, user1.ID)

	ctx, recorder := testContext(http.MethodPut, "/expenses/1", fmt.Sprintf(`{"category":%d}`, category2.ID), map[string]string{"Content-Type": "application/json", "Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(expense.ID))}}
	UpdateExpense(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestDeleteExpense_NotExist(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "deletenotexistexpense@example.com", "password123")
	header := authHeaderForUser(t, user.ID)

	ctx, recorder := testContext(http.MethodDelete, "/expenses/999", "", map[string]string{"Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: "999"}}
	DeleteExpense(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestDeleteExpense_BelongsToOtherUser(t *testing.T) {
	setupTestDB(t)
	user1 := createTestUser(t, "user1deleteexpense@example.com", "password123")
	user2 := createAnotherUser(t)
	category := createTestCategory(t, user1.ID, "TestCat")
	expense := createTestExpense(t, user1.ID, category.ID, 50.0, "2025-01-01")
	header := authHeaderForUser(t, user2.ID)

	ctx, recorder := testContext(http.MethodDelete, "/expenses/1", "", map[string]string{"Authorization": header})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(int(expense.ID))}}
	DeleteExpense(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestYears_ReturnsDistinctYears(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t, "years@example.com", "password123")
	category := createTestCategory(t, user.ID, "Bills")
	createTestExpense(t, user.ID, category.ID, 20.0, "2024-05-01")
	createTestExpense(t, user.ID, category.ID, 30.0, "2025-11-01")

	header := authHeaderForUser(t, user.ID)
	ctx, recorder := testContext(http.MethodGet, "/years", "", map[string]string{"Authorization": header})
	Years(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var years []int16
	if err := json.Unmarshal(recorder.Body.Bytes(), &years); err != nil {
		t.Fatal(err)
	}
	if len(years) != 2 {
		t.Fatalf("expected 2 distinct years, got %d", len(years))
	}
}
