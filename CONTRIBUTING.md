# 开发环境的配置

## 基础

- Golang
- NodeJS / NPM
- Caddy

## 开发与调试

要注意本项目前后端分离存放在不同的仓库中，在 clone 时注意加上 `--recurse-submodules`。

```
git clone --recurse-submodules git@github.com:Xinrea/JoiAsk.git
```

### 1. 启动后端

```
go run cmd/cmd.go
```

### 2. 启动前端 dev

```
cd frontend && npm run dev
```

### 3. 启动本地服务器

> [!NOTE]
> 由于未 Release 打包前，前后端完全独立，在本地需要进行一定的 Path Routing 设置才能够正常访问，因此推荐使用 Caddy 来启用一个本地服务器。

Caddyfile 已提供在项目根目录，只需 `caddy run` 即可启动本地服务器。

Caddyfile 的内容如下所示，十分简单，将 `/api/*` 指向后端 gin server；将其余请求指向前端 dev server。如果你更改了前后端默认端口，Caddyfile 中的内容也应同步更改。

```
localhost

handle /api/* {
    reverse_proxy /api/* :8080
}

handle * {
    reverse_proxy :5173
}
```

之后浏览器访问本地 localhost 即能够正常预览页面。
