#!/bin/bash

# 修复所有 handler 文件中的 traceId 问题
# 将 writeErrorResponse 方法添加 *http.Request 参数
# 将所有调用改为使用 ErrorWithContext

echo "开始修复 handler 文件中的 traceId 问题..."

# 需要修复的文件列表（auth_handler.go 已经手动修复）
files=(
    "internal/api/handler/tenant_handler.go"
    "internal/api/handler/user_handler.go"
    "internal/api/handler/session_handler.go"
    "internal/api/handler/message_handler.go"
    "internal/api/handler/audit_handler.go"
)

for file in "${files[@]}"; do
    if [ -f "$file" ]; then
        echo "处理文件: $file"
        
        # 备份原文件
        cp "$file" "${file}.bak"
        
        # 1. 修改 writeErrorResponse 方法签名，添加 r *http.Request 参数
        sed -i '' 's/func (h \*\([A-Za-z]*\)Handler) writeErrorResponse(w http.ResponseWriter, appErr \*errors.AppError)/func (h *\1Handler) writeErrorResponse(w http.ResponseWriter, r *http.Request, appErr *errors.AppError)/g' "$file"
        
        # 2. 修改 writeErrorResponse 方法体，使用 ErrorWithContext
        sed -i '' 's/resp := response\.Error\[any\](appErr\.Code, appErr\.Message)/ctx := r.Context()\n\tresp := response.ErrorWithContext[any](ctx, appErr.Code, appErr.Message)/g' "$file"
        
        # 3. 修改所有调用 writeErrorResponse 的地方，添加 r 参数
        sed -i '' 's/h\.writeErrorResponse(w, /h.writeErrorResponse(w, r, /g' "$file"
        
        # 4. 修改 writeValidationErrorResponse 方法签名（如果存在）
        sed -i '' 's/func (h \*\([A-Za-z]*\)Handler) writeValidationErrorResponse(w http.ResponseWriter, validationErrors/func (h *\1Handler) writeValidationErrorResponse(w http.ResponseWriter, r *http.Request, validationErrors/g' "$file"
        
        # 5. 修改 writeValidationErrorResponse 方法体，使用 ErrorWithDataContext
        sed -i '' 's/resp := response\.ErrorWithData(/ctx := r.Context()\n\tresp := response.ErrorWithDataContext(\n\t\tctx,/g' "$file"
        
        # 6. 修改所有调用 writeValidationErrorResponse 的地方
        sed -i '' 's/h\.writeValidationErrorResponse(w, /h.writeValidationErrorResponse(w, r, /g' "$file"
        
        echo "✓ 完成: $file"
    else
        echo "✗ 文件不存在: $file"
    fi
done

echo ""
echo "修复完成！请检查以下文件："
for file in "${files[@]}"; do
    echo "  - $file"
done
echo ""
echo "备份文件已保存为 *.bak，如果修复有问题可以恢复"
