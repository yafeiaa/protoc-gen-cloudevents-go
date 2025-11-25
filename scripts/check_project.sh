#!/bin/bash

# 项目完整性检查脚本
# 用于验证项目是否准备好发布

set -e

echo "🔍 开始检查 protoc-gen-cloudevents-go 项目..."
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

check_pass() {
    echo -e "${GREEN}✓${NC} $1"
}

check_fail() {
    echo -e "${RED}✗${NC} $1"
    FAILED=1
}

check_warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

FAILED=0

# 1. 检查 Go 环境
echo "📦 检查 Go 环境..."
if command -v go &> /dev/null; then
    GO_VERSION=$(go version | awk '{print $3}')
    check_pass "Go 已安装: $GO_VERSION"
else
    check_fail "Go 未安装"
fi
echo ""

# 2. 检查项目结构
echo "📁 检查项目结构..."
required_files=(
    "go.mod"
    "Makefile"
    "README.md"
    "LICENSE"
    ".gitignore"
    "cmd/protoc-gen-cloudevents/main.go"
    "transport/memory/memory.go"
    "transport/memory/memory_test.go"
    "examples/basic/integration_test.go"
    ".github/workflows/ci.yml"
    ".golangci.yml"
    "TESTING.md"
)

for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        check_pass "$file 存在"
    else
        check_fail "$file 缺失"
    fi
done
echo ""

# 3. 检查依赖
echo "📥 检查依赖..."
if go mod download &> /dev/null && go mod verify &> /dev/null; then
    check_pass "依赖完整"
else
    check_fail "依赖检查失败"
fi
echo ""

# 4. 运行测试
echo "🧪 运行测试..."
if go test -short ./... &> /dev/null; then
    check_pass "测试通过"
    
    # 统计测试数量
    TEST_COUNT=$(go test -v -short ./... 2>&1 | grep -c "^=== RUN" || true)
    check_pass "测试用例数: $TEST_COUNT"
else
    check_fail "测试失败"
fi
echo ""

# 5. 检查测试覆盖率
echo "📊 检查测试覆盖率..."
if go test -coverprofile=coverage.tmp ./... &> /dev/null; then
    COVERAGE=$(go tool cover -func=coverage.tmp | grep total | awk '{print $3}')
    rm coverage.tmp
    
    COVERAGE_NUM=$(echo $COVERAGE | sed 's/%//')
    if (( $(echo "$COVERAGE_NUM >= 70" | bc -l) )); then
        check_pass "测试覆盖率: $COVERAGE (目标: >= 70%)"
    else
        check_warn "测试覆盖率: $COVERAGE (建议: >= 70%)"
    fi
else
    check_warn "无法计算覆盖率"
fi
echo ""

# 6. 运行竞态检测
echo "🏃 运行竞态检测..."
if go test -race -short ./... &> /dev/null; then
    check_pass "竞态检测通过"
else
    check_fail "竞态检测发现问题"
fi
echo ""

# 7. 构建项目
echo "🔨 构建项目..."
if go build -o /tmp/protoc-gen-cloudevents ./cmd/protoc-gen-cloudevents &> /dev/null; then
    check_pass "构建成功"
    rm /tmp/protoc-gen-cloudevents
else
    check_fail "构建失败"
fi
echo ""

# 8. 代码格式检查
echo "🎨 检查代码格式..."
UNFORMATTED=$(gofmt -l . 2>/dev/null || true)
if [ -z "$UNFORMATTED" ]; then
    check_pass "代码格式正确"
else
    check_warn "以下文件需要格式化:\n$UNFORMATTED"
fi
echo ""

# 9. 检查 Git 状态
echo "🌿 检查 Git 状态..."
if [ -d ".git" ]; then
    check_pass "Git 仓库已初始化"
    
    if git remote -v | grep -q "origin"; then
        REMOTE=$(git remote get-url origin)
        check_pass "远程仓库: $REMOTE"
    else
        check_warn "未配置远程仓库"
    fi
else
    check_warn "Git 仓库未初始化"
fi
echo ""

# 10. 检查文档
echo "📚 检查文档..."
required_docs=(
    "README.md"
    "README_CN.md"
    "CONTRIBUTING.md"
)

for doc in "${required_docs[@]}"; do
    if [ -f "$doc" ] && [ -s "$doc" ]; then
        check_pass "$doc ($(wc -l < $doc) 行)"
    else
        check_fail "$doc 缺失或为空"
    fi
done
echo ""

# 11. 检查 CI 配置
echo "🤖 检查 CI 配置..."
if [ -f ".github/workflows/ci.yml" ]; then
    check_pass "GitHub Actions CI 配置存在"
    
    if grep -q "test" .github/workflows/ci.yml; then
        check_pass "包含测试步骤"
    fi
    
    if grep -q "codecov" .github/workflows/ci.yml; then
        check_pass "包含覆盖率上传"
    fi
else
    check_fail "CI 配置缺失"
fi
echo ""

# 12. 检查许可证
echo "📜 检查许可证..."
if [ -f "LICENSE" ]; then
    if grep -q "MIT" LICENSE; then
        check_pass "MIT 许可证"
    else
        check_warn "许可证类型未知"
    fi
else
    check_fail "LICENSE 文件缺失"
fi
echo ""

# 总结
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 项目检查完成！所有检查通过！${NC}"
    echo ""
    echo "✅ 项目已准备好发布到 GitHub"
    echo ""
    echo "下一步："
    echo "  1. git init"
    echo "  2. git add ."
    echo "  3. git commit -m 'Initial commit'"
    echo "  4. gh repo create protoc-gen-cloudevents-go --public --source=. --push"
    echo ""
    echo "或者查看 GET_STARTED.md 获取详细指南"
else
    echo -e "${RED}❌ 项目检查发现问题，请修复后再发布${NC}"
    exit 1
fi

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
