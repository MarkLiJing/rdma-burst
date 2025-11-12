# API 文档

RDMA 大文件传输服务的 RESTful API 接口文档。

## 基础信息

- **Base URL**: `http://localhost:8080`
- **API 版本**: `v1`
- **认证**: 当前版本无需认证
- **数据格式**: JSON

## 快速开始

### 1. 健康检查

检查服务是否正常运行。

```http
GET /api/health
```

**响应示例**:
```json
{
  "status": "healthy",
  "timestamp": "2025-11-06T09:00:00Z",
  "version": "1.0.0"
}
```

### 2. 创建传输任务

创建一个新的文件传输任务。

```http
POST /api/v1/transfers
```

**请求体**:
```json
{
  "source_path": "/data/largefile.iso",
  "destination_path": "/dev/hugepages/dir/largefile.iso",
  "transfer_mode": "hugepages",
  "file_size": 53687091200,
  "description": "50GB 大文件传输"
}
```

**响应示例**:
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "source_path": "/data/largefile.iso",
  "destination_path": "/dev/hugepages/dir/largefile.iso",
  "transfer_mode": "hugepages",
  "file_size": 53687091200,
  "status": "pending",
  "progress": 0.0,
  "start_time": null,
  "end_time": null
}
```

### 3. 获取传输状态

获取指定传输任务的详细状态。

```http
GET /api/v1/transfers/{task_id}
```

**响应示例**:
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "source_path": "/data/largefile.iso",
  "destination_path": "/dev/hugepages/dir/largefile.iso",
  "transfer_mode": "hugepages",
  "file_size": 53687091200,
  "status": "running",
  "progress": 0.75,
  "bytes_transferred": 40265318400,
  "transfer_rate": 125.5,
  "estimated_time_remaining": "2m30s",
  "start_time": "2025-11-06T09:00:00Z",
  "end_time": null
}
```

## API 端点详情

### 健康检查

**端点**: `GET /api/health`

检查服务健康状态。

**响应状态码**:
- `200 OK`: 服务健康

### 传输任务管理

#### 创建传输任务

**端点**: `POST /api/v1/transfers`

创建新的文件传输任务。

**请求体参数**:
| 参数 | 类型 | 必填 | 描述 |
|------|------|------|------|
| `source_path` | string | 是 | 源文件路径 |
| `destination_path` | string | 是 | 目标文件路径 |
| `transfer_mode` | string | 是 | 传输模式 (hugepages/tmpfs/filesystem) |
| `file_size` | integer | 是 | 文件大小（字节） |
| `description` | string | 否 | 任务描述 |

**响应状态码**:
- `201 Created`: 任务创建成功
- `400 Bad Request`: 请求参数错误
- `409 Conflict`: 存在正在进行的传输任务

#### 获取任务列表

**端点**: `GET /api/v1/transfers`

获取所有传输任务的列表。

**查询参数**:
| 参数 | 类型 | 描述 |
|------|------|------|
| `status` | string | 按状态过滤 (pending/running/completed/failed/cancelled) |
| `limit` | integer | 返回结果数量限制（默认 50） |
| `offset` | integer | 结果偏移量（默认 0） |

**响应示例**:
```json
{
  "tasks": [
    {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "source_path": "/data/file1.iso",
      "destination_path": "/dev/hugepages/dir/file1.iso",
      "transfer_mode": "hugepages",
      "file_size": 53687091200,
      "status": "completed",
      "progress": 1.0,
      "start_time": "2025-11-06T09:00:00Z",
      "end_time": "2025-11-06T09:10:00Z"
    }
  ],
  "total": 1,
  "limit": 50,
  "offset": 0
}
```

#### 获取任务详情

**端点**: `GET /api/v1/transfers/{task_id}`

获取指定传输任务的详细状态。

**路径参数**:
- `task_id`: 传输任务 ID (UUID)

**响应状态码**:
- `200 OK`: 成功获取任务详情
- `404 Not Found`: 任务不存在

#### 取消传输任务

**端点**: `DELETE /api/v1/transfers/{task_id}`

