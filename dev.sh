#!/bin/bash
# PixelBeast 开发模式启动脚本
# 支持 Go 后端热重载 + 前端静态资源热更新

# 配置 Go 国内代理（解决网络问题）
export GOPROXY=https://goproxy.cn,direct

# 配置 Go 路径
export GOROOT=/home/wwlhlf/.local/go
export GOPATH=/home/wwlhlf/go
export PATH=$GOROOT/bin:$GOPATH/bin:$PATH

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  🚀 PixelBeast 开发模式"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 清理旧进程（避免冲突）
if pgrep -f "air|pixelbeast" > /dev/null; then
    echo "  🧹 检测到旧进程，正在清理..."
    pkill -9 -f "air|pixelbeast" 2>/dev/null
    sleep 1
    echo "  ✅ 旧进程已清理"
    echo ""
fi

# 检查 go 是否安装
if ! command -v go &> /dev/null; then
    echo "❌ Go 未找到，请检查 GOROOT: $GOROOT"
    exit 1
fi

# 检查 air 是否安装
if ! command -v air &> /dev/null; then
    echo "📦 正在安装 air..."
    go install github.com/air-verse/air@latest
    if [ $? -eq 0 ]; then
        echo "✅ air 安装完成"
    else
        echo "❌ air 安装失败"
        exit 1
    fi
fi

# 确保 logs 目录存在
mkdir -p logs

echo "  ✅ 热重载: air"
echo "  ✅ 前端热更新: src/static/admin/"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  🎯 启动中..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 启动开发模式
air &
AIR_PID=$!

# 等待服务器启动
sleep 3

# 显示访问地址
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  🌐 访问地址"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "  管理面板: http://localhost:9527/admin"
echo "  默认网站: http://localhost:8080"
echo ""
echo "  默认凭据: admin / admin123"
echo ""
echo "  按 Ctrl+C 停止服务"
echo ""

# 捕获 Ctrl+C 信号，确保清理
trap "echo ''; echo '🛑 正在停止服务...'; pkill -9 -f 'air|pixelbeast' 2>/dev/null; echo '✅ 服务已停止'; exit 0" INT TERM

# 等待 air 进程
wait $AIR_PID
