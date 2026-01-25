# Weibo Spider

微博爬虫服务，支持爬取指定用户的微博内容、图片和视频。

## 功能特性

- 爬取用户微博列表
- 下载微博图片和视频
- 支持异步任务队列
- 多种输出格式（JSON、CSV、TXT）
- RESTful API 接口

## 快速开始

### 1. 配置

复制配置文件并填写微博 Cookie：

```bash
cp config.example.json config.json
```

### 2. 运行

```bash
go build -o weibo-spider ./cmd/weibo-spider
./weibo-spider -config config.json
```

## API 接口

### 提交爬取任务

```bash
POST /api/v1/weibos
{
  "user_id": "用户ID",
  "since_date": "2025-01-01",
  "end_date": "now",
  "download_media": true
}
```

### 查询任务状态

```bash
GET /api/v1/task/{task_id}
```

### 获取用户信息

```bash
GET /api/v1/user/{user_id}
```

## License

MIT
