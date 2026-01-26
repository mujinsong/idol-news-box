# Weibo Spider

微博爬虫服务，支持爬取指定用户的微博内容、图片和视频。

## 功能特性

- 爬取用户微博列表
- 下载微博图片和视频
- 支持异步任务队列
- 多种输出格式（JSON、CSV、TXT）
- RESTful API 接口
- 定时任务调度

## 快速开始

### 1. 启动中间件

使用 Docker Compose 启动 PostgreSQL 数据库：

```bash
docker-compose up -d
```

### 2. 配置

复制配置文件并修改：

```bash
cp configs/config.sample.json config.json
```

### 3. 获取微博 Cookie

**重要：Cookie 是爬虫能够正常工作的关键。**

#### 获取步骤：

1. 打开浏览器，访问 [https://weibo.cn](https://weibo.cn)
2. 登录你的微博账号
3. 按 `F12` 打开开发者工具
4. 切换到 `Network`（网络）标签
5. 刷新页面，点击任意一个请求
6. 在 `Headers` 中找到 `Cookie` 字段
7. 复制整个 Cookie 值到配置文件

![获取Cookie示意图](https://via.placeholder.com/600x300?text=F12+->+Network+->+Headers+->+Cookie)

#### Cookie 示例：

```
SUB=xxx; SUBP=xxx; _T_WM=xxx; WEIBOCN_FROM=xxx; MLOGIN=1; M_WEIBOCN_PARAMS=xxx
```

### 4. 运行

```bash
go build -o weibo-spider ./cmd/weibo-spider
./weibo-spider -config config.json
```

## 配置文件说明

```json
{
    "server": {
        "port": 8080,          // 服务端口
        "mode": "debug"        // 运行模式: debug/release
    },
    "database": {
        "host": "localhost",   // 数据库地址
        "port": 5432,          // 数据库端口
        "user": "postgres",    // 数据库用户名
        "password": "postgres123",  // 数据库密码
        "dbname": "weibo_spider",   // 数据库名
        "sslmode": "disable"   // SSL 模式
    },
    "random_wait_pages": [1, 5],    // 每爬取 1-5 页后随机等待
    "random_wait_seconds": [6, 10], // 随机等待 6-10 秒（防止被封）
    "write_mode": ["csv", "json"],  // 输出格式: csv/json/txt
    "cookie": "your_cookie_here",   // 微博 Cookie（必填）
    "output_dir": "./output"        // 输出目录
}
```

## Docker Compose 说明

`docker-compose.yml` 包含以下服务：

| 服务 | 镜像 | 端口 | 说明 |
|------|------|------|------|
| postgresql | bitnami/postgresql:latest | 5432 | 数据库，存储定时任务 |

启动命令：

```bash
# 启动
docker-compose up -d

# 查看状态
docker-compose ps

# 停止
docker-compose down

# 停止并删除数据
docker-compose down -v
```

## API 接口

### 提交爬取任务

```bash
POST /api/v1/weibos
Content-Type: application/json

{
  "user_id": "用户ID",
  "since_date": "2025-01-01",
  "end_date": "now",
  "filter": 0,
  "download_media": true
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| user_id | string | 是 | 微博用户 ID |
| since_date | string | 否 | 开始日期，默认一周前 |
| end_date | string | 否 | 结束日期，默认当前时间 |
| filter | int | 否 | 0=全部，1=仅原创 |
| download_media | bool | 否 | 是否下载图片和视频 |

**响应：**

```json
{
  "code": 0,
  "data": {
    "task_id": "a1b2c3d4-...",
    "status": "pending",
    "message": "任务提交成功"
  }
}
```

### 查询任务状态

```bash
GET /api/v1/task/{task_id}
```

**响应：**

```json
{
  "code": 0,
  "data": {
    "task_id": "a1b2c3d4-...",
    "status": "running",
    "progress": {
      "total_weibos": 50,
      "crawled_weibos": 30,
      "total_images": 100,
      "downloaded_images": 45,
      "total_videos": 5,
      "downloaded_videos": 2
    }
  }
}
```

### 获取用户信息

```bash
GET /api/v1/user/{user_id}
```

### 健康检查

```bash
GET /api/v1/health
```

## 输出目录结构

```
output/
└── {user_id}/
    ├── weibos.json      # 微博数据 (JSON)
    ├── weibos.csv       # 微博数据 (CSV)
    ├── images/          # 图片目录
    │   └── {weibo_id}/
    │       └── xxx.jpg
    └── videos/          # 视频目录
        └── {weibo_id}/
            └── xxx.mp4
```

## 注意事项

1. **Cookie 有效期**：微博 Cookie 会过期，如果爬取失败请重新获取
2. **频率限制**：配置合理的等待时间，避免被微博封禁
3. **仅供学习**：请遵守微博使用条款，不要用于商业用途

## License

MIT
