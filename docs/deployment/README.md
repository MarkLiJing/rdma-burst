# 部署指南

RDMA 大文件传输服务的生产环境部署指南。

## 部署架构

### 单机部署
适用于开发和测试环境。

```
[客户端] ←→ [RDMA 传输服务] ←→ [存储系统]
```

### 集群部署  
适用于生产环境高可用部署。

```
[负载均衡器]
    ↓
[RDMA 传输服务集群] ←→ [共享存储]
    ↓  
[监控系统]
```

## 系统要求

### 硬件要求

| 组件 | 最低要求 | 推荐配置 |
|------|----------|----------|
| CPU | 4 核心 | 8+ 核心 |
| 内存 | 8GB | 32GB+ |
| 存储 | 100GB | 1TB+ SSD |
| 网络 | 10GbE | 100GbE RDMA |

### 软件要求

- **操作系统**: Linux (Ubuntu 20.04+, CentOS 8+, RHEL 8+)
- **Go 版本**: 1.21+
- **RDMA 驱动**: Mellanox OFED 或内置驱动
- **容器运行时**: Docker 20.10+ (可选)

## 环境准备

### 1. RDMA 环境配置

```bash
# 检查 RDMA 设备
ibv_devices

# 安装 RDMA 工具
# Ubuntu/Debian
sudo apt update
sudo apt install rdma-core ibverbs-utils infiniband-diags

# CentOS/RHEL
sudo yum install rdma-core ibutils infiniband-diags

# 验证 RDMA 功能
ibv_rc_pingpong
```

### 2. 大页内存配置

```bash
# 配置 2MB 大页内存
echo 1024 > /sys/kernel/mm/hugepages/hugepages-2048kB/nr_hugepages

# 配置 1GB 大页内存（可选）
echo 4 > /sys/kernel/mm/hugepages/hugepages-1048576kB/nr_hugepages

# 创建挂载点
mkdir -p /dev/hugepages
mount -t hugetlbfs nodev /dev/hugepages

# 永久配置（添加到 /etc/fstab）
echo "nodev /dev/hugepages hugetlbfs pagesize=2MB 0 0" >> /etc/fstab
```

### 3. tmpfs 配置

```bash
# 创建 tmpfs 目录
mkdir -p /dev/shm/rtrans
mount -t tmpfs -o size=10G tmpfs /dev/shm/rtrans

# 永久配置（可选）
echo "tmpfs /dev/shm/rtrans tmpfs size=10G 0 0" >> /etc/fstab
```

## 部署方式

### 方式一：二进制部署

#### 1. 下载或构建二进制文件

```bash
# 从发布页面下载
wget https://github.com/your-org/rdma-burst/releases/latest/download/rdma-burst-linux-amd64.tar.gz
tar -xzf rdma-burst-linux-amd64.tar.gz

# 或从源码构建
git clone https://github.com/your-org/rdma-burst.git
cd rdma-burst
go build -o bin/server cmd/server/main.go
go build -o bin/client cmd/client/main.go
```

#### 2. 准备 rtranfile 工具

```bash
# 复制 rtranfile 到项目目录
cp /path/to/rtranfile ./bin/
chmod +x ./bin/rtranfile
```

#### 3. 配置系统服务

创建 systemd 服务文件 `/etc/systemd/system/rdma-burst.service`:

```ini
[Unit]
Description=RDMA Burst File Transfer Service
After=network.target

[Service]
Type=simple
User=rdma
Group=rdma
WorkingDirectory=/opt/rdma-burst
ExecStart=/opt/rdma-burst/bin/server --config /etc/rdma-burst/server.yaml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

# RDMA 相关权限
CapabilityBoundingSet=CAP_NET_RAW CAP_SYS_ADMIN
DeviceAllow=/dev/infiniband/ rw

# 资源限制
LimitMEMLOCK=infinity
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

#### 4. 创建系统用户和目录

```bash
# 创建系统用户
sudo useradd -r -s /bin/false rdma

# 创建数据目录
sudo mkdir -p /opt/rdma-burst/{bin,configs,logs}
sudo mkdir -p /var/lib/rtrans
sudo mkdir -p /var/log/rtrans

# 设置权限
sudo chown -R rdma:rdma /opt/rdma-burst
sudo chown -R rdma:rdma /var/lib/rtrans
sudo chown -R rdma:rdma /var/log/rtrans
```

#### 5. 启动服务

```bash
# 重载 systemd
sudo systemctl daemon-reload

# 启用服务
sudo systemctl enable rdma-burst

# 启动服务
sudo systemctl start rdma-burst

# 检查状态
sudo systemctl status rdma-burst
```

### 方式二：Docker 部署

#### 1. 创建 Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o /server cmd/server/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates
RUN mkdir -p /dev/hugepages

# 创建非 root 用户
RUN addgroup -S rdma && adduser -S rdma -G rdma

WORKDIR /root/
COPY --from=builder /server .
COPY configs/server.yaml .

# 复制 rtranfile（需要提前放入构建上下文）
COPY bin/rtranfile /usr/local/bin/
RUN chmod +x /usr/local/bin/rtranfile

USER rdma

EXPOSE 8080

CMD ["./server", "--config", "server.yaml"]
```

#### 2. 构建镜像

```bash
docker build -t rdma-burst:latest .
```

