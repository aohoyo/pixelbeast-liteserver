#!/bin/bash
# PixelBeast 开发模式启动脚本
# 支持 Go 后端热重载 + 前端静态资源热更新

export GOPROXY=https://goproxy.cn,direct
export GOROOT=/home/wwlhlf/.local/go
export GOPATH=/home/wwlhlf/go
export PATH=$GOROOT/bin:$GOPATH/bin:$PATH

# 清理旧进程
pkill -9 -f "air|pixelbeast" 2>/dev/null && sleep 1

# 检查依赖
command -v go &> /dev/null || { echo "Go 未找到，请检查 GOROOT: $GOROOT"; exit 1; }
command -v air &> /dev/null || { echo "安装 air..."; go install github.com/air-verse/air@latest || exit 1; }

mkdir -p log

# 读取端口配置
ADMIN_PORT=$(grep -o '"admin_port":[[:space:]]*[0-9]*' config/server.json 2>/dev/null | grep -o '[0-9]*$')
HTTP_PORT=$(grep -o '"http_port":[[:space:]]*[0-9]*' config/server.json 2>/dev/null | grep -o '[0-9]*$')

echo ""
echo "  🚀 PixelBeast Dev"
echo "  管理面板: http://localhost:${ADMIN_PORT:-9527}/admin"
echo "  默认网站: http://localhost:${HTTP_PORT:-3380}"
echo "  按 Ctrl+C 停止"
echo ""

trap "echo ''; echo '🛑 停止服务...'; pkill -9 -f 'air|pixelbeast' 2>/dev/null; exit 0" INT TERM

air
