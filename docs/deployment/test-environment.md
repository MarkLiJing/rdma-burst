# 测试环境部署指南

## 环境要求

### 硬件要求
- Linux 操作系统（Ubuntu 20.04+ 或 CentOS 8+）
- 支持 RDMA 的网络设备（mlx5_0）
- 至少 8GB 内存
- 50GB+ 可用磁盘空间

### 软件要求
- Go 1.21+ 环境
- RDMA 驱动已安装
- 大页内存配置（可选，用于高性能传输）

## 部署步骤

### 1. 环境准备

```bash
# 安装 Go 环境（如果未安装）
wget https://golang.org/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# 安装 RDMA 驱动（Ubuntu）
sudo apt update
sudo apt install rdma-core ibverbs-utils

# 配置大页内存（可选）
echo 1024 > /sys/kernel/mm/hugepages/hugepages-2048kB/nr_hugepages
mkdir -p /dev/hugepages/dir
mount -t hugetlbfs nodev /dev/hugepages/dir
```

### 2. 获取项目代码

```bash
# 克隆项目（如果使用 Git）
git clone <repository-url>
cd rdma-burst

# 或者直接下载代码
# 将项目文件上传到测试环境
```

### 3. 构建项目

```bash
# 安装依赖
make deps

# 构建所有目标
make build

# 验证构建结果
ls -la build/
# 应该看到三个可执行文件：server, client, rdma-burst
```

### 4. 准备配置文件

```bash
# 创建配置目录
sudo mkdir -p /etc/rtrans
sudo mkdir -p /var/log/rtrans
sudo mkdir -p /var/lib/rtrans

# 复制配置文件
sudo cp configs/combined.yaml /etc/rtrans/
sudo cp configs/server.yaml /etc/rtrans/
sudo cp configs/client.yaml /etc/rtrans/

# 设置权限
sudo chown -R $USER:$USER /var/log/rtrans /var/lib/rtrans
```

### 5. 部署服务端

#### 方法一：使用 systemd 服务（推荐）

```bash
# 创建 systemd 服务文件
sudo tee /etc/systemd/system/rdma-burst.service > /dev/null <<EOF
[Unit]
Description=RDMA Burst File Transfer Service
After=network.target

[Service]
Type=simple
User=$USER
WorkingDirectory=/root/lj/rdma-burst
ExecStart=/root/lj/rdma-burst/build/rdma-burst --mode server --config /etc/rtrans/combined.yaml
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF

# 启动服务
sudo systemctl daemon-reload
sudo systemctl enable rdma-burst
sudo systemctl start rdma-burst

# 检查服务状态
sudo systemctl status rdma-burst
```

#### 方法二：直接运行

```bash
# 后台运行服务端
nohup ./build/rdma-burst --mode server --config /etc/rtrans/combined.yaml > /var/log/rtrans/server.log 2>&1 &

# 检查是否运行成功
ps aux | grep rdma-burst
netstat -tlnp | grep 8080
```

### 6. 验证部署

```bash
# 检查服务健康状态
curl http://localhost:8080/api/health

# 检查运行模式
curl http://localhost:8080/api/v1/mode

# 检查服务端是否正常响应
curl http://localhost:8080/
```

### 7. 部署客户端

客户端不需要常驻运行，按需使用：

```bash
# 直接运行客户端命令
./build/rdma-burst --mode client --config /etc/rtrans/combined.yaml

# 或者使用统一可执行文件的自动检测模式
./build/rdma-burst --mode auto --config /etc/rtrans/combined.yaml
```

## 测试验证流程

### 1. 基础功能测试

```bash
# 测试1: 服务端健康检查
curl -s http://localhost:8080/api/health | jq .

# 测试2: 模式检测API
curl -s http://localhost:8080/api/v1/mode | jq .
curl -s http://localhost:8080/api/v1/mode/detect | jq .

# 测试3: 传输服务状态
curl -s http://localhost:8080/api/v1/transfers/active | jq .
```

### 2. 文件传输测试

```bash
# 准备测试文件
dd if=/dev/zero of=/tmp/testfile.bin bs=1M count=100

# 创建传输任务（需要先配置客户端）
./build/client transfer /tmp/testfile.bin filesystem put localhost

# 或者通过API创建传输任务
curl -X POST http://localhost:8080/api/v1/transfers \
  -H "Content-Type: application/json" \
  -d '{
    "filename": "/tmp/testfile.bin",
    "mode": "filesystem",
    "direction": "put",
    "server_ip": "localhost"
  }'
```

### 3. 性能测试

```bash
# 创建大文件进行性能测试
dd if=/dev/zero of=/tmp/largefile.bin bs=1G count=5

# 测试不同传输模式的性能
# 1. 文件系统模式（基础性能）
# 2. tmpfs模式（内存文件系统）
# 3. 大页内存模式（最高性能）
```

## 监控和日志

### 日志配置

```bash
# 查看服务端日志
tail -f /var/log/rtrans/server.log

# 查看客户端日志
tail -f /var/log/rtrans/client.log

# 查看传输任务日志
ls -la /var/log/rtrans/rtrans_*.log
```

### 监控指标

```bash
# 获取服务指标
curl http://localhost:8080/api/metrics

# 获取传输统计
curl http://localhost:8080/api/v1/transfers

# 检查活跃连接
curl http://localhost:8080/api/v1/transfers/active
```

## 故障排除

### 常见问题

1. **端口占用**: 检查8080端口是否被其他进程占用
2. **权限问题**: 确保对日志和配置目录有写权限
3. **RDMA设备**: 验证mlx5_0设备是否存在且可用
4. **内存配置**: 检查大页内存是否配置正确

### 调试命令

```bash
# 检查RDMA设备
ibv_devices

# 检查网络连接
ibstatus

# 检查系统资源
free -h
df -h

# 检查进程状态
ps aux | grep rdma-burst
netstat -tlnp | grep 8080
```

## 清理和卸载

```bash
# 停止服务
sudo systemctl stop rdma-burst
sudo systemctl disable rdma-burst

# 删除服务文件
sudo rm /etc/systemd/system/rdma-burst.service

# 清理日志和临时文件
sudo rm -rf /var/log/rtrans/*
sudo rm -rf /var/lib/rtrans/*

# 清理构建文件
make clean
```

这个部署指南提供了完整的测试环境部署流程，包括环境准备、服务部署、功能验证和故障排除。