#### 3. 运行容器

```bash
docker run -d \
  --name rdma-burst \
  --privileged \
  --network=host \
  -p 8080:8080 \
  -v /dev/hugepages:/dev/hugepages \
  -v /var/lib/rtrans:/var/lib/rtrans \
  -v /var/log/rtrans:/var/log/rtrans \
  rdma-burst:latest
```

#### 4. Docker Compose 部署

创建 `docker-compose.yml`:

```yaml
version: '3.8'

services:
  rdma-burst:
    image: rdma-burst:latest
    container_name: rdma-burst
    privileged: true
    network_mode: host
    ports:
      - "8080:8080"
    volumes:
      - /dev/hugepages:/dev/hugepages
      - ./data:/var/lib/rtrans
      - ./logs:/var/log/rtrans
    environment:
      - RDMA_DEVICE=mlx5_0
    restart: unless-stopped
```

## 配置管理

### 生产环境配置

创建生产环境配置文件 `/etc/rdma-burst/server.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  log_level: "info"

transfer:
  device: "mlx5_0"
  base_dir: "/var/lib/rtrans"
  transfer_interval: "5s"
  max_concurrent_transfers: 1

logging:
  file_path: "/var/log/rtrans/rtrans_server.log"
  max_size: 100
  max_backups: 10
  max_age: 30
  format: "json"

monitoring:
  health_check_interval: "30s"
  enable_metrics: true
  metrics_port: 9090
```

### 环境变量配置

支持通过环境变量覆盖配置：

```bash
export RDMA_BURST_SERVER_PORT=8080
export RDMA_BURST_TRANSFER_DEVICE=mlx5_0
export RDMA_BURST_LOGGING_LEVEL=info
```

## 监控和日志

### 系统监控

配置 Prometheus 监控：

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'rdma-burst'
    static_configs:
      - targets: ['localhost:9090']
    metrics_path: '/metrics'
```

### 日志管理

使用 logrotate 管理日志文件：

```bash
# /etc/logrotate.d/rdma-burst
/var/log/rtrans/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 644 rdma rdma
    postrotate
        systemctl reload rdma-burst
    endscript
}
```

### 健康检查

```bash
# 健康检查脚本
#!/bin/bash
curl -f http://localhost:8080/api/health || exit 1
```

## 安全配置

### 网络安全

```bash
# 防火墙配置
sudo ufw allow 8080/tcp
sudo ufw allow from 192.168.1.0/24 to any port 8080

# 或使用 iptables
sudo iptables -A INPUT -p tcp --dport 8080 -j ACCEPT
```

### TLS 配置（可选）

```yaml
server:
  tls:
    enabled: true
    cert_file: "/etc/ssl/certs/rdma-burst.crt"
    key_file: "/etc/ssl/private/rdma-burst.key"
```

## 性能优化

### 内核参数优化

```bash
# /etc/sysctl.d/99-rdma-optimization.conf

# 网络优化
net.core.rmem_max = 134217728
net.core.wmem_max = 134217728
net.ipv4.tcp_rmem = 4096 87380 134217728
net.ipv4.tcp_wmem = 4096 65536 134217728

# 内存优化
vm.swappiness = 10
vm.dirty_ratio = 10
vm.dirty_background_ratio = 5

# 文件系统优化
vm.vfs_cache_pressure = 50
```

### RDMA 优化

```bash
# 调整 RDMA 参数
echo 65536 > /sys/class/infiniband/mlx5_0/ports/1/counters/port_rcv_data
echo 65536 > /sys/class/infiniband/mlx5_0/ports/1/counters/port_xmit_data
```

## 故障排除

### 常见问题

1. **RDMA 设备不可用**
   ```bash
   # 检查设备状态
   ibstatus
   ibv_devices
   
   # 重新加载驱动
   modprobe -r mlx5_core
   modprobe mlx5_core
   ```

2. **大页内存分配失败**
   ```bash
   # 检查内存状态
   cat /proc/meminfo | grep Huge
   
   # 释放缓存
   echo 3 > /proc/sys/vm/drop_caches
   ```

3. **权限问题**
   ```bash
   # 检查用户权限
   id rdma
   
   # 检查目录权限
   ls -la /dev/hugepages/
   ls -la /var/lib/rtrans/
   ```

### 日志分析

```bash
# 查看服务日志
journalctl -u rdma-burst -f

# 查看应用日志
tail -f /var/log/rtrans/rtrans_server.log

# 查看 rtranfile 日志
tail -f /var/log/rtrans/rtrans_*.log
```

## 备份和恢复

### 数据备份

```bash
# 备份传输任务数据
tar -czf rdma-burst-backup-$(date +%Y%m%d).tar.gz /var/lib/rtrans/

# 备份配置
cp /etc/rdma-burst/server.yaml /backup/server-$(date +%Y%m%d).yaml
```

### 灾难恢复

1. 恢复配置文件
2. 恢复数据目录
3. 重新启动服务
4. 验证服务状态

## 升级流程

1. 停止当前服务
2. 备份数据和配置
3. 部署新版本
4. 验证功能
5. 启动服务

## 支持联系方式

- **文档**: https://github.com/your-org/rdma-burst/docs
- **问题反馈**: https://github.com/your-org/rdma-burst/issues
- **社区支持**: community@example.com