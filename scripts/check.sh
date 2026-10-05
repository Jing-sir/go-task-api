#!/usr/bin/env bash
# 本地检查脚本：把提交前该跑的验证串成一条命令。
#
# 用法：./scripts/check.sh
# pre-push 钩子会自动调用它，也可以随时手动跑。
#
# set -e  任一命令失败立刻退出，不继续往下跑
# set -u  用到未定义变量就报错
# set -o pipefail  管道中间环节失败也算失败（默认只看最后一个命令）
set -euo pipefail

cd "$(dirname "$0")/.."
BACKEND_DIR="backend"

# macOS 上没同意 Xcode 许可协议时，CGO 编译 runtime/cgo 会失败。
# 本项目依赖全是纯 Go，关掉 CGO 既能绕过这个问题，
# 也和 Dockerfile 里的构建方式保持一致。
export CGO_ENABLED=0

red()   { printf '\033[31m%s\033[0m\n' "$1"; }
green() { printf '\033[32m%s\033[0m\n' "$1"; }
blue()  { printf '\033[34m%s\033[0m\n' "$1"; }

step() { blue "▶ $1"; }

cd "$BACKEND_DIR"

step "gofmt 格式检查"
# gofmt -l 列出格式不对的文件。它对格式错误不返回非零退出码，
# 所以要自己判断输出是否为空。
unformatted=$(gofmt -l . | grep -v '^docs/' || true)
if [ -n "$unformatted" ]; then
  red "以下文件格式不对，执行 gofmt -w 修复："
  echo "$unformatted"
  exit 1
fi
green "  格式通过"

step "go build 编译"
go build ./...
green "  编译通过"

step "go vet 静态检查"
go vet ./...
green "  静态检查通过"

step "Swagger 文档是否和代码同步"
# 重新生成一份到临时目录，和仓库里的比对。
# 不一致说明改了注解没重新生成，文档会和代码脱节。
if command -v swag >/dev/null 2>&1; then
  SWAG=swag
elif [ -x "$(go env GOPATH)/bin/swag" ]; then
  SWAG="$(go env GOPATH)/bin/swag"
else
  SWAG=""
fi

if [ -z "$SWAG" ]; then
  red "  未安装 swag，跳过文档同步检查"
  red "  安装：go install github.com/swaggo/swag/cmd/swag@v1.16.4"
else
  tmpdir=$(mktemp -d)
  trap 'rm -rf "$tmpdir"' EXIT
  $SWAG init -g cmd/server/main.go -o "$tmpdir" --parseDependency --parseInternal >/dev/null 2>&1
  # 只比对 swagger.json。docs.go 里带生成时间戳之类的噪音，不适合逐字节比。
  if ! diff -q "$tmpdir/swagger.json" docs/swagger.json >/dev/null 2>&1; then
    red "Swagger 文档和代码不一致。执行下面这条重新生成并提交："
    red "  cd backend && swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal"
    exit 1
  fi
  green "  文档已同步"
fi

step "go test 全部测试"
# 集成测试需要 PostgreSQL。数据库没启动时它们会自动 SKIP 而不是失败，
# 所以这里先提示一下，避免你以为测试都跑过了。
if ! nc -z localhost 5433 >/dev/null 2>&1; then
  red "  提示：localhost:5433 没响应，集成测试会被跳过"
  red "  要完整验证请先执行：docker compose up -d db redis"
fi
go test ./...
green "  测试通过"

echo
green "✅ 全部检查通过"
