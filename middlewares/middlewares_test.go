package middlewares

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"moneytrack-api/utils/token"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestCORSMiddleware_SetsHeaders(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx.Request = req

	CORSMiddleware()(ctx)

	if ctx.IsAborted() {
		t.Fatal("expected request to continue for GET")
	}

	headers := recorder.Header()
	if headers.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("expected Access-Control-Allow-Origin header to be set")
	}
	if headers.Get("Access-Control-Allow-Methods") != "POST, OPTIONS, GET, PUT, DELETE" {
		t.Fatal("unexpected Access-Control-Allow-Methods header")
	}
}

func TestCORSMiddleware_OptionsAborts(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	ctx.Request = req

	CORSMiddleware()(ctx)

	if !ctx.IsAborted() {
		t.Fatal("expected request to abort for OPTIONS")
	}
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", recorder.Code)
	}
}

func TestJwtAuthMiddleware_AllowsValidToken(t *testing.T) {
	os.Setenv("TOKEN_HOUR_LIFESPAN", "24")
	os.Setenv("API_SECRET", "middleware-secret")
	defer os.Unsetenv("TOKEN_HOUR_LIFESPAN")
	defer os.Unsetenv("API_SECRET")

	tokenString, err := token.GenerateToken(1)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	ctx.Request = req

	JwtAuthMiddleware()(ctx)

	if ctx.IsAborted() {
		t.Fatal("expected valid token to allow request")
	}
}

func TestJwtAuthMiddleware_RejectsInvalidToken(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	ctx.Request = req

	JwtAuthMiddleware()(ctx)

	if !ctx.IsAborted() {
		t.Fatal("expected invalid token to abort request")
	}
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", recorder.Code)
	}
}
