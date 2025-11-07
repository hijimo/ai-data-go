package genkit

import (
	"context"
	"fmt"
	"time"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/monitoring"
)

// FlowMonitor Flow监控器
// 用于包装Flow执行，记录执行时间、成功率等监控指标
type FlowMonitor struct {
	metrics *monitoring.Metrics
}

// NewFlowMonitor 创建新的Flow监控器
func NewFlowMonitor() *FlowMonitor {
	return &FlowMonitor{
		metrics: monitoring.GetMetrics(),
	}
}

// MonitorFlow 包装Flow执行并记录监控指标
// 这是一个高阶函数，接收一个Flow函数并返回一个包装后的Flow函数
//
// 参数：
//   - flowName: Flow名称，用于标识和记录指标
//   - flowFunc: 原始的Flow函数
//
// 返回：
//   - 包装后的Flow函数，具有相同的签名
//
// 示例用法：
//
//	monitor := NewFlowMonitor()
//	wrappedFlow := monitor.MonitorFlow("contextBuildFlow", func(ctx context.Context, input ContextBuildInput) (ContextBuildOutput, error) {
//	    // 原始Flow逻辑
//	    return output, nil
//	})
func (m *FlowMonitor) MonitorFlow(flowName string, flowFunc interface{}) interface{} {
	// 注意：由于Go的类型系统限制，这里使用interface{}
	// 实际使用时需要根据具体的Flow签名进行类型断言
	// 为了类型安全，我们提供了针对不同签名的专用方法
	return flowFunc
}

// MonitorFlowWithInput 监控带有输入输出的Flow
// 这是一个泛型方法，适用于大多数Flow场景
//
// 类型参数：
//   - TInput: Flow输入类型
//   - TOutput: Flow输出类型
//
// 参数：
//   - flowName: Flow名称
//   - flowFunc: 原始Flow函数
//
// 返回：
//   - 包装后的Flow函数
func MonitorFlowWithInput[TInput any, TOutput any](
	flowName string,
	flowFunc func(context.Context, TInput) (TOutput, error),
) func(context.Context, TInput) (TOutput, error) {
	monitor := NewFlowMonitor()

	return func(ctx context.Context, input TInput) (TOutput, error) {
		// 1. 记录开始时间
		startTime := time.Now()

		// 2. 记录Flow开始执行
		logger.InfoContext(ctx, "Flow开始执行", logger.Fields{
			"flow_name": flowName,
			"timestamp": startTime.Format(time.RFC3339),
		})

		// 3. 执行原始Flow
		output, err := flowFunc(ctx, input)

		// 4. 计算执行时间
		duration := time.Since(startTime)

		// 5. 确定执行状态
		status := "success"
		if err != nil {
			status = "error"
		}

		// 6. 记录执行结果
		logFields := logger.Fields{
			"flow_name":   flowName,
			"status":      status,
			"duration_ms": duration.Milliseconds(),
			"timestamp":   time.Now().Format(time.RFC3339),
		}

		if err != nil {
			logFields["error"] = err.Error()
			logger.ErrorContext(ctx, "Flow执行失败", logFields)
		} else {
			logger.InfoContext(ctx, "Flow执行成功", logFields)
		}

		// 7. 记录监控指标
		monitor.recordMetrics(flowName, status, duration)

		return output, err
	}
}

// MonitorFlowNoInput 监控无输入参数的Flow
// 适用于不需要输入参数的Flow
//
// 类型参数：
//   - TOutput: Flow输出类型
//
// 参数：
//   - flowName: Flow名称
//   - flowFunc: 原始Flow函数
//
// 返回：
//   - 包装后的Flow函数
func MonitorFlowNoInput[TOutput any](
	flowName string,
	flowFunc func(context.Context) (TOutput, error),
) func(context.Context) (TOutput, error) {
	monitor := NewFlowMonitor()

	return func(ctx context.Context) (TOutput, error) {
		startTime := time.Now()

		logger.InfoContext(ctx, "Flow开始执行", logger.Fields{
			"flow_name": flowName,
			"timestamp": startTime.Format(time.RFC3339),
		})

		output, err := flowFunc(ctx)
		duration := time.Since(startTime)

		status := "success"
		if err != nil {
			status = "error"
		}

		logFields := logger.Fields{
			"flow_name":   flowName,
			"status":      status,
			"duration_ms": duration.Milliseconds(),
		}

		if err != nil {
			logFields["error"] = err.Error()
			logger.ErrorContext(ctx, "Flow执行失败", logFields)
		} else {
			logger.InfoContext(ctx, "Flow执行成功", logFields)
		}

		monitor.recordMetrics(flowName, status, duration)

		return output, err
	}
}