取消指定的传输任务。

**路径参数**:
- `task_id`: 传输任务 ID (UUID)

**响应状态码**:
- `200 OK`: 任务取消成功
- `404 Not Found`: 任务不存在
- `409 Conflict`: 任务无法取消（已完成或失败）

## 错误处理

### 错误响应格式

所有错误响应都遵循以下格式：

```json
{
  "error": "错误描述",
  "code": "错误代码",
  "details": {
    "field": "具体错误信息"
  },
  "timestamp": "2025-11-06T09:00:00Z"
}
```

### 常见错误代码

| 错误代码 | 描述 | HTTP 状态码 |
|----------|------|-------------|
| `INVALID_REQUEST` | 请求参数无效 | 400 |
| `TASK_NOT_FOUND` | 任务不存在 | 404 |
| `TASK_ALREADY_RUNNING` | 存在正在进行的任务 | 409 |
| `TASK_CANNOT_CANCEL` | 任务无法取消 | 409 |
| `INTERNAL_ERROR` | 内部服务器错误 | 500 |

## 传输模式说明

### hugepages (大页内存)

最高性能的传输模式，需要配置大页内存。

**适用场景**:
- 50GB+ 大文件传输
- 对性能要求极高的场景

**配置要求**:
```bash
# 配置大页内存
echo 1024 > /sys/kernel/mm/hugepages/hugepages-2048kB/nr_hugepages
mkdir -p /dev/hugepages/dir
mount -t hugetlbfs nodev /dev/hugepages/dir
```

### tmpfs (内存文件系统)

平衡性能和内存使用的传输模式。

**适用场景**:
- 中等大小文件传输
- 需要较好性能但内存有限的场景

### filesystem (文件系统)

兼容性最好的传输模式。

**适用场景**:
- 各种大小文件传输
- 通用场景，兼容性要求高

## 使用示例

### cURL 示例

```bash
# 创建大页内存传输
curl -X POST http://localhost:8080/api/v1/transfers \
  -H "Content-Type: application/json" \
  -d '{
    "source_path": "/data/50G_file.rfile",
    "destination_path": "/dev/hugepages/dir/50G_file.rfile",
    "transfer_mode": "hugepages",
    "file_size": 53687091200
  }'

# 监控传输进度
while true; do
  curl http://localhost:8080/api/v1/transfers/123e4567-e89b-12d3-a456-426614174000
  sleep 10
done
```

### Python 示例

```python
import requests
import time

# 创建传输任务
response = requests.post("http://localhost:8080/api/v1/transfers", json={
    "source_path": "/data/largefile.iso",
    "destination_path": "/dev/hugepages/dir/largefile.iso",
    "transfer_mode": "hugepages",
    "file_size": 53687091200
})

task_id = response.json()["id"]

# 监控传输进度
while True:
    response = requests.get(f"http://localhost:8080/api/v1/transfers/{task_id}")
    status = response.json()
    
    print(f"进度: {status['progress']:.1%}, 速率: {status['transfer_rate']} MB/s")
    
    if status["status"] in ["completed", "failed", "cancelled"]:
        break
        
    time.sleep(10)
```

## 性能指标

### 传输速率参考

| 传输模式 | 预期速率 | 适用文件大小 |
|----------|----------|-------------|
| hugepages | 10-20 GB/s | 10GB+ |
| tmpfs | 5-10 GB/s | 1GB-10GB |
| filesystem | 1-5 GB/s | 任意大小 |

### 监控指标

- **传输进度**: 0.0 - 1.0
- **传输速率**: MB/s
- **已传输字节**: 字节数
- **预计剩余时间**: 时间字符串

## 注意事项

1. **单次传输**: 系统设计为单次传输，避免并发操作
2. **传输间隔**: 默认传输间隔为 5 秒，可在配置中调整
3. **文件校验**: 建议启用文件校验确保数据完整性
4. **错误恢复**: 系统支持传输中断后的恢复机制
5. **资源限制**: 大文件传输需要足够的内存和存储空间