#!/usr/bin/env bash
set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

status=0

section() {
  printf '\n== %s ==\n' "$1"
}

ok() {
  printf '[OK] %s\n' "$1"
}

warn() {
  printf '[WARN] %s\n' "$1"
}

fail() {
  printf '[FAIL] %s\n' "$1"
  status=1
}

command_version() {
  local name="$1"
  local version_cmd="$2"
  if command -v "$name" >/dev/null 2>&1; then
    ok "$name: $($version_cmd 2>/dev/null | head -n 1)"
  else
    fail "缺少命令：$name"
  fi
}

check_port() {
  local port="$1"
  local label="$2"
  local output
  output="$(lsof -nP -iTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)"
  if [ -n "$output" ]; then
    warn "${label} 端口 ${port} 已被占用"
    printf '%s\n' "$output"
  else
    ok "${label} 端口 ${port} 空闲"
  fi
}

check_env_key() {
  local env_file="$1"
  local key="$2"
  if grep -Eq "^${key}=" "$env_file"; then
    ok ".env.local 已配置 ${key}"
  else
    fail ".env.local 缺少 ${key}"
  fi
}

env_value() {
  local env_file="$1"
  local key="$2"
  grep -E "^${key}=" "$env_file" | tail -n 1 | cut -d '=' -f 2-
}

check_non_empty_env_key() {
  local env_file="$1"
  local key="$2"
  local value
  value="$(env_value "$env_file" "$key")"
  if [ -n "$value" ]; then
    ok ".env.local 已配置 ${key}"
  else
    fail ".env.local 缺少 ${key} 或值为空"
  fi
}

section "路径"
ok "项目目录：${PROJECT_ROOT}"

section "依赖版本"
command_version "node" "node --version"
command_version "npm" "npm --version"
command_version "go" "go version"
command_version "python3" "python3 --version"
if command -v docker >/dev/null 2>&1; then
  ok "docker: $(docker --version 2>/dev/null | head -n 1)"
else
  warn "缺少 docker：无法启动 MySQL、Redis、MinIO 中间件"
fi

section "端口占用"
check_port 5173 "Web Vite"
check_port 8080 "Go API"
check_port 3306 "MySQL"
check_port 6379 "Redis"
check_port 9000 "MinIO API"
check_port 9001 "MinIO Console"

section "配置文件"
if [ -f "${PROJECT_ROOT}/.env.example" ]; then
  ok "找到 .env.example"
else
  fail "缺少 .env.example"
fi

if [ -f "${PROJECT_ROOT}/.env.local" ]; then
  ok "找到 .env.local"
  for key in APP_ENV PORT MYSQL_HOST MYSQL_PORT MYSQL_DATABASE MYSQL_USER MYSQL_PASSWORD REDIS_ADDR; do
    check_env_key "${PROJECT_ROOT}/.env.local" "$key"
  done
  storage_driver="$(env_value "${PROJECT_ROOT}/.env.local" "STORAGE_DRIVER")"
  case "$storage_driver" in
    obs)
      ok "对象存储驱动：obs"
      for key in OBS_ENDPOINT OBS_ACCESS_KEY_ID OBS_SECRET_ACCESS_KEY OBS_BUCKET OBS_REGION; do
        check_non_empty_env_key "${PROJECT_ROOT}/.env.local" "$key"
      done
      ;;
    minio|"")
      ok "对象存储驱动：${storage_driver:-minio}"
      for key in MINIO_ENDPOINT MINIO_ACCESS_KEY MINIO_SECRET_KEY MINIO_BUCKET; do
        check_non_empty_env_key "${PROJECT_ROOT}/.env.local" "$key"
      done
      ;;
    *)
      fail "不支持的 STORAGE_DRIVER：${storage_driver}"
      ;;
  esac
else
  fail "缺少 .env.local；可执行：cp .env.example .env.local"
fi

section "项目依赖"
if [ -d "${PROJECT_ROOT}/apps/web/node_modules" ]; then
  ok "Web node_modules 已安装"
else
  warn "Web 依赖未安装；可执行：cd apps/web && npm ci"
fi

if [ -f "${PROJECT_ROOT}/services/api/go.sum" ]; then
  ok "Go module 依赖锁文件存在"
else
  fail "缺少 services/api/go.sum"
fi

if command -v docker >/dev/null 2>&1; then
  section "Docker Compose 配置"
  if docker compose -f "${PROJECT_ROOT}/infra/docker-compose.yml" config >/dev/null 2>&1; then
    ok "infra/docker-compose.yml 配置可解析"
  else
    fail "infra/docker-compose.yml 配置解析失败"
  fi
fi

section "结论"
if [ "$status" -eq 0 ]; then
  ok "基础诊断通过；如服务仍无法启动，请优先查看上面的端口占用和中间件状态。"
else
  fail "诊断发现阻塞项；请先修复 [FAIL] 项。"
fi

exit "$status"
