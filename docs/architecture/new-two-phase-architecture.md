# 新的两阶段传输架构文档

## 架构概述

新的两阶段传输架构解决了之前API调用卡住的问题，采用了更清晰的分离设计：

1. **准备阶段 (PrepareTransfer)**: 客户端发送HTTP请求到服务端，服务端启动传输模式
2. **传输阶段 (StartTransfer)**: 服务端响应成功后，客户端执行传输任务
3. **资源回收**: 传输完成后，服务端和客户端都进行资源回收

## 主要修改

### 1. 传输服务 (`internal/services/transfer/transfer.go`)

#### 新增方法: `PrepareTransfer`
```go
func (ts *TransferService) PrepareTransfer(req *models.TransferRequest, serverConfig *models.TransferSettings) error
```
- 启动服务端监听进程
- 等待服务端进程启动（带超时机制）
- 返回准备状态

#### 简化方法: `startTransferTask`
- 移除了重复的服务端启动逻辑
- 直接启动客户端传输，假设服务端已准备就绪
- 添加了更好的错误处理和超时机制

### 2. API处理器 (`internal/api/handlers/transfers.go`)

#### 修改方法: `CreateTransfer`
```go
// 第一步：准备传输环境（启动服务端监听进程）
if err := h.transferService.PrepareTransfer(&req, serverConfig); err != nil {
    // 错误处理
}

// 第二步：创建传输任务
response, err := h.transferService.StartTransfer(&req, serverConfig)
```

### 3. 服务端主程序 (`cmd/combined/main.go`)

- 移除了同时启动三个监听进程的复杂逻辑
- 采用按需启动的方式，只在需要时启动对应模式的监听进程

## 架构优势

### 解决的问题
1. **API调用卡住**: 添加了超时机制和更好的错误处理
2. **资源管理**: 按需启动服务端进程，避免资源浪费
3. **并发限制**: 改进了连接状态管理，避免并发限制错误

### 性能改进
1. **按需启动**: 只在需要时启动服务端监听进程
2. **资源回收**: 传输完成后自动清理资源
3. **超时控制**: 添加了5秒超时机制，避免无限等待

## 使用流程

### 客户端使用API的流程
**注意：两个阶段已经合成为一个API调用，客户端只需要执行一次请求**

```bash
# 单个API调用完成整个传输过程
curl -X POST http://localhost:8080/api/v1/transfers \
  -H "Content-Type: application/json" \
  -d '{
    "filename": "/tmp/testfile.bin",
    "mode": "filesystem",
    "direction": "put"
  }'
```

### 内部处理流程
虽然架构分为两个阶段，但对客户端来说是透明的：
1. **自动准备阶段**: API内部调用 `PrepareTransfer` 启动服务端监听进程
2. **自动传输阶段**: API内部调用 `StartTransfer` 执行传输任务
3. **客户端体验**: 只需要一次API调用

### 服务端配置
服务端不再需要预先启动所有模式的监听进程，而是根据客户端请求按需启动。

## 测试验证

### 测试脚本
创建了测试脚本 `scripts/test-new-architecture.sh` 来验证新架构：

```bash
./scripts/test-new-architecture.sh
```

### 测试内容
1. 文件系统模式上传测试
2. 文件系统模式下载测试
3. 传输状态检查
4. 活跃传输数量检查

## 配置要求

### 服务端配置 (`configs/combined.yaml`)
```yaml
device: "mlx5_0"
base_dir: "/var/lib/rtrans"
transfer_interval: 5000000000  # 5秒
max_concurrent_transfers: 1
chunk_size: 4194304

modes:
  hugepages:
    enabled: true
    base_dir: "/dev/hugepages/dir"
  tmpfs:
    enabled: true
    base_dir: "/dev/shm/dir"
  filesystem:
    enabled: true
    base_dir: "/var/lib/rtrans/files"
```

## 部署说明

### 1. 构建可执行文件
```bash
make build
```

### 2. 启动服务端
```bash
./build/rdma-burst --mode combined --config configs/combined.yaml
```

### 3. 测试API
```bash
# 测试新的两阶段架构
./scripts/test-new-architecture.sh
```

## 问题解答

### Q: API调用应该在客户端还是服务端执行？
A: **客户端执行**。localhost指的是客户端的地址，客户端通过HTTP API与服务端通信。

### Q: 使用API传输时是否需要手动启动客户端？
A: **不需要**。新的两阶段架构中，客户端只需要发送HTTP请求到服务端，服务端会自动处理传输过程。

### Q: 服务端地址如何配置？
A: 服务端地址现在从配置文件中获取，不再需要客户端在请求中指定。这简化了API使用。

## 总结

新的两阶段传输架构提供了更清晰、更可靠的传输流程，解决了之前遇到的API卡住和资源管理问题。通过分离准备和传输阶段，实现了更好的错误处理和资源管理。