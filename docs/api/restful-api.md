# RDMA大文件传输服务 RESTful API 文档

## 基础信息

### API 端点
- **基础URL**: `http://localhost:8080/api/v1`
- **健康检查**: `http://localhost:8080/api/health`
- **模式检测**: `http://localhost:8080/api/v1/mode`

### 认证
当前版本无需认证，生产环境建议启用TLS和认证。

### 响应格式
所有API响应都使用JSON格式，包含标准字段：
```json
{
  "status": "success|error",
  "data": {...},
  "message": "描述信息",
  "timestamp": "2025-11-07T07:00:00Z"
}
```

## 模式检测 API

### 1. 获取当前运行模式

**端点**: `GET /api/v1/mode`

**描述**: 获取当前服务的运行模式

**响应**:
```json
{
  "mode": "server",
  "version": "1.0.0",
  "status": "running",
  "timestamp": "2025-11-07T07:00:00Z",
  "uptime": "1h23m45s"
}
```

**示例**:
```bash
curl http://localhost:8080/api/v1/mode
```

### 2. 检测运行模式

**端点**: `GET /api/v1/mode/detect`

**描述**: 检测当前环境应该运行的模式

**响应**:
```json
{
  "mode": "client",
  "version": "1.0.0",
  "status": "detected",
  "timestamp": "2025-11-07T07:00:00Z",
  "uptime": "1h23m45s"
}
```

**示例**:
```bash
curl http://localhost:8080/api/v1/mode/detect
```

### 3. 获取模式状态

**端点**: `GET /api/v1/mode/status`

**描述**: 获取详细的模式状态信息

**响应**:
```json
{
  "mode": {
    "current": "server",
    "detected": "client",
    "supported_modes": ["server", "client", "auto"]
  },
  "service": {
    "version": "1.0.0",
    "uptime": "1h23m45s",
    "start_time": "2025-11-07T05:36:15Z",
    "status": "running"
  },
  "detection": {
    "method": "health_check",
    "timeout": "3s",
    "endpoint": "http://localhost:8080/api/health"
  },
  "timestamp": "2025-11-07T07:00:00Z"
}
```

**示例**:
```bash
curl http://localhost:8080/api/v1/mode/status
```

### 4. 切换运行模式

**端点**: `POST /api/v1/mode/switch`

**描述**: 请求切换运行模式（需要重启服务生效）

**请求体**:
```json
{
  "mode": "client"
}
```

**响应**:
```json
{
  "current_mode": "server",
  "target_mode": "client",
  "message": "模式切换请求已接受，需要重启服务生效",
  "restart_required": true,
  "timestamp": "2025-11-07T07:00:00Z"
}
```

**示例**:
```bash
curl -X POST http://localhost:8080/api/v1/mode/switch \
  -H "Content-Type: application/json" \
  -d '{"mode": "client"}'
```

## 传输管理 API

### 1. 创建传输任务

**端点**: `POST /api/v1/transfers`

**描述**: 创建新的RDMA文件传输任务

**请求体**:
```json
{
  "filename": "/path/to/file.bin",
  "mode": "filesystem",
  "direction": "put",
  "server_ip": "192.168.1.100"
}
```

**参数说明**:
- `filename`: 文件名（必需）
- `mode`: 传输模式 `hugepages|tmpfs|filesystem`（必需）
- `direction`: 传输方向 `put|get`（必需）
- `server_ip`: 服务端IP地址（客户端传输时必需）

**响应**:
```json
{
  "id": "task_1234567890",
  "status": "pending",
  "message": "传输任务已创建",
  "created_at": "2025-11-07T07:00:00Z"
}
```

**示例**:

**在服务端执行**（服务端IP: 192.168.1.100）:
```bash
curl -X POST http://localhost:8080/api/v1/transfers \
  -H "Content-Type: application/json" \
  -d '{
    "filename": "/data/largefile.iso",
    "mode": "filesystem",
    "direction": "put",
    "server_ip": "192.168.1.100"
  }'
```

**在客户端执行**（客户端API端口为8081）:
```bash
curl -X POST http://localhost:8081/api/v1/transfers \
  -H "Content-Type: application/json" \
  -d '{
    "filename": "/data/largefile.iso",
    "mode": "filesystem",
    "direction": "put",
    "server_ip": "192.168.1.100"
  }'
```

### 2. 获取传输状态

**端点**: `GET /api/v1/transfers/{task_id}`

**描述**: 获取指定传输任务的状态和进度

**路径参数**:
- `task_id`: 任务ID

**响应**:
```json
{
  "id": "task_1234567890",
  "status": "in_progress",
  "progress": 45.5,
  "bytes_transferred": 455000000,
  "total_bytes": 1000000000,
  "transfer_rate": 125.5,
  "elapsed_time": "3m45s",
  "estimated_time": "4m15s",
  "error": "",
  "last_updated": "2025-11-07T07:03:45Z"
}
```

**示例**:
```bash
curl http://localhost:8080/api/v1/transfers/task_1234567890
```

### 3. 列出传输任务

**端点**: `GET /api/v1/transfers`

**描述**: 获取传输任务列表，支持分页

**查询参数**:
- `page`: 页码（默认: 1）
- `size`: 每页大小（默认: 20，最大: 100）

