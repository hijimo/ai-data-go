package genkit

import (
	"context"
	"errors"
	"testing"
	"time"

	"genkit-ai-service/internal/monitoring"
)

// TestFlowExecutionInfo 测试Flow执行信息
func TestFlowExecutionInfo(t *testing.T) {
	// 创建执行信息
	info := NewFlowExecutionInfo("testFlow")

	// 验证初始状态
	if info.FlowName != "testFlow" {
		t.Errorf("期望FlowName为testFlow，实际为%s", info.FlowName)
	}

	if info.Status != "running" {
		t.Errorf("期望初始状态为running，实际为%s", info.Status)
	}

	// 添加元数据
	info.AddMetadata("key1", "value1")
	info.AddMetadata("key2", 123)

	if len(info.Metadata) != 2 {
		t.Errorf("期望元数据数量为2，实际为%d", len(info.Metadata))
	}

	// 标记成功完成
	info.Complete(nil)
	if info.Status != "success" {
		t.Errorf("期望状态为success，实际为%s", info.Status)
	}

	// 验证执行时长
	duration := info.Duration()
	if duration <= 0 {
		t.Error("期望执行时长大于0")
	}
}

// TestFlowExecutionInfoWithError 测试带错误的Flow执行信息
func TestFlowExecutionInfoWithError(t *testing.T) {
	info := NewFlowExecutionInfo("testFlow")

	// 标记失败完成
	testErr := errors.New("测试错误")
	info.Complete(testErr)

	if info.Status != "error" {
		t.Errorf("期望状态为error，实际为%s", info.Status)
	}

	if info.Error == nil {
		t.Error("期望Error不为nil")
	}

	if info.Error.Error() != "测试错误" {
		t.Errorf("期望错误消息为'测试错误'，实际为'%s'", info.Error.Error())
	}
}

// TestMonitorFlowWithInput 测试带输入输出的Flow监控
func TestMonitorFlowWithInput(t *testing.T) {
	// 重置监控指标
	monitoring.GetMetrics().Reset()

	// 定义测试Flow
	testFlow := func(ctx context.Context, input string) (string, error) {
		time.Sleep(10 * time.Millisecond) // 模拟处理时间
		return "output: " + input, nil
	}

	// 包装Flow
	wrappedFlow := MonitorFlowWithInput("testFlow", testFlow)

	// 执行Flow
	ctx := context.Background()
	output, err := wrappedFlow(ctx, "test input")

	// 验证结果
	if err != nil {
		t.Errorf("期望没有错误，实际为%v", err)
	}

	if output != "output: test input" {
		t.Errorf("期望输出为'output: test input'，实际为'%s'", output)
	}

	// 验证监控指标
	metrics := monitoring.GetMetrics().GetFlowMetrics("testFlow")
	if metrics.Executions != 1 {
		t.Errorf("期望执行次数为1，实际为%d", metrics.Executions)
	}

	if metrics.Successes != 1 {
		t.Errorf("期望成功次数为1，实际为%d", metrics.Successes)
	}

	if metrics.Errors != 0 {
		t.Errorf("期望错误次数为0，实际为%d", metrics.Errors)
	}

	if metrics.AvgDuration <= 0 {
		t.Error("期望平均执行时间大于0")
	}
}

// TestMonitorFlowWithInputError 测试Flow执行失败的监控
func TestMonitorFlowWithInputError(t *testing.T) {
	// 重置监控指标
	monitoring.GetMetrics().Reset()

	// 定义会失败的测试Flow
	testFlow := func(ctx context.Context, input string) (string, error) {
		return "", errors.New("测试错误")
	}

	// 包装Flow
	wrappedFlow := MonitorFlowWithInput("testFlowError", testFlow)

	// 执行Flow
	ctx := context.Background()
	_, err := wrappedFlow(ctx, "test input")

	// 验证错误
	if err == nil {
		t.Error("期望有错误，实际为nil")
	}

	// 验证监控指标
	metrics := monitoring.GetMetrics().GetFlowMetrics("testFlowError")
	if metrics.Executions != 1 {
		t.Errorf("期望执行次数为1，实际为%d", metrics.Executions)
	}

	if metrics.Successes != 0 {
		t.Errorf("期望成功次数为0，实际为%d", metrics.Successes)
	}

	if metrics.Errors != 1 {
		t.Errorf("期望错误次数为1，实际为%d", metrics.Errors)
	}
}

