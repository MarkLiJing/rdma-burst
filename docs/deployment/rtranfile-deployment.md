# rtranfile 二进制文件部署指南

## 概述

rtranfile 是 RDMA 文件传输服务的核心工具，程序通过 `exec.Command` 调用外部 rtranfile 二进制文件来实现文件传输功能。本文档适用于单服务和多服务环境部署。

## 程序如何识别和使用 rtranfile

### 调用流程
1. **程序启动** → 调用 `getRtranfilePath()` 获取 rtranfile 路径
2. **创建传输服务** → 使用获取的路径初始化 `RtranfileWrapper`
3. **执行传输** → 通过 `exec.Command(w.binPath, args...)` 调用 rtranfile

### 路径查找优先级
程序按以下顺序查找 rtranfile：

1. **环境变量** `RTRANFILE_PATH`（最高优先级）
2. **系统路径** `/usr/local/bin/rtranfile`
3. **本地路径** `./bin/rtranfile`（当前目录下的 bin 目录）
4. **PATH 查找** 在系统 PATH 中查找 `rtranfile`
5. **默认路径** `./bin/rtranfile`（兼容旧版本）

## 多服务环境部署

### 场景1：单服务部署（简单环境）
- 单个 RDMA 文件传输服务
- 使用系统级部署或应用专用部署
- 所有传输任务使用同一个 rtranfile 实例

### 场景2：多服务部署（复杂环境）
- 多个独立的 RDMA 文件传输服务实例
- 每个服务可能运行在不同服务器或容器中
- 需要确保每个服务都能正确找到 rtranfile

### 多服务部署策略

#### 策略1：共享系统级部署
```bash
# 所有服务共享同一个 rtranfile
sudo cp rtranfile /usr/local/bin/
sudo chmod +x /usr/local/bin/rtranfile

# 每个服务都会自动找到 /usr/local/bin/rtranfile
```

#### 策略2：环境变量隔离部署
```bash
# 服务1
export RTRANFILE_PATH="/opt/service1/bin/rtranfile"
./service1 --mode server

# 服务2
export RTRANFILE_PATH="/opt/service2/bin/rtranfile"
./service2 --mode server
```

#### 策略3：容器化部署
```dockerfile
# 每个容器包含独立的 rtranfile
COPY rtranfile /app/bin/rtranfile
RUN chmod +x /app/bin/rtranfile

# 在容器内使用相对路径
CMD ["./rdma-burst", "--mode", "server"]
```

## 部署方案

### 方案1：系统级部署（推荐用于生产环境）

```bash
# 1. 将 rtranfile 复制到系统路径
sudo cp rtranfile /usr/local/bin/
sudo chmod +x /usr/local/bin/rtranfile

# 2. 验证安装
which rtranfile
# 应该输出: /usr/local/bin/rtranfile

rtranfile --help  # 如果支持帮助命令
```

**优势：**
- 系统范围内可用
- 符合 Linux 标准目录结构
- 易于管理

### 方案2：应用专用部署（推荐用于测试环境）

```bash
# 1. 在应用目录下创建 bin 目录
mkdir -p /path/to/rdma-burst/bin

# 2. 复制 rtranfile
cp rtranfile /path/to/rdma-burst/bin/
chmod +x /path/to/rdma-burst/bin/rtranfile

# 3. 确保从应用目录运行程序
cd /path/to/rdma-burst
./build/rdma-burst --mode server --config configs/combined.yaml
```

### 方案3：环境变量部署（灵活部署）

```bash
# 1. 将 rtranfile 放在任意位置
cp rtranfile /opt/rdma-tools/
chmod +x /opt/rdma-tools/rtranfile

# 2. 设置环境变量
export RTRANFILE_PATH=/opt/rdma-tools/rtranfile

# 3. 运行程序（可以从任何目录）
./build/rdma-burst --mode server --config configs/combined.yaml
```

## 环境特定配置

### 测试环境配置
```bash
# 使用应用专用部署
mkdir -p /root/lj/rdma-burst/bin
cp rtranfile /root/lj/rdma-burst/bin/
chmod +x /root/lj/rdma-burst/bin/rtranfile
```

