#!/bin/bash

# 日志持久化功能测试脚本

echo "=========================================="
echo "日志持久化功能测试"
echo "=========================================="
echo ""

# 1. 检查日志目录
echo "1. 检查日志目录..."
if [ -d "logs" ]; then
    echo "✓ 日志目录已存在"
    ls -lh logs/
else
    echo "✗ 日志目录不存在，将在服务启动时自动创建"
fi
echo ""

# 2. 检查配置
echo "2. 检查日志配置..."
if [ -f ".env" ]; then
    echo "当前日志配置："
    grep "^LOG_" .env || echo "未找到日志配置，将使用默认值"
else
    echo "✗ .env 文件不存在，请先创建配置文件"
    echo "提示：可以复制 .env.example 为 .env"
    exit 1
fi
echo ""

# 3. 显示推荐配置
echo "3. 推荐的日志配置："
echo "----------------------------------------"
echo "# 开发环境（同时输出到控制台和文件）"
echo "LOG_ENABLE_FILE=true"
echo "LOG_ENABLE_CONSOLE=true"
echo "LOG_LEVEL=debug"
echo "LOG_FORMAT=text"
echo "LOG_DIR=logs"
echo ""
echo "# 生产环境（仅输出到文件）"
echo "LOG_ENABLE_FILE=true"
echo "LOG_ENABLE_CONSOLE=false"
echo "LOG_LEVEL=info"
echo "LOG_FORMAT=json"
echo "LOG_DIR=logs"
echo "----------------------------------------"
echo ""

# 4. 查看今天的日志文件
TODAY=$(date +%Y-%m-%d)
LOG_FILE="logs/app-${TODAY}.log"

echo "4. 查看今天的日志文件..."
if [ -f "$LOG_FILE" ]; then
    echo "✓ 找到今天的日志文件: $LOG_FILE"
    echo "文件大小: $(du -h "$LOG_FILE" | cut -f1)"
    echo "行数: $(wc -l < "$LOG_FILE")"
    echo ""
    echo "最近 5 条日志："
    echo "----------------------------------------"
    tail -n 5 "$LOG_FILE"
    echo "----------------------------------------"
else
    echo "✗ 今天的日志文件不存在: $LOG_FILE"
    echo "提示：启动服务后会自动创建"
fi
echo ""

# 5. 统计日志文件
echo "5. 日志文件统计..."
if [ -d "logs" ]; then
    FILE_COUNT=$(find logs/ -name "app-*.log" -type f | wc -l)
    echo "日志文件总数: $FILE_COUNT"
    
    if [ $FILE_COUNT -gt 0 ]; then
        echo "日志文件列表："
        find logs/ -name "app-*.log" -type f -exec ls -lh {} \; | awk '{print $9, $5}'
        
        TOTAL_SIZE=$(du -sh logs/ | cut -f1)
        echo "日志目录总大小: $TOTAL_SIZE"
    fi
else
    echo "日志目录不存在"
fi
echo ""

# 6. TraceID 查询示例
echo "6. TraceID 查询示例..."
if [ -f "$LOG_FILE" ]; then
    # 尝试提取一个 traceId
    TRACE_ID=$(grep -o '"traceId":"[^"]*"' "$LOG_FILE" 2>/dev/null | head -n 1 | cut -d'"' -f4)
    
    if [ -n "$TRACE_ID" ]; then
        echo "找到 TraceID: $TRACE_ID"
        echo "查询该 TraceID 的所有日志："
        echo "----------------------------------------"
        grep "$TRACE_ID" "$LOG_FILE" | head -n 3
        echo "----------------------------------------"
        echo ""
        echo "查询命令："
        echo "  grep \"$TRACE_ID\" $LOG_FILE"
        echo "  grep \"$TRACE_ID\" logs/app-*.log  # 查询所有日志文件"
    else
        echo "未找到 TraceID，可能是："
        echo "  - 服务尚未启动"
        echo "  - 日志格式为 text（TraceID 格式不同）"
        echo "  - 还没有处理过 HTTP 请求"
    fi
else
    echo "日志文件不存在，无法演示查询"
fi
echo ""

# 7. 日志清理建议
echo "7. 日志清理建议..."
echo "----------------------------------------"
echo "# 删除 30 天前的日志"
echo "find logs/ -name \"app-*.log\" -mtime +30 -delete"
echo ""
echo "# 压缩 7 天前的日志"
echo "find logs/ -name \"app-*.log\" -mtime +7 -exec gzip {} \;"
echo ""
echo "# 查看日志目录大小"
echo "du -sh logs/"
echo "----------------------------------------"
echo ""

echo "=========================================="
echo "测试完成！"
echo "=========================================="
echo ""
echo "下一步："
echo "1. 启动服务: make run 或 ./bin/server"
echo "2. 发送测试请求"
echo "3. 查看日志文件: tail -f $LOG_FILE"
echo "4. 通过 TraceID 追踪请求链路"
