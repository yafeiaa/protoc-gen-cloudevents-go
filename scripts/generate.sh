#!/bin/bash

# 代码生成脚本
# 用法: ./scripts/generate.sh [proto_file]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

cd "${PROJECT_ROOT}"

echo "🚀 开始生成 CloudEvents 代码..."

# 默认生成所有示例
PROTO_FILES=(
    "examples/basic/events.proto"
)

if [ $# -gt 0 ]; then
    PROTO_FILES=("$@")
fi

for PROTO_FILE in "${PROTO_FILES[@]}"; do
    echo ""
    echo "📦 处理: ${PROTO_FILE}"
    
    OUTPUT_DIR="$(dirname "${PROTO_FILE}")"
    
    # 生成 protobuf 基础代码
    echo "  └─ 生成 protobuf 消息定义..."
    protoc \
        -I . \
        -I ./proto \
        --go_out="${OUTPUT_DIR}" \
        --go_opt=paths=source_relative \
        "${PROTO_FILE}"
    
    # 生成 CloudEvents 代码
    echo "  └─ 生成 CloudEvents 发布/订阅函数..."
    protoc \
        -I . \
        -I ./proto \
        --cloudevents_out="${OUTPUT_DIR}" \
        --cloudevents_opt=paths=source_relative \
        "${PROTO_FILE}"
    
    echo "  ✅ 完成"
done

echo ""
echo "✅ 所有代码生成完成!"