### 生产环境配置
```bash
# 使用系统级部署
sudo cp rtranfile /usr/local/bin/
sudo chmod +x /usr/local/bin/rtranfile

# 或者使用环境变量（在 systemd 服务文件中设置）
echo 'Environment="RTRANFILE_PATH=/opt/rdma-burst/bin/rtranfile"' >> /etc/systemd/system/rdma-burst.service
```

### Docker 环境配置
```dockerfile
# 在 Dockerfile 中
COPY rtranfile /usr/local/bin/rtranfile
RUN chmod +x /usr/local/bin/rtranfile
```

## 验证部署

### 1. 验证 rtranfile 可访问性
```bash
# 检查文件是否存在和可执行
ls -la /usr/local/bin/rtranfile
# 应该显示: -rwxr-xr-x

# 测试执行（如果支持）
/usr/local/bin/rtranfile --version  # 或 --help
```

### 2. 验证程序路径查找
```bash
# 测试路径查找功能（可以添加临时日志）
export RTRANFILE_PATH="/custom/path/rtranfile"
./build/rdma-burst --version
```

### 3. 完整功能测试
```bash
# 启动服务
./build/rdma-burst --mode server --config configs/combined.yaml

# 在另一个终端测试传输
curl -X POST http://localhost:8080/api/v1/transfers \
  -H "Content-Type: application/json" \
  -d '{
    "filename": "/tmp/testfile",
    "mode": "filesystem", 
    "direction": "put",
    "server_ip": "localhost"
  }'
```

## 故障排除

### 常见问题

**问题1：rtranfile 未找到**
```
错误: 启动传输进程失败: exec: "./bin/rtranfile": stat ./bin/rtranfile: no such file or directory
```

**解决方案：**
- 检查 rtranfile 文件是否存在
- 验证部署路径是否正确
- 使用环境变量指定路径

**问题2：权限不足**
```
错误: 启动传输进程失败: exec: "./bin/rtranfile": permission denied
```

**解决方案：**
```bash
chmod +x /path/to/rtranfile
```

**问题3：rtranfile 不可执行**
```
错误: 传输进程异常退出
```

**解决方案：**
- 验证 rtranfile 是否针对当前系统架构编译
- 检查依赖库是否齐全

**问题4：配置解析错误**
```
错误: 连接超时必须大于 0
错误: 日志文件路径不能为空
错误: 最大并行传输数必须大于 0
```

**解决方案：**
- 配置解析已修复，支持时间字符串格式（如 "30s", "5m"）
- 统一配置文件中的字段映射已修复
- 使用测试脚本验证配置解析：
```bash
# 测试配置解析
go run scripts/test-config-parsing.go

# 测试客户端模式
./scripts/test-client-mode.sh
```

### 调试方法

1. **启用详细日志**
```bash
export RTRANFILE_DEBUG=1
./build/rdma-burst --mode server
```

2. **检查系统 PATH**
```bash
echo $PATH
which rtranfile
```

3. **手动测试 rtranfile**
```bash
# 模拟程序调用
/path/to/rtranfile -d mlx5_0 --dir /tmp --logfile /tmp/test.log
```

## 最佳实践

1. **生产环境**：使用系统级部署 (`/usr/local/bin/rtranfile`)
2. **测试环境**：使用应用专用部署 (`./bin/rtranfile`)  
3. **多环境部署**：使用环境变量 (`RTRANFILE_PATH`)
4. **版本控制**：为不同版本的 rtranfile 创建符号链接
5. **权限管理**：确保 rtranfile 有适当的执行权限

## 更新和维护

### 更新 rtranfile
```bash
# 备份旧版本
sudo cp /usr/local/bin/rtranfile /usr/local/bin/rtranfile.backup

# 安装新版本
sudo cp new-rtranfile /usr/local/bin/rtranfile
sudo chmod +x /usr/local/bin/rtranfile

# 重启服务
sudo systemctl restart rdma-burst
```

### 多版本管理
```bash
# 使用版本化命名
sudo cp rtranfile-v1.2.3 /usr/local/bin/
sudo ln -sf /usr/local/bin/rtranfile-v1.2.3 /usr/local/bin/rtranfile
```

通过遵循本指南，您可以确保 rtranfile 在各种环境中正确部署并被程序识别使用。