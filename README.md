# Droid API Key Usage Monitor (Go + Redis)

🚀 高性能的 Droid API Key 余额监控系统，使用 Go + Redis 构建，支持管理数千个 API Keys。

## ✨ 特性

- **高并发支持**: 使用 Worker Pool 并发处理数千个 API Keys
- **Redis 存储**: 高性能的数据存储和缓存
- **批量操作**: 支持批量导入、删除、复制 API Keys  
- **实时监控**: 自动刷新功能，实时追踪使用情况
- **密码保护**: 可选的管理员认证机制
- **Docker 部署**: 一键部署，支持生产环境配置
- **性能优化**: Redis Pipeline、本地缓存、连接池等优化

## 🏗️ 架构

- **后端**: Go + Fiber v2 (高性能 Web 框架)
- **存储**: Redis 7.x (支持 Pipeline 批量操作)
- **前端**: 原生 HTML/CSS/JavaScript (Apple 风格 UI)
- **部署**: Docker + Docker Compose

## 📊 性能指标

| 指标 | 目标值 | 说明 |
|------|--------|------|
| 并发查询 | 5000 keys/10s | 使用 100 个 worker |
| Redis 响应 | < 1ms | Pipeline 批量操作 |
| 内存占用 | < 200MB | 应用本身 |
| API 延迟 | P99 < 500ms | 缓存命中时 |
| 吞吐量 | 1000 req/s | 单实例 |

## 🚀 快速开始

### 使用 Docker Compose (推荐)

1. **克隆项目**
```bash
git clone <repository>
cd Droid-keyusage-go
```

2. **配置环境变量**
```bash
cp .env.example .env
# 编辑 .env 文件设置密码等配置
```

3. **启动服务**
```bash
# 开发环境
docker-compose up -d

# 或使用 Make
make docker-up
```

4. **访问应用**
```
http://localhost:8080
```

### 本地开发

1. **安装依赖**
```bash
# 需要 Go 1.21+
go mod download
```

2. **启动 Redis**
```bash
docker run -d -p 6379:6379 redis:7-alpine
```

3. **运行应用**
```bash
make run
# 或
go run cmd/server/main.go
```

## 📝 配置说明

### 环境变量

```env
# 服务器配置
PORT=8080                    # 服务端口
ENV=development             # 环境: development/production

# Redis 配置
REDIS_URL=redis://localhost:6379/0
REDIS_PASSWORD=             # 生产环境设置密码

# 认证
ADMIN_PASSWORD=your-password  # 管理员密码

# 性能调优
MAX_WORKERS=100             # Worker 池大小
QUEUE_SIZE=10000            # 任务队列大小
HTTP_TIMEOUT=30s            # HTTP 请求超时
CACHE_TTL=5m                # 缓存有效期
```

## 🛠️ 开发

### 目录结构

```
Droid-keyusage-go/
├── cmd/server/         # 程序入口
├── internal/           # 内部包
│   ├── api/           # HTTP 处理器和路由
│   ├── services/      # 业务逻辑
│   ├── storage/       # Redis 存储层
│   └── models/        # 数据模型
├── web/static/        # 前端资源
├── docker/            # Docker 配置
└── docker-compose.yml # 编排文件
```

### 常用命令

```bash
# 构建
make build              # 构建二进制文件
make docker-build       # 构建 Docker 镜像

# 运行
make run               # 本地运行
make docker-up         # Docker 运行

# 测试
make test              # 运行测试
make test-coverage     # 生成覆盖率报告

# 代码质量
make fmt               # 格式化代码
make lint              # 运行 linter
make vet               # 运行 go vet

# Docker
make docker-logs       # 查看日志
make docker-restart    # 重启服务
make redis-cli         # 连接 Redis CLI

# 监控
make monitor           # 启动 Prometheus + Grafana
```

## 🚢 生产部署

### 使用 Docker Swarm/K8s

```bash
# 构建生产镜像
docker build -f docker/Dockerfile -t keyusage:latest .

# 使用生产配置启动
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

### 性能优化建议

1. **Redis 配置**
   - 设置合适的 `maxmemory` 和淘汰策略
   - 开启持久化 (AOF)
   - 使用 Redis Sentinel 实现高可用

2. **应用配置**
   - 根据服务器资源调整 `MAX_WORKERS`
   - 设置合理的 `CACHE_TTL` 减少 API 调用
   - 使用连接池管理 HTTP 连接

3. **部署建议**
   - 使用 Nginx 反向代理和负载均衡
   - 开启 HTTPS
   - 配置监控告警

## 📊 监控

启动监控栈:

```bash
make monitor
```

访问:
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000 (admin/admin)

## 🔒 安全建议

1. 设置强密码 (`ADMIN_PASSWORD`)
2. 生产环境使用 HTTPS
3. 配置防火墙规则
4. 定期备份 Redis 数据
5. 使用环境变量管理敏感信息

## 📈 性能测试

```bash
# 运行基准测试
make benchmark

# 压力测试 (需要安装 vegeta)
echo "GET http://localhost:8080/api/data" | vegeta attack -rate=100 -duration=30s | vegeta report
```

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可

MIT License

## 🙋 支持

如有问题，请提交 Issue 或联系维护者。

---

Made with ❤️ by Droid Team
