#!/usr/bin/env python3
"""
为 handler 文件添加 context 包导入
"""

import re
from pathlib import Path

def add_context_import(filepath):
    """为文件添加 context 导入"""
    print(f"处理文件: {filepath}")
    
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # 检查是否已经导入了 context
    if re.search(r'^\s*"context"\s*$', content, re.MULTILINE):
        print(f"  已有 context 导入，跳过")
        return False
    
    # 检查是否使用了 context.Context
    if 'context.Context' not in content:
        print(f"  未使用 context.Context，跳过")
        return False
    
    # 在 import 块中添加 context
    # 查找 import 块
    import_pattern = r'(import \(\n)((?:\s+[^\n]+\n)*)'
    
    def add_import(match):
        import_start = match.group(1)
        import_body = match.group(2)
        
        # 在第一行添加 context 导入
        return import_start + '\t"context"\n' + import_body
    
    new_content = re.sub(import_pattern, add_import, content, count=1)
    
    if new_content != content:
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(new_content)
        print(f"  已添加 context 导入")
        return True
    else:
        print(f"  无法添加导入（未找到 import 块）")
        return False

def main():
    """主函数"""
    print("开始添加 context 导入...\n")
    
    # 定义需要处理的文件
    handler_dir = Path('internal/api/handler')
    handler_files = list(handler_dir.glob('*.go'))
    
    fixed_count = 0
    for filepath in handler_files:
        if add_context_import(filepath):
            fixed_count += 1
    
    print(f"\n完成！共修复 {fixed_count} 个文件")

if __name__ == '__main__':
    main()
