#!/bin/bash

# API 调用耗时日志测试脚本
# 用于验证 API 调用耗时记录功能

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
BASE_URL="${BASE_URL:-http://localhost:8080}"
LOG_FILE="logs/app-$(date +%Y-%m-%d).log"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}API 调用耗时日志测试${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 检查日志文件是否存在
if [ ! -f "$LOG_FILE" ]; then
    echo -e "${YELLOW}警告: 日志文件不存在: $LOG_FILE${NC}"
    echo -e "${YELLOW}请确保应用正在运行并生成日志${NC}"
    exit 1
fi

echo -e "${GREEN}✓ 日志文件存在: $LOG_FILE${NC}"
echo ""

# 测试 1: 检查非流式调用的耗时记录
echo -e "${BLUE}测试 1: 检查非流式调用的耗时记录${NC}"
echo "查找包含 'duration' 字段的成功日志..."

SUCCESS_COUNT=$(grep "生成内容成功" "$LOG_FILE" 2>/dev/null | wc -l || echo "0")
if [ "$SUCCESS_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✓ 找到 $SUCCESS_COUNT 条非流式调用成功日志${NC}"
    
    # 显示最近的一条日志
    echo "最近的一条日志:"
    grep "生成内容成功" "$LOG_FILE" | tail -1 | jq '{
        timestamp: .timestamp,
        message: .message,
        tenantId: .fields.tenantId,
        modelName: .fields.modelName,
        duration: .fields.duration,
        totalTokens: .fields.totalTokens
    }' 2>/dev/null || echo "无法解析日志"
    
    # 计算平均响应时间
    echo ""
    echo "统计信息:"
    grep "生成内容成功" "$LOG_FILE" | \
        jq -r '.fields.duration' | \
        sed 's/s$//' | \
        awk '{
            sum += $1;
            count++;
            if (NR == 1 || $1 < min) min = $1;
            if (NR == 1 || $1 > max) max = $1;
        } END {
            if (count > 0) {
                printf "  - 平均响应时间: %.3fs\n", sum/count;
                printf "  - 最快响应时间: %.3fs\n", min;
                printf "  - 最慢响应时间: %.3fs\n", max;
            }
        }'
else
    echo -e "${YELLOW}⚠ 未找到非流式调用成功日志${NC}"
fi
echo ""

# 测试 2: 检查流式调用的耗时记录
echo -e "${BLUE}测试 2: 检查流式调用的耗时记录${NC}"
echo "查找包含 'duration' 和 'ttfb' 字段的流式日志..."

STREAM_COUNT=$(grep "流式生成完成" "$LOG_FILE" 2>/dev/null | wc -l || echo "0")
if [ "$STREAM_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✓ 找到 $STREAM_COUNT 条流式调用完成日志${NC}"
    
    # 显示最近的一条日志
    echo "最近的一条日志:"
    grep "流式生成完成" "$LOG_FILE" | tail -1 | jq '{
        timestamp: .timestamp,
        message: .message,
        tenantId: .fields.tenantId,
        modelName: .fields.modelName,
        duration: .fields.duration,
        ttfb: .fields.ttfb,
        chunkCount: .fields.chunkCount,
        totalTokens: .fields.totalTokens
    }' 2>/dev/null || echo "无法解析日志"
    
    # 计算统计信息
    echo ""
    echo "统计信息:"
    grep "流式生成完成" "$LOG_FILE" | \
        jq -r '[.fields.duration, .fields.ttfb] | @tsv' | \
        awk '{
            gsub(/s$/, "", $1);
            gsub(/s$/, "", $2);
            duration_sum += $1;
            ttfb_sum += $2;
            count++;
            if (NR == 1 || $1 < duration_min) duration_min = $1;
            if (NR == 1 || $1 > duration_max) duration_max = $1;
            if (NR == 1 || $2 < ttfb_min) ttfb_min = $2;
            if (NR == 1 || $2 > ttfb_max) ttfb_max = $2;
        } END {
            if (count > 0) {
                printf "  总耗时:\n";
                printf "    - 平均: %.3fs\n", duration_sum/count;
                printf "    - 最快: %.3fs\n", duration_min;
                printf "    - 最慢: %.3fs\n", duration_max;
                printf "  首字节时间 (TTFB):\n";
                printf "    - 平均: %.3fs\n", ttfb_sum/count;
                printf "    - 最快: %.3fs\n", ttfb_min;
                printf "    - 最慢: %.3fs\n", ttfb_max;
            }
        }'
else
    echo -e "${YELLOW}⚠ 未找到流式调用完成日志${NC}"
fi
echo ""

# 测试 3: 检查失败调用的耗时记录
echo -e "${BLUE}测试 3: 检查失败调用的耗时记录${NC}"
echo "查找包含 'duration' 字段的失败日志..."

