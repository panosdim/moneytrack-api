package token

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGenerateToken_ValidEnv(t *testing.T) {
	os.Setenv("TOKEN_HOUR_LIFESPAN", "24")
	os.Setenv("API_SECRET", "test-secret")
	defer os.Unsetenv("TOKEN_HOUR_LIFESPAN")
	defer os.Unsetenv("API_SECRET")

	tokenString, err := GenerateToken(42)
	if err != nil {
		t.Fatal(err)
	}
	if tokenString == "" {
		t.Fatal("expected token string")
	}
}

func TestExtractToken_FromQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?token=query-token", nil)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	got := ExtractToken(ctx)
	if got != "query-token" {
		t.Fatalf("expected query-token, got %q", got)
	}
}

func TestExtractToken_FromHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer header-token")
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	got := ExtractToken(ctx)
	if got != "header-token" {
		t.Fatalf("expected header-token, got %q", got)
	}
}

func TestValid_ReturnsNilForValidToken(t *testing.T) {
	os.Setenv("TOKEN_HOUR_LIFESPAN", "24")
	os.Setenv("API_SECRET", "test-secret")
	defer os.Unsetenv("TOKEN_HOUR_LIFESPAN")
	defer os.Unsetenv("API_SECRET")

	tokenString, err := GenerateToken(99)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	if err := Valid(ctx); err != nil {
		t.Fatalf("expected valid token, got error %v", err)
	}
}

func TestValid_ReturnsErrorForInvalidToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	if err := Valid(ctx); err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestExtractTokenID_ReturnsUserId(t *testing.T) {
	os.Setenv("TOKEN_HOUR_LIFESPAN", "24")
	os.Setenv("API_SECRET", "test-secret")
	defer os.Unsetenv("TOKEN_HOUR_LIFESPAN")
	defer os.Unsetenv("API_SECRET")

	tokenString, err := GenerateToken(123)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	id, err := ExtractTokenID(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if id != 123 {
		t.Fatalf("expected id 123, got %d", id)
	}
}

func TestExtractTokenID_ReturnsErrorForMissingToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	if _, err := ExtractTokenID(ctx); err == nil {
		t.Fatal("expected error when token is missing")
	}
}
