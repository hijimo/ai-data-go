#!/bin/bash

# Genkit 会话管理模块集成测试运行脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查Go环境
check_go() {
    if ! command -v go &> /dev/null; then
        print_error "Go未安装，请先安装Go"
        exit 1
    fi
    print_info "Go版本: $(go version)"
}

# 设置测试环境变量
setup_env() {
    export ENV=test
    export LOG_LEVEL=error
    print_info "测试环境变量已设置"
}

# 运行单元测试
run_unit_tests() {
    print_info "运行单元测试..."
    go test ./internal/genkit/flows/... -v -short || {
        print_error "Flow单元测试失败"
        return 1
    }
    
    go test ./internal/service/... -v -short || {
        print_error "Service单元测试失败"
        return 1
    }
    
    print_info "单元测试通过 ✓"
}

# 运行Flow集成测试
run_flow_integration_tests() {
    print_info "运行Flow集成测试..."
    go test ./internal/genkit/flows/integration_test.go -v || {
        print_error "Flow集成测试失败"
        return 1
    }
    print_info "Flow集成测试通过 ✓"
}

# 运行Service集成测试
run_service_integration_tests() {
    print_info "运行Service集成测试..."
    go test ./internal/service/integration_test.go -v || {
        print_error "Service集成测试失败"
        return 1
    }
    print_info "Service集成测试通过 ✓"
}

# 运行端到端测试
run_e2e_tests() {
    print_info "运行端到端测试..."
    go test ./cmd/server/e2e_test.go -v || {
        print_error "端到端测试失败"
        return 1
    }
    print_info "端到端测试通过 ✓"
}

# 生成测试覆盖率报告
generate_coverage() {
    print_info "生成测试覆盖率报告..."
    
    # 创建覆盖率目录
    mkdir -p coverage
    
    # 运行测试并生成覆盖率
    go test ./internal/... -coverprofile=coverage/coverage.out -covermode=atomic || {
        print_error "生成覆盖率失败"
        return 1
    }
    
    # 生成HTML报告
    go tool cover -html=coverage/coverage.out -o coverage/coverage.html
    
    # 显示覆盖率统计
    print_info "覆盖率统计:"
    go tool cover -func=coverage/coverage.out | tail -n 1
    
    print_info "覆盖率报告已生成: coverage/coverage.html"
}

# 运行竞态检测
run_race_detection() {
    print_info "运行竞态检测..."
    go test ./internal/... -race -short || {
        print_warn "竞态检测发现问题"
        return 1
    }
    print_info "竞态检测通过 ✓"
}

# 运行性能测试
run_benchmark() {
    print_info "运行性能测试..."
    go test ./internal/... -bench=. -benchmem -run=^$ || {
        print_warn "性能测试失败"
        return 1
    }
    print_info "性能测试完成 ✓"
}

# 清理测试数据
cleanup() {
    print_info "清理测试数据..."
    rm -f test.db
    print_info "清理完成 ✓"
}

# 主函数
main() {
    print_info "========================================="
    print_info "Genkit 会话管理模块集成测试"
    print_info "========================================="
    
    # 检查Go环境
    check_go
    
    # 设置环境变量
    setup_env
    
    # 解析命令行参数
    case "${1:-all}" in
        unit)
            run_unit_tests
            ;;
        flow)
            run_flow_integration_tests
            ;;
        service)
            run_service_integration_tests
            ;;
        e2e)
            run_e2e_tests
            ;;
        coverage)
            generate_coverage
            ;;
        race)
            run_race_detection
            ;;
        bench)
            run_benchmark
            ;;
        all)
            run_unit_tests && \
            run_flow_integration_tests && \
            run_service_integration_tests && \
            run_e2e_tests && \
            generate_coverage
            ;;
        *)
            print_error "未知的测试类型: $1"
            echo "用法: $0 [unit|flow|service|e2e|coverage|race|bench|all]"
            exit 1
            ;;
    esac
    
    # 清理
    cleanup
    
    print_info "========================================="
    print_info "所有测试完成！"
    print_info "========================================="
}

# 捕获退出信号
trap cleanup EXIT

# 运行主函数
main "$@"
