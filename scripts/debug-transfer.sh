#!/bin/bash

# 调试 RDMA 文件传输服务

echo "=== RDMA 文件传输服务调试 ==="

# 1. 检查服务端状态
echo "1. 检查服务端状态..."
curl -v http://localhost:8080/api/health 2>&1 | grep -E "(HTTP|{.*})"

# 2. 检查客户端状态
echo "2. 检查客户端状态..."
curl -v http://localhost:8081/api/health 2>&1 | grep -E "(HTTP|{.*})"

# 3. 检查服务端模式
echo "3. 检查服务端模式..."
curl -v http://localhost:8080/api/v1/mode 2>&1 | grep -E "(HTTP|{.*})"

# 4. 检查客户端模式
echo "4. 检查客户端模式..."
curl -v http://localhost:8081/api/v1/mode 2>&1 | grep -E "(HTTP|{.*})"

# 5. 测试创建传输任务（详细输出）
echo "5. 测试创建传输任务..."
curl -v -X POST http://localhost:8081/api/v1/transfers \
  -H "Content-Type: application/json" \
  -d '{
    "filename": "/tmp/testfile.bin",
    "mode": "filesystem",
    "direction": "put",
    "server_ip": "localhost"
  }' 2>&1 | grep -E "(HTTP|{.*}|error|Error)"

# 6. 检查活跃传输任务
echo "6. 检查活跃传输任务..."
curl -v http://localhost:8081/api/v1/transfers/active 2>&1 | grep -E "(HTTP|{.*})"

# 7. 检查传输任务列表
echo "7. 检查传输任务列表..."
curl -v http://localhost:8081/api/v1/transfers 2>&1 | grep -E "(HTTP|{.*})"

echo "=== 调试完成 ==="