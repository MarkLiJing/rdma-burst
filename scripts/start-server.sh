#!/bin/bash

# RDMA文件传输服务端启动脚本

set -e

echo "=== RDMA文件传输服务端启动脚本 ==="

# 检查构建文件是否存在
if [ ! -f "./build/rdma-burst" ]; then
    echo "错误: 可执行文件不存在，请先运行 'make build'"
    exit 1
fi

# 检查配置文件是否存在
if [ ! -f "./configs/combined.yaml" ]; then
    echo "错误: 配置文件不存在: ./configs/combined.yaml"
    exit 1
fi

# 检查端口是否被占用
PORT=8080
if netstat -tln | grep ":$PORT " > /dev/null; then
    echo "警告: 端口 $PORT 已被占用"
    echo "正在检查占用进程..."
    netstat -tlnp | grep ":$PORT "
    echo "请先停止占用进程或修改配置文件中的端口"
    exit 1
fi

echo "✓ 端口 $PORT 可用"

# 启动服务端
echo "启动 RDMA 文件传输服务端..."
echo "监听地址: 0.0.0.0:$PORT"
echo "配置文件: ./configs/combined.yaml"
echo ""

# 显示启动命令
echo "执行命令:"
echo "./build/rdma-burst --mode combined --config configs/combined.yaml"
echo ""

# 实际启动服务端
./build/rdma-burst --mode combined --config configs/combined.yaml