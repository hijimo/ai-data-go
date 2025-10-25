#!/usr/bin/env python3
"""
修复所有 handler 文件中的 traceId 问题
"""

import re
import os

# 需要修复的文件列表
files_to_fix = [
    "internal/api/handler/tenant_handler.go",
    "internal/api/handler/user_handler.go",
    "internal/api/handler/session_handler.go",
    "internal/api/handler/message_handler.go",
    "internal/api/handler/audit_handler.go",
]

def fix_write_error_response(content):
    """修复 writeErrorResponse 方法"""
    # 1. 修改方法签名
    pattern1 = r'func \(h \*(\w+)Handler\) writeErrorResponse\(w http\.ResponseWriter, appErr \*errors\.AppError\)'
    replacement1 = r'func (h *\1Handler) writeErrorResponse(w http.ResponseWriter, r *http.Request, appErr *errors.AppError)'
    content = re.sub(pattern1, replacement1, content)
    
    # 2. 修改方法体中的 response.Error 调用
    pattern2 = r'(\s+)resp := response\.Error\[any\]\(appErr\.Code, appErr\.Message\)'
    replacement2 = r'\1ctx := r.Context()\n\1resp := response.ErrorWithContext[any](ctx, appErr.Code, appErr.Message)'
    content = re.sub(pattern2, replacement2, content)
    
    # 3. 修改所有调用 writeErrorResponse 的地方
    pattern3 = r'h\.writeErrorResponse\(w, '
    replacement3 = r'h.writeErrorResponse(w, r, '
    content = re.sub(pattern3, replacement3, content)
    
    return content

def fix_write_validation_error_response(content):
    """修复 writeValidationErrorResponse 方法"""
    # 1. 修改方法签名
    pattern1 = r'func \(h \*(\w+)Handler\) writeValidationErrorResponse\(w http\.ResponseWriter, validationErrors'
    replacement1 = r'func (h *\1Handler) writeValidationErrorResponse(w http.ResponseWriter, r *http.Request, validationErrors'
    content = re.sub(pattern1, replacement1, content)
    
    # 2. 修改方法体中的 response.ErrorWithData 调用
    # 需要在 resp := response.ErrorWithData 之前添加 ctx := r.Context()
    pattern2 = r'(\s+)resp := response\.ErrorWithData\('
    replacement2 = r'\1ctx := r.Context()\n\1resp := response.ErrorWithDataContext(\n\1\tctx,'
    content = re.sub(pattern2, replacement2, content)
    
    # 3. 修改所有调用 writeValidationErrorResponse 的地方
    pattern3 = r'h\.writeValidationErrorResponse\(w, '
    replacement3 = r'h.writeValidationErrorResponse(w, r, '
    content = re.sub(pattern3, replacement3, content)
    
    return content

def fix_file(filepath):
    """修复单个文件"""
    print(f"处理文件: {filepath}")
    
    # 读取文件
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # 备份原文件
    backup_path = filepath + '.bak'
    with open(backup_path, 'w', encoding='utf-8') as f:
        f.write(content)
    print(f"  已备份到: {backup_path}")
    
    # 应用修复
    content = fix_write_error_response(content)
    content = fix_write_validation_error_response(content)
    
    # 写回文件
    with open(filepath, 'w', encoding='utf-8') as f:
        f.write(content)
    
    print(f"  ✓ 完成")

def main():
    print("开始修复 handler 文件中的 traceId 问题...\n")
    
    for filepath in files_to_fix:
        if os.path.exists(filepath):
            fix_file(filepath)
        else:
            print(f"✗ 文件不存在: {filepath}")
        print()
    
    print("修复完成！")
    print("\n备份文件已保存为 *.bak，如果修复有问题可以恢复")

if __name__ == '__main__':
    main()