// MonitorFlowNoOutput 监控无返回值的Flow
// 适用于只执行操作不返回数据的Flow
//
// 类型参数：
//   - TInput: Flow输入类型
//
// 参数：
//   - flowName: Flow名称
//   - flowFunc: 原始Flow函数
//
// 返回：
//   - 包装后的Flow函数
func MonitorFlowNoOutput[TInput any](
	flowName string,
	flowFunc func(context.Context, TInput) error,
) func(context.Context, TInput) error {
	monitor := NewFlowMonitor()

	return func(ctx context.Context, input TInput) error {
		startTime := time.Now()

		logger.InfoContext(ctx, "Flow开始执行", logger.Fields{
			"flow_name": flowName,
			"timestamp": startTime.Format(time.RFC3339),
		})

		err := flowFunc(ctx, input)
		duration := time.Since(startTime)

		status := "success"
		if err != nil {
			status = "error"
		}

		logFields := logger.Fields{
			"flow_name":   flowName,
			"status":      status,
			"duration_ms": duration.Milliseconds(),
		}

		if err != nil {
			logFields["error"] = err.Error()
			logger.ErrorContext(ctx, "Flow执行失败", logFields)
		} else {
			logger.InfoContext(ctx, "Flow执行成功", logFields)
		}

		monitor.recordMetrics(flowName, status, duration)

		return err
	}
}

// recordMetrics 记录监控指标
// 这是一个内部方法，用于统一记录Flow执行的监控指标
func (m *FlowMonitor) recordMetrics(flowName string, status string, duration time.Duration) {
	// 记录Flow执行次数和状态
	m.metrics.RecordFlowExecution(flowName, status)

	// 记录Flow执行时间
	m.metrics.RecordFlowDuration(flowName, duration)
}

// WrapFlowFunc 通用的Flow包装函数
// 这是一个便捷函数，用于快速包装Flow并添加监控
//
// 参数：
//   - flowName: Flow名称
//   - flowFunc: 原始Flow函数
//
// 返回：
//   - 包装后的Flow函数
//
// 注意：这个函数使用了类型推断，适用于标准的Flow签名
func WrapFlowFunc[TInput any, TOutput any](
	flowName string,
	flowFunc func(context.Context, TInput) (TOutput, error),
) func(context.Context, TInput) (TOutput, error) {
	return MonitorFlowWithInput(flowName, flowFunc)
}

// RecordFlowMetrics 记录Flow执行指标的便捷函数
// 可以在Flow内部手动调用，用于记录额外的指标
//
// 参数：
//   - ctx: 上下文
//   - flowName: Flow名称
//   - status: 执行状态（"success" 或 "error"）
//   - duration: 执行时长
//
// 示例用法：
//
//	startTime := time.Now()
//	// ... Flow逻辑 ...
//	RecordFlowMetrics(ctx, "myFlow", "success", time.Since(startTime))
func RecordFlowMetrics(ctx context.Context, flowName string, status string, duration time.Duration) {
	monitor := NewFlowMonitor()
	monitor.recordMetrics(flowName, status, duration)

	// 记录详细日志
	logger.InfoContext(ctx, "Flow指标已记录", logger.Fields{
		"flow_name":   flowName,
		"status":      status,
		"duration_ms": duration.Milliseconds(),
	})
}

// FlowExecutionInfo Flow执行信息
// 用于在Flow执行过程中传递和记录额外的元数据
type FlowExecutionInfo struct {
	FlowName  string
	StartTime time.Time
	Status    string
	Error     error
	Metadata  map[string]interface{}
}

// NewFlowExecutionInfo 创建新的Flow执行信息
func NewFlowExecutionInfo(flowName string) *FlowExecutionInfo {
	return &FlowExecutionInfo{
		FlowName:  flowName,
		StartTime: time.Now(),
		Status:    "running",
		Metadata:  make(map[string]interface{}),
	}
}

// Complete 标记Flow执行完成
func (info *FlowExecutionInfo) Complete(err error) {
	if err != nil {
		info.Status = "error"
		info.Error = err
	} else {
		info.Status = "success"
	}
}

// Duration 获取执行时长
func (info *FlowExecutionInfo) Duration() time.Duration {
	return time.Since(info.StartTime)
}

// AddMetadata 添加元数据
func (info *FlowExecutionInfo) AddMetadata(key string, value interface{}) {
	info.Metadata[key] = value
}

// Record 记录执行信息到监控系统
func (info *FlowExecutionInfo) Record(ctx context.Context) {
	RecordFlowMetrics(ctx, info.FlowName, info.Status, info.Duration())

	// 记录详细的执行信息
	logFields := logger.Fields{
		"flow_name":   info.FlowName,
		"status":      info.Status,
		"duration_ms": info.Duration().Milliseconds(),
	}

	// 添加元数据到日志
	for k, v := range info.Metadata {
		logFields[k] = fmt.Sprintf("%v", v)
	}

	if info.Error != nil {
		logFields["error"] = info.Error.Error()
		logger.ErrorContext(ctx, "Flow执行完成（失败）", logFields)
	} else {
		logger.InfoContext(ctx, "Flow执行完成（成功）", logFields)
	}
}
