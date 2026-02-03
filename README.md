# JoiAsk 提问箱

![main page](doc/screenshot.png)

样例可见 [轴伊Joi的提问箱](https://ask.vjoi.cn/)

## Quick Start 快速开始

### 使用 Docker（推荐）

#### 1. 准备配置文件

创建一个配置文件 `config.json`，最简单的配置示例（使用 SQLite）：

```json
{
    "db_type": "sqlite",
    "sqlite": "/work/db/jask.db",
    "server": {
        "host": "0.0.0.0",
        "port": 8080
    },
    "storage_type": "local"
}
```

#### 2. 启动容器

```bash
docker run -d --restart always \
    --name joiask \
    -p 80:80 \
    -v $(pwd)/config.json:/work/config/config.json \
    -v $(pwd)/data:/work/db \
    -v $(pwd)/uploads:/work/frontend/public/upload-img \
    ghcr.io/xinrea/joiask:latest
```

#### 3. 访问服务

容器只暴露一个端口（80），内部 Nginx 自动处理路由：

- 前端提问箱: `http://localhost/`
- 管理后台: `http://localhost/admin`
- 后端 API: `http://localhost/api/`

**默认管理员账号**: `admin` / `admin`

⚠️ **重要**: 首次登录后台后请立即修改管理员密码！

## Build 构建

### 从源码构建

```bash
docker build -t joiask .
```

构建完成后运行：

```bash
docker run -d --restart always \
    --name joiask \
    -p 80:80 \
    -v $(pwd)/config.json:/work/config/config.json \
    -v $(pwd)/data:/work/db \
    -v $(pwd)/uploads:/work/frontend/public/upload-img \
    joiask
```

## Configuration 配置

### 数据库配置

#### 使用 SQLite（推荐新手）

SQLite 是单文件数据库，无需额外安装数据库服务。

```json
{
    "db_type": "sqlite",
    "sqlite": "/work/db/jask.db",
    "server": {
        "host": "0.0.0.0",
        "port": 8080
    },
    "storage_type": "local"
}
```

挂载数据目录：

```bash
-v $(pwd)/data:/work/db
```

#### 使用 MySQL

```json
{
    "db_type": "mysql",
    "mysql": {
        "host": "192.168.1.100",
        "port": 3306,
        "user": "root",
        "pass": "your_password",
        "name": "jask"
    },
    "server": {
        "host": "0.0.0.0",
        "port": 8080
    },
    "storage_type": "local"
}
```

### 图片存储配置

#### 本地存储（推荐）

图片存储在容器的 `/work/frontend/public/upload-img/` 目录。

```json
{
    "storage_type": "local"
}
```

挂载存储目录：

```bash
-v $(pwd)/uploads:/work/frontend/public/upload-img
```

#### OSS 存储（阿里云）

使用阿里云 OSS 存储图片，适合大规模部署。

```json
{
    "storage_type": "oss",
    "oss": {
        "address": "https://your-domain.com",
        "endpoint": "oss-cn-beijing.aliyuncs.com",
        "access_key": "your_access_key",
        "secret_key": "your_secret_key",
        "bucket": "your-bucket-name"
    }
}
```

## 使用 Nginx 反向代理（HTTPS）

如果需要使用 HTTPS，可以在宿主机上配置 Nginx 反向代理：

```nginx
server {
    listen 443 ssl;
    server_name ask.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:80;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## 完整配置示例

### 最小配置（SQLite + 本地存储）

```json
{
    "db_type": "sqlite",
    "sqlite": "/work/db/jask.db",
    "server": {
        "host": "0.0.0.0",
        "port": 8080
    },
    "storage_type": "local"
}
```

### 生产环境配置（MySQL + OSS）

```json
{
    "db_type": "mysql",
    "mysql": {
        "host": "192.168.1.100",
        "port": 3306,
        "user": "joiask",
        "pass": "secure_password",
        "name": "joiask"
    },
    "server": {
        "host": "0.0.0.0",
        "port": 8080
    },
    "storage_type": "oss",
    "oss": {
        "address": "https://cdn.example.com",
        "endpoint": "oss-cn-beijing.aliyuncs.com",
        "access_key": "your_access_key",
        "secret_key": "your_secret_key",
        "bucket": "joiask-images"
    }
}
```

## Docker Compose（可选）

创建 `docker-compose.yml`:

```yaml
version: '3.8'

services:
  joiask:
    image: ghcr.io/xinrea/joiask:latest
    container_name: joiask
    restart: always
    ports:
      - "80:80"
    volumes:
      - ./config.json:/work/config/config.json
      - ./data:/work/db
      - ./uploads:/work/frontend/public/upload-img
```

启动：

```bash
docker-compose up -d
```

## 更新容器

```bash
# 停止并删除旧容器
docker stop joiask && docker rm joiask

# 拉取最新镜像
docker pull ghcr.io/xinrea/joiask:latest

# 启动新容器
docker run -d --restart always \
    --name joiask \
    -p 80:80 \
    -v $(pwd)/config.json:/work/config/config.json \
    -v $(pwd)/data:/work/db \
    -v $(pwd)/uploads:/work/frontend/public/upload-img \
    ghcr.io/xinrea/joiask:latest
```

## 故障排查

### 查看容器日志

```bash
docker logs joiask
```

### 进入容器调试

```bash
docker exec -it joiask bash
```

### 常见问题

**Q: 容器启动后无法访问？**

- 检查端口是否被占用：`netstat -tlnp | grep 80`
- 检查防火墙设置
- 查看容器日志：`docker logs joiask`

**Q: 图片上传失败？**

- 检查存储目录是否正确挂载
- 如果使用 OSS，检查 access_key 和 bucket 配置
- 查看后端日志确认错误信息

**Q: 数据丢失？**

- 确保数据库文件和上传目录已正确挂载到宿主机
- 使用 `-v` 参数挂载目录，避免数据随容器删除而丢失
