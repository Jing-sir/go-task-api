#!/usr/bin/env bash
# 启用仓库里的 git 钩子。克隆仓库后执行一次。
set -euo pipefail

cd "$(dirname "$0")/.."
git config core.hooksPath .githooks

echo "已启用 .githooks 下的钩子："
ls -1 .githooks
echo
echo "之后每次 git push 会自动跑 scripts/check.sh。"
