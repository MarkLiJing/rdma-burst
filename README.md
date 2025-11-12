# RDMA 大文件传输服务

基于现有 rtranfile 命令行工具的 RDMA 大文件传输 RESTful API 服务。

## 功能特性

- 🚀 **高性能传输**: 支持 RDMA 大页内存、tmpfs、文件系统三种传输模式
- 🔄 **RESTful API**: 完整的 HTTP API 接口，支持传输管理和状态监控
- ⚡ **并发控制**: 单次传输，避免并发操作，支持传输间隔配置
- 📊 **状态监控**: 实时传输进度、速度、错误信息监控
- 🔒 **错误恢复**: 传输中断后的恢复机制和完整性校验
- 🐳 **容器化支持**: Docker 容器化部署配置

## 传输模式

### 1. 大页内存传输 (Hugepages)
- 最高性能的传输模式
- 适用于 50GB+ 大文件传输
- 需要配置大页内存

### 2. tmpfs 文件传输
- 内存文件系统传输
- 平衡性能和内存使用
- 适用于中等大小文件

### 3. 文件系统传输
- 传统文件系统传输
- 最大兼容性
- 适用于通用场景

## 快速开始

### 系统要求

- Linux 操作系统
- RDMA 设备支持 (mlx5_0)
- Go 1.21+ 环境
- rtranfile 二进制文件

### 安装部署

```bash
# 1. 克隆项目
git clone <repository-url>
cd rdma-burst

# 2. 准备 rtranfile 工具
cp /path/to/rtranfile ./bin/
chmod +x ./bin/rtranfile

# 3. 安装依赖
go mod tidy

# 4. 配置大页内存 (可选)
echo 1024 > /sys/kernel/mm/hugepages/hugepages-2048kB/nr_hugepages
mkdir -p /dev/hugepages/dir
mount -t hugetlbfs nodev /dev/hugepages/dir

# 5. 启动服务
go run cmd/server/main.go --config configs/server.yaml
```

### API 使用示例

```bash
# 创建传输任务
curl -X POST http://localhost:8080/api/v1/transfers \
  -H "Content-Type: application/json" \
  -d '{
    "source_path": "/data/largefile.iso",
    "destination_path": "/dev/hugepages/dir/largefile.iso", 
    "transfer_mode": "hugepages",
    "file_size": 53687091200
  }'

# 检查传输状态
curl http://localhost:8080/api/v1/transfers/{task_id}

# 列出所有任务
curl http://localhost:8080/api/v1/transfers
```

## 项目结构

```
rdma-burst/
├── cmd/                 # 应用程序入口
│   ├── server/         # 服务端主程序
│   └── client/         # 客户端主程序
├── internal/           # 内部包
│   ├── api/           # API 处理层
│   ├── services/      # 业务逻辑层
│   ├── models/        # 数据模型
│   └── wrapper/       # rtranfile 包装器
├── pkg/               # 可重用包
│   ├── logger/        # 日志系统
│   ├── utils/         # 工具函数
│   └── types/         # 公共类型
├── configs/           # 配置文件
├── tests/             # 测试文件
├── docs/              # 文档
└── specs/             # 项目规范
```

## 配置说明

### 服务端配置 (configs/server.yaml)

```yaml
server:
  host: "0.0.0.0"
  port: 8080

transfer:
  device: "mlx5_0"
  base_dir: "/var/lib/rtrans"
  transfer_interval: "5s"
  max_concurrent_transfers: 1
```

### 客户端配置 (configs/client.yaml)

```yaml
server:
  host: "localhost" 
  port: 8080
  timeout: "30s"

transfer:
  device: "mlx5_0"
  default_mode: "filesystem"
```

## API 文档

详细的 API 接口文档请参考 [API 文档](docs/api/README.md) 或 [OpenAPI 规范](specs/001-rdma-file-transfer/contracts/openapi.yaml)。

## 开发指南

### 构建项目

```bash
# 构建服务端
go build -o bin/server cmd/server/main.go

# 构建客户端
go build -o bin/client cmd/client/main.go
```

### 运行测试

```bash
# 运行单元测试
go test ./tests/unit/...

# 运行集成测试  
go test ./tests/integration/...

# 运行所有测试
go test ./...
```

### 代码规范

项目遵循标准的 Go 代码规范：
- 使用 `go fmt` 格式化代码
- 使用 `go vet` 检查代码问题
- 遵循 Go 命名约定

## 部署指南

### Docker 部署

```bash
# 构建 Docker 镜像
docker build -t rdma-burst .

# 运行容器
docker run -d \
  --name rdma-burst \
  --privileged \
  -p 8080:8080 \
  -v /dev/hugepages:/dev/hugepages \
  rdma-burst
```

### 系统服务部署

参考 [部署文档](docs/deployment/README.md) 了解 systemd 服务配置和监控设置。

## 故障排除

常见问题请参考 [故障排除指南](docs/troubleshooting.md)。

## 贡献指南

欢迎提交 Issue 和 Pull Request！请参考 [贡献指南](CONTRIBUTING.md)。

## 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。

## 联系方式

- 项目主页: <repository-url>
- 问题反馈: <issues-url>
- 文档: <docs-url>