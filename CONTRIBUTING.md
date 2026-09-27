# 开发环境的配置

## 基础

- Golang
- NodeJS / NPM
- Caddy

## 使用 Docker Compose（推荐）

项目提供了包含 Caddy、前端、管理后台和后端的开发环境：

```bash
docker compose -f docker-compose.dev.yml up --build
```

启动后访问：

- 前端：`http://localhost/`
- 管理后台：`http://localhost/admin/`
- 后端 API：`http://localhost/api/`

前端和管理后台使用 Next.js Fast Refresh，后端使用 Air 监听 Go 源文件并自动重新构建。源码目录以 bind mount 挂载，修改后无需重建镜像。默认监听宿主机的 80 端口；如果端口被占用，可运行：

```bash
DEV_PORT=8088 docker compose -f docker-compose.dev.yml up --build
```

此时通过 `http://localhost:8088/` 访问。依赖文件发生变化时，使用 `docker compose -f docker-compose.dev.yml up --build` 重新构建对应开发镜像。

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

### 3. 启动管理后台 dev

```
cd admin && npm run dev -- --port 3001
```

### 4. 启动本地服务器

> [!NOTE]
> 由于未 Release 打包前，前后端完全独立，在本地需要进行一定的 Path Routing 设置才能够正常访问，因此推荐使用 Caddy 来启用一个本地服务器。

开发配置 `Caddyfile.dev` 已提供在项目根目录，只需 `caddy run --config Caddyfile.dev` 即可启动本地服务器。

之后浏览器访问本地 localhost 即能够正常预览页面。