**响应**:
```json
{
  "tasks": [
    {
      "id": "task_1234567890",
      "filename": "/data/largefile.iso",
      "mode": "filesystem",
      "direction": "put",
      "status": "completed",
      "progress": 100.0,
      "bytes_transferred": 1000000000,
      "total_bytes": 1000000000,
      "start_time": "2025-11-07T07:00:00Z",
      "end_time": "2025-11-07T07:08:15Z",
      "created_at": "2025-11-07T07:00:00Z",
      "updated_at": "2025-11-07T07:08:15Z"
    }
  ],
  "total": 15,
  "page": 1,
  "size": 20
}
```

**示例**:
```bash
curl "http://localhost:8080/api/v1/transfers?page=1&size=10"
```

### 4. 取消传输任务

**端点**: `DELETE /api/v1/transfers/{task_id}`

**描述**: 取消指定的传输任务

**路径参数**:
- `task_id`: 任务ID

**响应**:
```json
{
  "id": "task_1234567890",
  "status": "cancelled",
  "message": "传输任务已取消"
}
```

**示例**:
```bash
curl -X DELETE http://localhost:8080/api/v1/transfers/task_1234567890
```

### 5. 获取活跃传输数量

**端点**: `GET /api/v1/transfers/active`

**描述**: 获取当前活跃的传输任务数量

**响应**:
```json
{
  "active_transfers": 2,
  "timestamp": "2025-11-07T07:00:00Z"
}
```

**示例**:
```bash
curl http://localhost:8080/api/v1/transfers/active
```

## 健康检查 API

### 1. 健康检查

**端点**: `GET /api/health`

**描述**: 检查服务健康状态

**响应**:
```json
{
  "status": "healthy",
  "timestamp": "2025-11-07T07:00:00Z",
  "version": "1.0.0",
  "extra_info": {
    "uptime": "1h23m45s",
    "active_transfers": 2,
    "start_time": "2025-11-07T05:36:15Z"
  }
}
```

**示例**:
```bash
curl http://localhost:8080/api/health
```

### 2. 就绪检查

**端点**: `GET /api/ready`

**描述**: 检查服务是否就绪

**响应**:
```json
{
  "status": "ready",
  "timestamp": "2025-11-07T07:00:00Z",
  "version": "1.0.0"
}
```

**示例**:
```bash
curl http://localhost:8080/api/ready
```

### 3. 存活检查

**端点**: `GET /api/live`

**描述**: 检查服务是否存活

**响应**:
```json
{
  "status": "alive",
  "timestamp": "2025-11-07T07:00:00Z",
  "version": "1.0.0"
}
```

**示例**:
```bash
curl http://localhost:8080/api/live
```

### 4. 服务指标

**端点**: `GET /api/metrics`

**描述**: 获取服务运行指标

**响应**:
```json
{
  "service": {
    "name": "rdma-burst",
    "version": "1.0.0",
    "uptime_seconds": 5025.45,
    "start_time": "2025-11-07T05:36:15Z"
  },
  "transfers": {
    "active": 2,
    "total": 15
  },
  "system": {
    "goroutines": 25,
    "timestamp": "2025-11-07T07:00:00Z"
  }
}
```

**示例**:
```bash
curl http://localhost:8080/api/metrics
```

## 根路径 API

### 服务信息

**端点**: `GET /`

**描述**: 获取服务基本信息

**响应**:
```json
{
  "service": "rdma-burst",
  "mode": "server",
  "version": "1.0.0",
  "status": "running"
}
```

**示例**:
```bash
curl http://localhost:8080/
```

## 错误处理

### 错误响应格式

所有错误都返回标准格式：

```json
{
  "error": "ERROR_CODE",
  "message": "错误描述信息",
  "code": 400
}
```

### 常见错误码

- `400 Bad Request`: 请求参数无效
- `404 Not Found`: 资源不存在
- `409 Conflict`: 资源冲突（如重复启动）
- `500 Internal Server Error`: 服务器内部错误
- `503 Service Unavailable`: 服务不可用

### 错误示例

```json
{
  "error": "TASK_NOT_FOUND",
  "message": "任务不存在: task_1234567890",
  "code": 404
}
```

## 使用示例

### 完整的文件传输流程

```bash
# 1. 检查服务状态
curl http://localhost:8080/api/health

# 2. 创建传输任务
response=$(curl -s -X POST http://localhost:8080/api/v1/transfers \
  -H "Content-Type: application/json" \
  -d '{
    "filename": "/data/largefile.iso",
    "mode": "filesystem",
    "direction": "put",
    "server_ip": "192.168.1.100"
  }')

# 提取任务ID
task_id=$(echo $response | jq -r '.id')

# 3. 监控传输进度
while true; do
  status=$(curl -s http://localhost:8080/api/v1/transfers/$task_id)
  progress=$(echo $status | jq -r '.progress')
  echo "进度: $progress%"
  
  if [ "$progress" = "100" ]; then
    echo "传输完成"
    break
  fi
  
  sleep 5
done

# 4. 获取最终状态
curl http://localhost:8080/api/v1/transfers/$task_id
```

### 批量操作示例

```bash
# 批量创建传输任务
files=("file1.bin" "file2.bin" "file3.bin")
for file in "${files[@]}"; do
  curl -X POST http://localhost:8080/api/v1/transfers \
    -H "Content-Type: application/json" \
    -d "{\"filename\": \"$file\", \"mode\": \"filesystem\", \"direction\": \"put\"}"
done

# 批量检查状态
curl -s http://localhost:8080/api/v1/transfers?size=100 | jq '.tasks[] | {id, filename, status, progress}'
```

这个API文档提供了完整的RESTful接口说明，包括模式检测、传输管理、健康检查等所有功能。