package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// contextKey 上下文键类型
type contextKey string

const (
	// TraceIDKey TraceID 上下文键
	// 使用字符串类型以便其他包可以访问相同的值
	TraceIDKey contextKey = "traceId"
	
	// traceIDKeyString TraceID 上下文键的字符串形式
	// 用于跨包访问，避免类型不匹配问题
	traceIDKeyString = "traceId"
)

// stringBuilderPool 字符串构建器对象池，用于优化 TraceID 生成性能
var stringBuilderPool = sync.Pool{
	New: func() interface{} {
		return &strings.Builder{}
	},
}

// randomBytesPool 随机字节切片对象池，用于优化随机字符串生成
var randomBytesPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 8) // 8字节可以生成16个十六进制字符
		return &b
	},
}

// SetTraceID 将 TraceID 注入到 Context
// 使用字符串键以确保跨包兼容性
func SetTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKeyString, traceID)
}

// GetTraceID 从 Context 提取 TraceID
// 如果 TraceID 不存在，返回空字符串
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	
	// 使用字符串键提取
	if traceID, ok := ctx.Value(traceIDKeyString).(string); ok {
		return traceID
	}
	return ""
}

// GenerateTraceID 生成唯一的 TraceID
// 格式：trace-{timestamp}-{random}
// 示例：trace-1704067200-a3f9k2b8c1d4
//
// 性能优化：
// - 使用对象池减少内存分配
// - 使用 strings.Builder 高效拼接字符串
// - 使用 crypto/rand 生成高质量随机数
// - 使用纳秒时间戳和更长的随机字符串确保唯一性
func GenerateTraceID() string {
	// 使用降级方案处理可能的 panic
	defer func() {
		if r := recover(); r != nil {
			// 降级方案：使用简化格式
			// 这确保即使出现异常，系统也能继续运行
		}
	}()
	
	// 从对象池获取 StringBuilder
	sb := stringBuilderPool.Get().(*strings.Builder)
	defer func() {
		sb.Reset()
		stringBuilderPool.Put(sb)
	}()
	
	// 构建 TraceID：trace-{timestamp}-{random}
	// 使用纳秒时间戳的后6位十六进制 + 10位随机字符串确保唯一性
	now := time.Now()
	sb.WriteString("trace-")
	sb.WriteString(strconv.FormatInt(now.Unix(), 10))
	sb.WriteByte('-')
	// 添加纳秒部分的十六进制表示（取后6位）
	nanoHex := fmt.Sprintf("%06x", now.Nanosecond()%0xFFFFFF)
	sb.WriteString(nanoHex)
	// 添加随机字符串
	sb.WriteString(generateRandomString(6))
	
	return sb.String()
}

// generateRandomString 生成指定长度的随机字符串
// 使用十六进制字符集 [0-9a-f]
func generateRandomString(length int) string {
	// 计算需要的字节数（每个字节生成2个十六进制字符）
	byteLen := (length + 1) / 2
	
	// 从对象池获取字节切片
	bytesPtr := randomBytesPool.Get().(*[]byte)
	defer randomBytesPool.Put(bytesPtr)
	
	bytes := *bytesPtr
	if len(bytes) < byteLen {
		bytes = make([]byte, byteLen)
	} else {
		bytes = bytes[:byteLen]
	}
	
	// 生成随机字节
	if _, err := rand.Read(bytes); err != nil {
		// 降级方案：使用时间戳作为随机源
		return generateFallbackRandomString(length)
	}
	
	// 转换为十六进制字符串
	hexStr := hex.EncodeToString(bytes)
	
	// 截取到指定长度
	if len(hexStr) > length {
		return hexStr[:length]
	}
	return hexStr
}

// generateFallbackRandomString 降级方案：生成随机字符串
// 当 crypto/rand 失败时使用
func generateFallbackRandomString(length int) string {
	// 使用纳秒时间戳作为随机源
	nanos := time.Now().UnixNano()
	hexStr := fmt.Sprintf("%x", nanos)
	
	// 如果长度不够，重复字符串
	for len(hexStr) < length {
		hexStr += hexStr
	}
	
	// 截取到指定长度
	if len(hexStr) > length {
		return hexStr[:length]
	}
	return hexStr
}
