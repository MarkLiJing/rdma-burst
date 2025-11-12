#!/bin/bash

# 调试服务端进程启动脚本

set -e

echo "=== 调试 rtranfile 服务端进程启动 ==="

# 检查 rtranfile 是否存在
if [ ! -f "/usr/local/bin/rtranfile" ]; then
    echo "错误: rtranfile 未找到"
    exit 1
fi

echo "rtranfile 路径: /usr/local/bin/rtranfile"

# 创建日志目录
mkdir -p /var/log/rtrans

# 测试1: 直接运行 rtranfile 服务端（不带 -l 参数）
echo ""
echo "=== 测试1: 直接运行 rtranfile 服务端（不带 -l 参数）==="
LOG_FILE1="/var/log/rtrans/debug_test1.log"
echo "日志文件: $LOG_FILE1"

timeout 5s /usr/local/bin/rtranfile -d mlx5_0 --dir /var/lib/rtrans/files --logfile "$LOG_FILE1" &
PID1=$!
echo "进程PID: $PID1"

# 等待2秒
sleep 2

# 检查进程状态
if ps -p $PID1 > /dev/null; then
    echo "测试1: 进程仍在运行"
    kill $PID1 2>/dev/null || true
else
    echo "测试1: 进程已退出"
    # 检查退出码
    wait $PID1 2>/dev/null && EXIT_CODE=$? || EXIT_CODE=$?
    echo "退出码: $EXIT_CODE"
fi

# 显示日志
echo "测试1日志内容:"
cat "$LOG_FILE1" 2>/dev/null || echo "无日志内容"

# 测试2: 运行 rtranfile 服务端（带 -l 参数）
echo ""
echo "=== 测试2: 运行 rtranfile 服务端（带 -l 参数）==="
LOG_FILE2="/var/log/rtrans/debug_test2.log"
echo "日志文件: $LOG_FILE2"

timeout 5s /usr/local/bin/rtranfile -d mlx5_0 --dir /var/lib/rtrans/files -l --logfile "$LOG_FILE2" &
PID2=$!
echo "进程PID: $PID2"

# 等待2秒
sleep 2

# 检查进程状态
if ps -p $PID2 > /dev/null; then
    echo "测试2: 进程仍在运行"
    kill $PID2 2>/dev/null || true
else
    echo "测试2: 进程已退出"
    # 检查退出码
    wait $PID2 2>/dev/null && EXIT_CODE=$? || EXIT_CODE=$?
    echo "退出码: $EXIT_CODE"
fi

# 显示日志
echo "测试2日志内容:"
cat "$LOG_FILE2" 2>/dev/null || echo "无日志内容"

# 测试3: 检查设备可用性
echo ""
echo "=== 测试3: 检查 RDMA 设备状态 ==="
if command -v ibstat >/dev/null 2>&1; then
    echo "ibstat 输出:"
    ibstat mlx5_0 2>/dev/null || echo "无法获取 mlx5_0 设备状态"
else
    echo "ibstat 命令不可用"
fi

if command -v ibv_devices >/dev/null 2>&1; then
    echo "ibv_devices 输出:"
    ibv_devices 2>/dev/null || echo "无法列出 RDMA 设备"
else
    echo "ibv_devices 命令不可用"
fi

echo ""
echo "=== 调试完成 ==="