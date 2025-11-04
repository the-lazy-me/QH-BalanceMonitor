#!/bin/bash

# 余额监控系统停止脚本

echo "正在停止余额监控系统..."

# 从 PID 文件读取并停止
if [ -f ".backend.pid" ]; then
    BACKEND_PID=$(cat .backend.pid)
    kill $BACKEND_PID 2>/dev/null && echo "✓ 后端服务已停止 (PID: $BACKEND_PID)" || echo "后端服务未运行"
    rm -f .backend.pid
fi

if [ -f ".frontend.pid" ]; then
    FRONTEND_PID=$(cat .frontend.pid)
    kill $FRONTEND_PID 2>/dev/null && echo "✓ 前端服务已停止 (PID: $FRONTEND_PID)" || echo "前端服务未运行"
    rm -f .frontend.pid
fi

# 确保清理所有相关进程
pkill -f "balance-monitor" 2>/dev/null
pkill -f "next dev" 2>/dev/null

echo "所有服务已停止"

