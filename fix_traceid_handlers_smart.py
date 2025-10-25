#!/usr/bin/env python3
"""
智能修复 handler 中的日志调用，添加 Context 支持
"""

import re
import os
from pathlib import Path

def fix_handler_file(filepath):
    """修复单个 handler 文件"""
    print(f"处理文件: {filepath}")
    
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()
    
    original_content = content
    
    # 1. 修复 writeJSONResponse 方法签名，添加 ctx 参数
    content = re.sub(
        r'func \(h \*(\w+)\) writeJSONResponse\(w http\.ResponseWriter, statusCode int, data interface\{\}\)',
        r'func (h *\1) writeJSONResponse(ctx context.Context, w http.ResponseWriter, statusCode int, data interface{})',
        content
    )
    
    # 2. 在 writeJSONResponse 方法内部，将 h.logger.Error( 替换为 h.logger.ErrorContext(ctx,
    # 使用更精确的正则表达式，只匹配 writeJSONResponse 方法内的日志调用
    def replace_in_write_json(match):
        method_body = match.group(0)
        # 替换方法体内的日志调用
        method_body = re.sub(r'h\.logger\.Error\(', 'h.logger.ErrorContext(ctx, ', method_body)
        return method_body
    
    # 匹配整个 writeJSONResponse 方法
    content = re.sub(
        r'func \(h \*\w+\) writeJSONResponse\(ctx context\.Context, w http\.ResponseWriter, statusCode int, data interface\{\}\) \{[^}]*\}',
        replace_in_write_json,
        content,
        flags=re.DOTALL
    )
    
    # 3. 在主 handler 方法中（有 ctx := r.Context() 的方法），替换日志调用
    # 查找所有包含 ctx := r.Context() 的函数
    def replace_in_handler_method(match):
        method_body = match.group(0)
        # 只在有 ctx := r.Context() 的方法中替换
        if 'ctx := r.Context()' in method_body or 'ctx = r.Context()' in method_body:
            method_body = re.sub(r'h\.logger\.Debug\(', 'h.logger.DebugContext(ctx, ', method_body)
            method_body = re.sub(r'h\.logger\.Info\(', 'h.logger.InfoContext(ctx, ', method_body)
            method_body = re.sub(r'h\.logger\.Warn\(', 'h.logger.WarnContext(ctx, ', method_body)
            method_body = re.sub(r'h\.logger\.Error\(', 'h.logger.ErrorContext(ctx, ', method_body)
        return method_body
    
    # 匹配所有 handler 方法（以 func (h *XXXHandler) Handle 开头）
    content = re.sub(
        r'func \(h \*\w+\) Handle\w+\([^)]*\) \{.*?(?=\nfunc |\Z)',
        replace_in_handler_method,
        content,
        flags=re.DOTALL
    )
    
    # 4. 更新所有调用 writeJSONResponse 的地方，添加 ctx 参数
    content = re.sub(
        r'h\.writeJSONResponse\(w,',
        'h.writeJSONResponse(ctx, w,',
        content
    )
    
    # 5. 处理 provider_handler.go 中的特殊情况（没有 ctx := r.Context()）
    if 'provider_handler.go' in filepath:
        # 在每个 Handle 方法开始处添加 ctx := r.Context()
        def add_ctx_to_provider_handler(match):
            method_start = match.group(0)
            # 如果已经有 ctx := r.Context()，跳过
            if 'ctx := r.Context()' in method_start or 'ctx = r.Context()' in method_start:
                return method_start
            # 在方法体开始处添加 ctx := r.Context()
            return method_start.replace(') {', ') {\n\tctx := r.Context()\n', 1)
        
        content = re.sub(
            r'func \(h \*ProviderHandler\) Handle\w+\(w http\.ResponseWriter, r \*http\.Request\) \{',
            add_ctx_to_provider_handler,
            content
        )
        
        # 然后替换日志调用
        content = re.sub(r'h\.logger\.Debug\(', 'h.logger.DebugContext(ctx, ', content)
        content = re.sub(r'h\.logger\.Info\(', 'h.logger.InfoContext(ctx, ', content)
        content = re.sub(r'h\.logger\.Warn\(', 'h.logger.WarnContext(ctx, ', content)
        content = re.sub(r'h\.logger\.Error\(', 'h.logger.ErrorContext(ctx, ', content)
    
    # 只有在内容发生变化时才写入
    if content != original_content:
        # 备份原文件
        backup_path = filepath + '.bak'
        with open(backup_path, 'w', encoding='utf-8') as f:
            f.write(original_content)
        print(f"  已备份到: {backup_path}")
        
        # 写入修改后的内容
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(content)
        print(f"  已修复")
        return True
    else:
        print(f"  无需修改")
        return False

def main():
    """主函数"""
    print("开始智能修复 handler 日志调用...\n")
    
    # 定义需要处理的文件
    handler_dir = Path('internal/api/handler')
    handler_files = [
        'abort.go',
        'tenant_handler.go',
        'audit_handler.go',
        'health.go',
        'user_handler.go',
        'provider_handler.go',
        'chat.go',
        'chat_stream.go',
        'session_handler.go',
        'message_handler.go',
        'auth_handler.go',
    ]
    
    fixed_count = 0
    for filename in handler_files:
        filepath = handler_dir / filename
        if filepath.exists():
            if fix_handler_file(str(filepath)):
                fixed_count += 1
        else:
            print(f"跳过（文件不存在）: {filepath}")
    
    print(f"\n修复完成！共修复 {fixed_count} 个文件")
    print("\n如果需要恢复，可以运行：")
    print("  for file in internal/api/handler/*.go.bak; do mv \"$file\" \"${file%.bak}\"; done")

if __name__ == '__main__':
    main()
