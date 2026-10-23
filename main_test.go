package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"

	"github.com/hanxi/gamepass/handlers"
)

// 本地起一个最小的 IdP：只提供 discovery 文档，goth 构造 provider 时会去拉它
func fakeIdP(t *testing.T) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{
			"issuer":                 srv.URL,
			"authorization_endpoint": srv.URL + "/authorize",
			"token_endpoint":         srv.URL + "/token",
			"userinfo_endpoint":      srv.URL + "/userinfo",
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

// 注册名、登录入口与回调路径必须一致：IdP 回调到 /auth/<段>/callback 时，goth 按路径段
// 找 provider，段与注册名对不上就认证失败
func TestOIDCProviderNameMatchesCallbackRoute(t *testing.T) {
	idp := fakeIdP(t)
	p, err := newOIDCProvider("http://localhost:3180", "client", "secret", idp.URL+"/.well-known/openid-configuration")
	if err != nil {
		t.Fatalf("newOIDCProvider: %v", err)
	}
	if p.Name() == "local" {
		t.Fatal("OIDC provider must not be named local: the local branch would shadow it")
	}
	want := "http://localhost:3180/auth/" + p.Name() + "/callback"
	if p.CallbackURL != want {
		t.Fatalf("callback %q, want %q", p.CallbackURL, want)
	}
}

// 走真实的 handler：/auth/<OIDC 名> 必须跳到 IdP 的授权地址，且 redirect_uri 是回调路由
func TestOIDCLoginRedirectsToIdentityProvider(t *testing.T) {
	idp := fakeIdP(t)
	p, err := newOIDCProvider("http://localhost:3180", "client", "secret", idp.URL+"/.well-known/openid-configuration")
	if err != nil {
		t.Fatalf("newOIDCProvider: %v", err)
	}
	goth.UseProviders(p)
	t.Cleanup(goth.ClearProviders)
	gothic.Store = sessions.NewCookieStore([]byte("test-secret"))

	gin.SetMode(gin.TestMode)
	h := handlers.NewAuthHandler(nil, nil, nil, []*regexp.Regexp{regexp.MustCompile(`^http://localhost:[0-9]+/$`)})
	r := gin.New()
	r.GET("/auth/:provider", h.BeginAuth)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/auth/"+handlers.OIDCProvider+"?callback_url=http://localhost:3000/", nil))
	if w.Code != http.StatusTemporaryRedirect && w.Code != http.StatusFound {
		t.Fatalf("status %d, want redirect to IdP; body %s", w.Code, w.Body.String())
	}
	loc, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatalf("bad Location: %v", err)
	}
	if got := loc.Scheme + "://" + loc.Host + loc.Path; got != idp.URL+"/authorize" {
		t.Fatalf("redirected to %q, want the IdP authorize endpoint", got)
	}
	if got := loc.Query().Get("redirect_uri"); got != "http://localhost:3180/auth/"+handlers.OIDCProvider+"/callback" {
		t.Fatalf("redirect_uri %q", got)
	}
}
