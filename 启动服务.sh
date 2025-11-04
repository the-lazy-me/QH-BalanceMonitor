#!/bin/bash

# 余额监控系统启动脚本

echo "================================"
echo "   余额监控系统 - 启动脚本"
echo "================================"
echo ""

# 检查并启动后端
echo "1. 启动后端服务..."
cd backend

# 检查是否已编译
if [ ! -f "balance-monitor" ]; then
    echo "   后端程序未编译，正在编译..."
    go build -o balance-monitor main.go
    if [ $? -ne 0 ]; then
        echo "   ❌ 后端编译失败"
        exit 1
    fi
    echo "   ✓ 后端编译成功"
fi

# 启动后端
./balance-monitor &
BACKEND_PID=$!
echo "   ✓ 后端服务已启动 (PID: $BACKEND_PID)"
echo "   → 后端地址: http://localhost:8080"
cd ..

sleep 2

# 检查并启动前端
echo ""
echo "2. 启动前端服务..."
cd frontend

# 检查 node_modules
if [ ! -d "node_modules" ]; then
    echo "   依赖未安装，正在安装..."
    npm install
    if [ $? -ne 0 ]; then
        echo "   ❌ 依赖安装失败"
        kill $BACKEND_PID
        exit 1
    fi
    echo "   ✓ 依赖安装成功"
fi

# 启动前端
npm run dev &
FRONTEND_PID=$!
echo "   ✓ 前端服务已启动 (PID: $FRONTEND_PID)"
echo "   → 前端地址: http://localhost:3000"
cd ..

echo ""
echo "================================"
echo "   系统启动完成！"
echo "================================"
echo ""
echo "后端服务: http://localhost:8080"
echo "前端服务: http://localhost:3000"
echo ""
echo "首次使用请访问: http://localhost:3000"
echo "系统将引导您完成初始化设置"
echo ""
echo "按 Ctrl+C 停止所有服务"
echo ""

# 保存 PID 到文件
echo $BACKEND_PID > .backend.pid
echo $FRONTEND_PID > .frontend.pid

# 等待 Ctrl+C
trap "echo ''; echo '正在停止服务...'; kill $BACKEND_PID $FRONTEND_PID 2>/dev/null; rm -f .backend.pid .frontend.pid; echo '服务已停止'; exit 0" INT TERM

# 保持脚本运行
wait

