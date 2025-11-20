#!/bin/bash
set -euo pipefail

# 线上一键部署（Ubuntu）- online-hk 环境（不包含 git 拉取）
# 用法（请先手动将最新仓库代码放到 WORKSPACE，再执行）：
#   WORKSPACE=/opt/wumitech-com \
#   CONFIG_HOST_PATH=/opt/mdcp_server_operator/configs/online-hk.yaml \
#   /bin/bash docker_run_online_hk.sh
#
# 说明：
# - 不进行 git clone/pull；仅根据 WORKSPACE 中的现有代码构建
# - 使用 online-hk 配置：挂载到容器 /app/configs/_runtime_online_hk.yaml 并通过 RUNTIME_CONFIG_PATH 生效
# - 暴露 gRPC 50058 端口
# - 自动处理容器内服务地址映射
# - 使用本地 replace 方式构建（无需 GitHub Token）

# ----------------------- 可配置项 -----------------------
# 智能检测项目根目录
if [ -n "${WORKSPACE:-}" ]; then
  # 如果设置了WORKSPACE环境变量，使用它
  PROJECT_ROOT="${WORKSPACE/#\~/$HOME}"
else
  # 自动检测：从脚本所在目录向上两级得到项目根目录
  SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  AUTO_DETECTED_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
  
  # 检测真实用户的home目录（即使在sudo下）
  if [ -n "${SUDO_USER:-}" ]; then
    REAL_USER_HOME=$(getent passwd "${SUDO_USER}" | cut -d: -f6)
  else
    REAL_USER_HOME="${HOME}"
  fi
  
  # 优先级：1. 自动检测的目录 2. 真实用户home下的wumitech-com 3. 当前HOME下的wumitech-com
  if [ -d "${AUTO_DETECTED_ROOT}/mdcp_server_operator" ]; then
    PROJECT_ROOT="${AUTO_DETECTED_ROOT}"
  elif [ -d "${REAL_USER_HOME}/wumitech-com/mdcp_server_operator" ]; then
    PROJECT_ROOT="${REAL_USER_HOME}/wumitech-com"
  elif [ -d "${HOME}/wumitech-com/mdcp_server_operator" ]; then
    PROJECT_ROOT="${HOME}/wumitech-com"
  else
    PROJECT_ROOT="${REAL_USER_HOME}/wumitech-com"
  fi
fi

IMAGE_NAME=${IMAGE_NAME:-"mdcp_server_operator:online-hk"}
CONTAINER_NAME=${CONTAINER_NAME:-"mdcp-server-operator-online-hk"}

# 线上配置文件所在的宿主机路径（需提前放好）
if [ -n "${CONFIG_HOST_PATH:-}" ]; then
  HOST_CONFIG="${CONFIG_HOST_PATH/#\~/$HOME}"
else
  HOST_CONFIG="${PROJECT_ROOT}/mdcp_server_operator/configs/online-hk.yaml"
fi
CONTAINER_CONFIG_MOUNT="/app/configs/_runtime_online_hk.yaml"
TEMP_RUNTIME_CONFIG="/tmp/mdcp_server_operator_runtime_online_hk.yaml"

# 端口映射：gRPC 50058
HOST_GRPC_PORT=${HOST_GRPC_PORT:-50058}

# ----------------------- 环境准备 -----------------------
echo "📍 检测到的项目根目录: ${PROJECT_ROOT}"

if [ ! -d "${PROJECT_ROOT}/mdcp_server_operator" ]; then
  echo "❌ 未找到代码目录: ${PROJECT_ROOT}/mdcp_server_operator"
  echo "请先手动拉取最新代码（该目录需包含 mdcp_server_operator、mdcp_common、mdcp_proto）"
  echo "或通过环境变量指定: WORKSPACE=/path/to/wumitech-com bash $0"
  exit 1
fi

echo "[0/3] 生成容器内可用的临时配置: ${TEMP_RUNTIME_CONFIG}"
mkdir -p "$(dirname "${TEMP_RUNTIME_CONFIG}")"
if [ -d "${TEMP_RUNTIME_CONFIG}" ]; then
  echo "⚠️ 检测到 ${TEMP_RUNTIME_CONFIG} 是目录，清理后重建文件"
  rm -rf "${TEMP_RUNTIME_CONFIG}"
fi
cp "${HOST_CONFIG}" "${TEMP_RUNTIME_CONFIG}"

echo "[1/3] 构建镜像: ${IMAGE_NAME}"
# 检测当前架构
if [[ "$(uname -m)" == "arm64" ]] || [[ "$(uname -m)" == "aarch64" ]]; then
  TARGET_ARCH="arm64"
else
  TARGET_ARCH="amd64"
fi
echo "🔍 检测到系统架构: $(uname -m) -> 构建目标: ${TARGET_ARCH}"

docker build --network host \
  --build-arg http_proxy= \
  --build-arg https_proxy= \
  --build-arg HTTP_PROXY= \
  --build-arg HTTPS_PROXY= \
  --build-arg ALL_PROXY= \
  --build-arg TARGETARCH=${TARGET_ARCH} \
  -t "${IMAGE_NAME}" \
  -f "${PROJECT_ROOT}/mdcp_server_operator/Dockerfile.online_hk" \
  "${PROJECT_ROOT}"

echo "[2/3] 停止并删除旧容器（如存在）: ${CONTAINER_NAME}"
if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
  docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
fi

# 创建日志目录
HOST_LOG_DIR="/opt/logs"
mkdir -p "${HOST_LOG_DIR}/server_operator"

echo "[3/3] 以线上配置运行容器（带端口映射功能）"
docker run -d \
  --name "${CONTAINER_NAME}" \
  --network online-hk_mdcp-network \
  --pid=host \
  --privileged \
  -p "${HOST_GRPC_PORT}:50058" \
  -v "${TEMP_RUNTIME_CONFIG}:${CONTAINER_CONFIG_MOUNT}:ro" \
  -v "${HOST_LOG_DIR}:/app/logs" \
  -e "TZ=Asia/Shanghai" \
  -e "RUNTIME_CONFIG_PATH=${CONTAINER_CONFIG_MOUNT}" \
  --restart unless-stopped \
  "${IMAGE_NAME}"

echo "✅ 启动完成。"
echo "- gRPC: localhost:${HOST_GRPC_PORT}"
echo "- 日志目录: ${HOST_LOG_DIR}/server_operator"
echo "- 端口映射: 已启用（通过nsenter执行宿主机nftables）"