// TestMonitorFlowNoInput 测试无输入参数的Flow监控
func TestMonitorFlowNoInput(t *testing.T) {
	// 重置监控指标
	monitoring.GetMetrics().Reset()

	// 定义无输入的测试Flow
	testFlow := func(ctx context.Context) (string, error) {
		return "result", nil
	}

	// 包装Flow
	wrappedFlow := MonitorFlowNoInput("testFlowNoInput", testFlow)

	// 执行Flow
	ctx := context.Background()
	output, err := wrappedFlow(ctx)

	// 验证结果
	if err != nil {
		t.Errorf("期望没有错误，实际为%v", err)
	}

	if output != "result" {
		t.Errorf("期望输出为'result'，实际为'%s'", output)
	}

	// 验证监控指标
	metrics := monitoring.GetMetrics().GetFlowMetrics("testFlowNoInput")
	if metrics.Executions != 1 {
		t.Errorf("期望执行次数为1，实际为%d", metrics.Executions)
	}
}

// TestMonitorFlowNoOutput 测试无返回值的Flow监控
func TestMonitorFlowNoOutput(t *testing.T) {
	// 重置监控指标
	monitoring.GetMetrics().Reset()

	// 定义无返回值的测试Flow
	testFlow := func(ctx context.Context, input string) error {
		// 执行一些操作
		return nil
	}

	// 包装Flow
	wrappedFlow := MonitorFlowNoOutput("testFlowNoOutput", testFlow)

	// 执行Flow
	ctx := context.Background()
	err := wrappedFlow(ctx, "test input")

	// 验证结果
	if err != nil {
		t.Errorf("期望没有错误，实际为%v", err)
	}

	// 验证监控指标
	metrics := monitoring.GetMetrics().GetFlowMetrics("testFlowNoOutput")
	if metrics.Executions != 1 {
		t.Errorf("期望执行次数为1，实际为%d", metrics.Executions)
	}
}

// TestWrapFlowFunc 测试便捷包装函数
func TestWrapFlowFunc(t *testing.T) {
	// 重置监控指标
	monitoring.GetMetrics().Reset()

	// 定义测试Flow
	testFlow := func(ctx context.Context, input int) (int, error) {
		return input * 2, nil
	}

	// 使用便捷函数包装
	wrappedFlow := WrapFlowFunc("testWrapFlow", testFlow)

	// 执行Flow
	ctx := context.Background()
	output, err := wrappedFlow(ctx, 5)

	// 验证结果
	if err != nil {
		t.Errorf("期望没有错误，实际为%v", err)
	}

	if output != 10 {
		t.Errorf("期望输出为10，实际为%d", output)
	}

	// 验证监控指标
	metrics := monitoring.GetMetrics().GetFlowMetrics("testWrapFlow")
	if metrics.Executions != 1 {
		t.Errorf("期望执行次数为1，实际为%d", metrics.Executions)
	}
}

// TestRecordFlowMetrics 测试手动记录Flow指标
func TestRecordFlowMetrics(t *testing.T) {
	// 重置监控指标
	monitoring.GetMetrics().Reset()

	ctx := context.Background()
	duration := 100 * time.Millisecond

	// 记录成功指标
	RecordFlowMetrics(ctx, "manualFlow", "success", duration)

	// 验证指标
	metrics := monitoring.GetMetrics().GetFlowMetrics("manualFlow")
	if metrics.Executions != 1 {
		t.Errorf("期望执行次数为1，实际为%d", metrics.Executions)
	}

	if metrics.Successes != 1 {
		t.Errorf("期望成功次数为1，实际为%d", metrics.Successes)
	}

	// 记录失败指标
	RecordFlowMetrics(ctx, "manualFlow", "error", duration)

	// 验证指标
	metrics = monitoring.GetMetrics().GetFlowMetrics("manualFlow")
	if metrics.Executions != 2 {
		t.Errorf("期望执行次数为2，实际为%d", metrics.Executions)
	}

	if metrics.Errors != 1 {
		t.Errorf("期望错误次数为1，实际为%d", metrics.Errors)
	}
}

// BenchmarkMonitorFlowWithInput 基准测试：监控开销
func BenchmarkMonitorFlowWithInput(b *testing.B) {
	// 定义简单的测试Flow
	testFlow := func(ctx context.Context, input string) (string, error) {
		return input, nil
	}

	// 包装Flow
	wrappedFlow := MonitorFlowWithInput("benchFlow", testFlow)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = wrappedFlow(ctx, "test")
	}
}

// BenchmarkFlowWithoutMonitoring 基准测试：无监控的Flow
func BenchmarkFlowWithoutMonitoring(b *testing.B) {
	// 定义简单的测试Flow（无监控）
	testFlow := func(ctx context.Context, input string) (string, error) {
		return input, nil
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = testFlow(ctx, "test")
	}
}
