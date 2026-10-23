package config

import (
	"testing"

	"github.com/joho/godotenv"
)

func TestParseCallbackWhitelistTrimsAndSkipsEmpty(t *testing.T) {
	list, err := ParseCallbackWhitelist(" ^a$ ,, ^b$ ,")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 || list[0].String() != "^a$" || list[1].String() != "^b$" {
		t.Fatalf("got %v, want [^a$ ^b$]", list)
	}
}

func TestParseCallbackWhitelistEmptyMeansNothingAllowed(t *testing.T) {
	list, err := ParseCallbackWhitelist("")
	if err != nil || len(list) != 0 {
		t.Fatalf("got %v, %v; want empty list", list, err)
	}
}

func TestParseCallbackWhitelistRejectsInvalidRegex(t *testing.T) {
	if _, err := ParseCallbackWhitelist("^ok$,(unclosed"); err == nil {
		t.Fatal("want error for invalid regex")
	}
}

// .env.example 是大多数人照抄的起点：默认白名单必须放行 skyext 示例客户端 phaser-game
// 的回调（登录页把当前页面地址 http://localhost:3000/ 当 callback_url），否则浏览器登录
// 第一步就被拒
func TestEnvExampleWhitelistAllowsPhaserGameCallback(t *testing.T) {
	env, err := godotenv.Read("../.env.example")
	if err != nil {
		t.Fatalf("read .env.example: %v", err)
	}
	list, err := ParseCallbackWhitelist(env["CALLBACK_WHITELIST"])
	if err != nil {
		t.Fatalf("parse CALLBACK_WHITELIST: %v", err)
	}
	matches := func(url string) bool {
		for _, re := range list {
			if re.MatchString(url) {
				return true
			}
		}
		return false
	}

	for _, url := range []string{
		"http://localhost:3000/",
		"http://localhost:8080/callback",
		"https://game.my-app.com/auth/callback",
		"my-awesome-game://auth",
	} {
		if !matches(url) {
			t.Errorf("%s should be allowed", url)
		}
	}
	for _, url := range []string{
		"http://evil.com/",
		"http://localhost.evil.com:3000/",
		"http://localhost:3000/elsewhere",
		"http://localhost:3000/?next=http://evil.com/",
	} {
		if matches(url) {
			t.Errorf("%s should be rejected", url)
		}
	}
}
