#!/bin/bash

# 修复所有 handler 中的日志调用，添加 Context 支持

echo "开始修复 handler 日志调用..."

# 定义需要处理的文件列表
files=(
    "internal/api/handler/abort.go"
    "internal/api/handler/tenant_handler.go"
    "internal/api/handler/audit_handler.go"
    "internal/api/handler/health.go"
    "internal/api/handler/user_handler.go"
    "internal/api/handler/provider_handler.go"
    "internal/api/handler/chat.go"
    "internal/api/handler/chat_stream.go"
    "internal/api/handler/session_handler.go"
    "internal/api/handler/message_handler.go"
    "internal/api/handler/auth_handler.go"
)

# 备份文件
echo "备份文件..."
for file in "${files[@]}"; do
    if [ -f "$file" ]; then
        cp "$file" "${file}.bak"
        echo "  已备份: $file"
    fi
done

# 替换日志调用
echo ""
echo "替换日志调用..."
for file in "${files[@]}"; do
    if [ -f "$file" ]; then
        # 替换 h.logger.Debug( 为 h.logger.DebugContext(ctx,
        sed -i '' 's/h\.logger\.Debug(/h.logger.DebugContext(ctx, /g' "$file"
        
        # 替换 h.logger.Info( 为 h.logger.InfoContext(ctx,
        sed -i '' 's/h\.logger\.Info(/h.logger.InfoContext(ctx, /g' "$file"
        
        # 替换 h.logger.Warn( 为 h.logger.WarnContext(ctx,
        sed -i '' 's/h\.logger\.Warn(/h.logger.WarnContext(ctx, /g' "$file"
        
        # 替换 h.logger.Error( 为 h.logger.ErrorContext(ctx,
        sed -i '' 's/h\.logger\.Error(/h.logger.ErrorContext(ctx, /g' "$file"
        
        echo "  已处理: $file"
    else
        echo "  跳过（文件不存在）: $file"
    fi
done

echo ""
echo "修复完成！"
echo ""
echo "如果需要恢复，可以运行："
echo "  for file in internal/api/handler/*.go.bak; do mv \"\$file\" \"\${file%.bak}\"; done"
