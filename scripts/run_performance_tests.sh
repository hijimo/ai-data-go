#!/bin/bash

# 性能测试运行脚本
# 用于运行所有性能基准测试并生成报告

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 创建结果目录
RESULTS_DIR="test-results/performance"
mkdir -p "$RESULTS_DIR"

TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
REPORT_FILE="$RESULTS_DIR/benchmark_report_$TIMESTAMP.txt"

print_info "性能测试开始..."
print_info "结果将保存到: $REPORT_FILE"

# 运行性能测试的函数
run_benchmark() {
    local package=$1
    local name=$2
    
    print_info "运行 $name 性能测试..."
    
    {
        echo "========================================="
        echo "$name 性能测试结果"
        echo "时间: $(date)"
        echo "========================================="
        echo ""
    } >> "$REPORT_FILE"
    
    # 运行基准测试，输出到文件
    if go test -bench=. -benchmem -benchtime=3s "$package" >> "$REPORT_FILE" 2>&1; then
        print_success "$name 测试完成"
    else
        print_error "$name 测试失败"
        return 1
    fi
    
    echo "" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
}

# 1. 服务层性能测试
run_benchmark "./internal/service" "服务层"

# 2. 仓储层性能测试
run_benchmark "./internal/repository" "仓储层（向量检索）"

# 3. 缓存层性能测试
run_benchmark "./internal/storage" "缓存层"

# 生成性能摘要
print_info "生成性能摘要..."

{
    echo "========================================="
    echo "性能测试摘要"
    echo "========================================="
    echo ""
    echo "测试时间: $(date)"
    echo "Go版本: $(go version)"
    echo ""
    echo "关键性能指标:"
    echo ""
    
    # 提取关键指标
    echo "1. 上下文构建性能:"
    grep "BenchmarkContextService_BuildContext-" "$REPORT_FILE" | head -5
    echo ""
    
    echo "2. 向量检索性能:"
    grep "BenchmarkMemoryRepository_SearchByVector-" "$REPORT_FILE" | head -5
    echo ""
    
    echo "3. 缓存操作性能:"
    grep "BenchmarkCacheService_" "$REPORT_FILE" | head -5
    echo ""
    
    echo "4. 并发性能:"
    grep "Parallel" "$REPORT_FILE" | head -5
    echo ""
    
} > "$RESULTS_DIR/summary_$TIMESTAMP.txt"

print_success "性能测试完成！"
print_info "详细报告: $REPORT_FILE"
print_info "性能摘要: $RESULTS_DIR/summary_$TIMESTAMP.txt"

# 检查性能指标是否达标
print_info "检查性能指标..."

# 定义性能阈值（纳秒）
CONTEXT_BUILD_THRESHOLD=10000000  # 10ms
VECTOR_SEARCH_THRESHOLD=50000000  # 50ms
CACHE_GET_THRESHOLD=1000000       # 1ms

# 提取实际性能数据并检查
check_performance() {
    local benchmark_name=$1
    local threshold=$2
    local metric_name=$3
    
    # 从报告中提取性能数据（ns/op）
    local actual=$(grep "$benchmark_name" "$REPORT_FILE" | awk '{print $3}' | head -1)
    
    if [ -z "$actual" ]; then
        print_warning "未找到 $metric_name 的性能数据"
        return
    fi
    
    # 移除单位，只保留数字
    actual_value=$(echo "$actual" | sed 's/[^0-9.]//g')
    
    if (( $(echo "$actual_value < $threshold" | bc -l) )); then
        print_success "$metric_name: ${actual_value}ns (阈值: ${threshold}ns) ✓"
    else
        print_warning "$metric_name: ${actual_value}ns (阈值: ${threshold}ns) ✗"
    fi
}

check_performance "BenchmarkContextService_BuildContext-" "$CONTEXT_BUILD_THRESHOLD" "上下文构建"
check_performance "BenchmarkMemoryRepository_SearchByVector-" "$VECTOR_SEARCH_THRESHOLD" "向量检索"
check_performance "BenchmarkCacheService_Get_Small-" "$CACHE_GET_THRESHOLD" "缓存读取"

echo ""
print_info "性能测试报告已生成"
print_info "查看完整报告: cat $REPORT_FILE"
print_info "查看性能摘要: cat $RESULTS_DIR/summary_$TIMESTAMP.txt"
