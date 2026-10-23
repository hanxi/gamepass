package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

const allowedCallback = "http://localhost:3000/"
const rejectedCallback = "http://evil.com/"

// 这些用例都在碰到数据库之前返回，服务依赖传 nil 即可
func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewAuthHandler(nil, nil, nil, []*regexp.Regexp{regexp.MustCompile(`^http://localhost:[0-9]+/$`)})
	r := gin.New()
	auth := r.Group("/auth")
	auth.GET("/:provider", h.BeginAuth)
	auth.GET("/:provider/callback", h.Callback)
	auth.POST("/local/login", h.Login)
	auth.POST("/local/register", h.Register)
	return r
}

func serve(r *gin.Engine, method, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// 三个入口用的是同一个参数名 callback_url，被拒时的报错也必须一致
func TestRejectedCallbackErrorIsConsistentAcrossEntrypoints(t *testing.T) {
	r := newTestRouter()
	cases := map[string]*httptest.ResponseRecorder{
		"GET /auth/local": serve(r, "GET", "/auth/local?callback_url="+rejectedCallback, ""),
		"POST /auth/local/login": serve(r, "POST", "/auth/local/login",
			`{"email":"a@b.com","password":"x","callback_url":"`+rejectedCallback+`"}`),
		"POST /auth/local/register": serve(r, "POST", "/auth/local/register",
			`{"name":"a","email":"a@b.com","password":"x","callback_url":"`+rejectedCallback+`"}`),
	}
	for name, w := range cases {
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", name, w.Code)
			continue
		}
		var body map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Errorf("%s: bad json %q", name, w.Body.String())
			continue
		}
		if body["error"] != "Provided callback_url is not allowed" {
			t.Errorf("%s: error %q", name, body["error"])
		}
	}
}

func TestBeginAuthLocalStoresAllowedCallbackAndGoesToLoginPage(t *testing.T) {
	w := serve(newTestRouter(), "GET", "/auth/local?callback_url="+allowedCallback, "")
	if w.Code != http.StatusFound || w.Header().Get("Location") != "/login" {
		t.Fatalf("status %d location %q, want 302 /login", w.Code, w.Header().Get("Location"))
	}
	cookies := w.Header().Values("Set-Cookie")
	found := false
	for _, c := range cookies {
		if strings.HasPrefix(c, AuthCallbackURLCookie+"=") && strings.Contains(c, "localhost%3A3000") {
			found = true
		}
	}
	if !found {
		t.Fatalf("callback cookie not set: %v", cookies)
	}
}

// 注册给 IdP 的回调地址必须落到 /auth/:provider/callback，且 provider 段就是注册名，
// 否则 IdP 回调时 goth 按路径取到的 provider 名找不到对应 provider
func TestProviderCallbackURLRoutesBackToSameProvider(t *testing.T) {
	url := ProviderCallbackURL("http://localhost:3180", OIDCProvider)
	if url != "http://localhost:3180/auth/"+OIDCProvider+"/callback" {
		t.Fatalf("callback url %q", url)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	var got string
	r.GET("/auth/:provider/callback", func(c *gin.Context) { got = c.Param("provider") })
	serve(r, "GET", strings.TrimPrefix(url, "http://localhost:3180"), "")
	if got != OIDCProvider {
		t.Fatalf("route resolved provider %q, want %q", got, OIDCProvider)
	}
}
