#!/bin/bash
# 清理 aigw:config:model_versions 中的脏数据
# 只保留有实际 endpoints 配置的模型

set -e

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATEWAY_CONFIG="$SCRIPT_DIR/../../ai-gateway/config/local.yml"

# 检查配置文件是否存在
if [ ! -f "$GATEWAY_CONFIG" ]; then
    echo "错误: 找不到配置文件 $GATEWAY_CONFIG"
    exit 1
fi

# 读取 Redis 配置
REDIS_ADDR=$(grep -A 5 "redis:" "$GATEWAY_CONFIG" | grep "addr:" | awk '{print $2}' | tr -d '"')
REDIS_PASSWORD=$(grep -A 5 "redis:" "$GATEWAY_CONFIG" | grep "password:" | awk '{print $2}' | tr -d '"')
REDIS_HOST=$(echo $REDIS_ADDR | cut -d: -f1)
REDIS_PORT=$(echo $REDIS_ADDR | cut -d: -f2)

echo "=== 清理 aigw:config:model_versions 中的脏数据 ==="
echo "Redis: $REDIS_ADDR"
echo

# 获取所有 model_versions 中的模型
MODEL_CODES=$(redis-cli -h $REDIS_HOST -p $REDIS_PORT -a "$REDIS_PASSWORD" --no-auth-warning HKEYS "aigw:config:model_versions")

CLEANED=0
KEPT=0

for model in $MODEL_CODES; do
    # 检查是否有对应的 endpoints 配置
    EXISTS=$(redis-cli -h $REDIS_HOST -p $REDIS_PORT -a "$REDIS_PASSWORD" --no-auth-warning EXISTS "aigw:config:endpoints:$model")

    if [ "$EXISTS" = "0" ]; then
        echo "删除脏数据: $model (无 endpoints 配置)"
        redis-cli -h $REDIS_HOST -p $REDIS_PORT -a "$REDIS_PASSWORD" --no-auth-warning HDEL "aigw:config:model_versions" "$model" > /dev/null
        CLEANED=$((CLEANED + 1))
    else
        KEPT=$((KEPT + 1))
    fi
done

echo
echo "=== 清理完成 ==="
echo "保留: $KEPT 个模型"
echo "删除: $CLEANED 个脏数据"
