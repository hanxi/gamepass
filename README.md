# GamePass - OAuth2 Authentication Service

GamePass 是一个基于 Go 语言实现的身份认证服务，支持 OAuth2 登录和 JWT 令牌管理。

## 特性

- OAuth2 认证支持（目前支持 GitHub）
- JWT 令牌管理（访问令牌和刷新令牌）
- MongoDB 持久化存储
- RESTful API
- 支持令牌刷新和撤销
- 用户会话管理

## 技术栈

- [Gin](https://github.com/gin-gonic/gin) - Web 框架
- [Goth](https://github.com/markbates/goth) - OAuth2 认证库
- [JWT](https://github.com/golang-jwt/jwt) - JWT 实现
- [MongoDB Go Driver](https://github.com/mongodb/mongo-go-driver) - MongoDB 驱动
- [GoDotEnv](https://github.com/joho/godotenv) - 环境变量管理

## 安装

1. 克隆仓库：

```bash
git clone https://github.com/yourusername/gamepass.git
cd gamepass
```

2. 安装依赖：

```bash
go mod tidy
```

3. 配置环境变量，创建 `.env` 文件：

```env
PORT=3180
BASE_URL=http://localhost:3180
MONGODB_URI=mongodb://localhost:27017
DB_NAME=authdb
GITHUB_KEY=your_github_key
GITHUB_SECRET=your_github_secret
JWT_ACCESS_SECRET=your_access_secret
JWT_REFRESH_SECRET=your_refresh_secret
```

4. 运行服务：

```bash
go run main.go
```

## API 端点

### 公开 API

- `GET /api/providers` - 获取可用的登录提供商列表

示例响应：

```json
{
    "providers": [
        {
            "name": "github",
            "auth_url": "/auth/github",
            "display_name": "Github"
        },
        {
            "name": "google",
            "auth_url": "/auth/google",
            "display_name": "Google"
        }
    ],
    "total": 2
}
```

### 认证相关

- `GET /auth/github` - 开始 GitHub OAuth 登录流程
- `GET /auth/github/callback` - GitHub OAuth 回调处理
- `POST /auth/refresh` - 刷新访问令牌
- `POST /auth/logout` - 注销会话

### 受保护的 API

- `GET /api/profile` - 获取用户资料（需要认证）

## API 使用示例

### OAuth2 登录

```bash
# 重定向到 GitHub 登录页面
curl -i -H "Accept: application/json" http://localhost:3180/auth/github
```

### 刷新令牌

```bash
curl -X POST \
  http://localhost:3180/auth/refresh \
  -H 'Refresh-Token: your_refresh_token'
```

### 获取用户资料

```bash
curl -X GET \
  http://localhost:3180/api/profile \
  -H 'Authorization: Bearer your_access_token'
```

### 注销

```bash
curl -X POST \
  http://localhost:3180/auth/logout \
  -H 'Authorization: Bearer your_access_token' \
  -H 'Refresh-Token: your_refresh_token'
```

## 项目结构

```
.
├── config/         # 配置相关
├── db/            # 数据库相关
├── handlers/      # HTTP 处理器
├── middlewares/   # 中间件
├── models/        # 数据模型
├── services/      # 业务逻辑
├── utils/         # 工具函数
├── .env           # 环境变量
├── main.go        # 程序入口
└── README.md      # 项目文档
```

## 开发

### 测试

```bash
go test ./...
```

### 添加新的 OAuth 提供商

在 `main.go` 中添加新的提供商。回调地址用 `handlers.ProviderCallbackURL` 生成，保证与路由 `/auth/:provider/callback` 以及 provider 的注册名一致：

```go
goth.UseProviders(
    github.New(cfg.GithubKey, cfg.GithubSecret, handlers.ProviderCallbackURL(cfg.BaseURL, "github")),
    // 添加其他提供商...
)
```

provider 名不能叫 `local`，`/auth/local` 留给本地账号密码登录。OpenID Connect provider 的注册名是 `local-oidc`（`handlers.OIDCProvider`），入口 `/auth/local-oidc`，回调 `/auth/local-oidc/callback`。

## 生产环境部署

1. 使用 HTTPS
2. 设置安全的环境变量
3. 配置 MongoDB 认证
4. 使用反向代理（如 Nginx）
5. 设置适当的日志级别

## 贡献指南

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

## 作者

- hanxi - 项目维护者

## 致谢

- 感谢所有贡献者
- 感谢使用到的开源项目