FAIL_COUNT=$(grep -E "(生成内容失败|流式生成失败)" "$LOG_FILE" 2>/dev/null | wc -l || echo "0")
if [ "$FAIL_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✓ 找到 $FAIL_COUNT 条失败调用日志${NC}"
    
    # 显示最近的一条日志
    echo "最近的一条日志:"
    grep -E "(生成内容失败|流式生成失败)" "$LOG_FILE" | tail -1 | jq '{
        timestamp: .timestamp,
        message: .message,
        tenantId: .fields.tenantId,
        modelName: .fields.modelName,
        duration: .fields.duration,
        error: .fields.error
    }' 2>/dev/null || echo "无法解析日志"
else
    echo -e "${GREEN}✓ 未找到失败调用日志（这是好事）${NC}"
fi
echo ""

# 测试 4: 查找慢查询
echo -e "${BLUE}测试 4: 查找慢查询 (响应时间 > 2s)${NC}"

SLOW_COUNT=$(grep -E "(生成内容成功|流式生成完成)" "$LOG_FILE" 2>/dev/null | \
    jq -r 'select(.fields.duration | gsub("s$";"") | tonumber > 2) | .fields.duration' | \
    wc -l || echo "0")

if [ "$SLOW_COUNT" -gt 0 ]; then
    echo -e "${YELLOW}⚠ 找到 $SLOW_COUNT 条慢查询${NC}"
    
    echo "慢查询详情:"
    grep -E "(生成内容成功|流式生成完成)" "$LOG_FILE" | \
        jq 'select(.fields.duration | gsub("s$";"") | tonumber > 2) | {
            timestamp: .timestamp,
            modelName: .fields.modelName,
            duration: .fields.duration,
            type: (if .message == "生成内容成功" then "非流式" else "流式" end)
        }' 2>/dev/null | head -5
else
    echo -e "${GREEN}✓ 未找到慢查询${NC}"
fi
echo ""

# 测试 5: 按模型统计平均响应时间
echo -e "${BLUE}测试 5: 按模型统计平均响应时间${NC}"

MODEL_STATS=$(grep "生成内容成功" "$LOG_FILE" 2>/dev/null | \
    jq -r '[.fields.modelName, .fields.duration] | @tsv' | \
    awk '{
        gsub(/s$/, "", $2);
        sum[$1] += $2;
        count[$1]++;
    } END {
        for (model in sum) {
            printf "%s: %.3fs\n", model, sum[model]/count[model];
        }
    }' | sort -t: -k2 -n || echo "")

if [ -n "$MODEL_STATS" ]; then
    echo -e "${GREEN}✓ 模型响应时间统计:${NC}"
    echo "$MODEL_STATS" | while read line; do
        echo "  - $line"
    done
else
    echo -e "${YELLOW}⚠ 无法生成统计信息${NC}"
fi
echo ""

# 测试 6: 检查首字节时间异常
echo -e "${BLUE}测试 6: 检查首字节时间异常 (TTFB > 1s)${NC}"

SLOW_TTFB_COUNT=$(grep "流式生成完成" "$LOG_FILE" 2>/dev/null | \
    jq -r 'select(.fields.ttfb | gsub("s$";"") | tonumber > 1) | .fields.ttfb' | \
    wc -l || echo "0")

if [ "$SLOW_TTFB_COUNT" -gt 0 ]; then
    echo -e "${YELLOW}⚠ 找到 $SLOW_TTFB_COUNT 条首字节时间异常的请求${NC}"
    
    echo "异常请求详情:"
    grep "流式生成完成" "$LOG_FILE" | \
        jq 'select(.fields.ttfb | gsub("s$";"") | tonumber > 1) | {
            timestamp: .timestamp,
            modelName: .fields.modelName,
            ttfb: .fields.ttfb,
            duration: .fields.duration
        }' 2>/dev/null | head -5
else
    echo -e "${GREEN}✓ 未找到首字节时间异常的请求${NC}"
fi
echo ""

# 总结
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}测试总结${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

TOTAL_REQUESTS=$((SUCCESS_COUNT + STREAM_COUNT))
if [ "$TOTAL_REQUESTS" -gt 0 ]; then
    echo -e "${GREEN}✓ API 调用耗时记录功能正常${NC}"
    echo ""
    echo "统计摘要:"
    echo "  - 非流式调用: $SUCCESS_COUNT 次"
    echo "  - 流式调用: $STREAM_COUNT 次"
    echo "  - 失败调用: $FAIL_COUNT 次"
    echo "  - 慢查询: $SLOW_COUNT 次"
    echo "  - 首字节时间异常: $SLOW_TTFB_COUNT 次"
    echo ""
    
    if [ "$SLOW_COUNT" -gt 0 ] || [ "$SLOW_TTFB_COUNT" -gt 0 ]; then
        echo -e "${YELLOW}建议: 存在性能问题，请检查慢查询和首字节时间异常的请求${NC}"
    else
        echo -e "${GREEN}✓ 所有请求性能正常${NC}"
    fi
else
    echo -e "${YELLOW}⚠ 未找到足够的日志数据进行分析${NC}"
    echo "请确保:"
    echo "  1. 应用正在运行"
    echo "  2. 已经有 API 调用"
    echo "  3. 日志级别设置正确"
fi
echo ""

echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}测试完成${NC}"
echo -e "${BLUE}========================================${NC}"
