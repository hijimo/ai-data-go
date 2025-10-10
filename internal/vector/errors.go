package vector

import (
	"errors"
	"fmt"
)

// 预定义错误
var (
	// 连接相关错误
	ErrConnectionFailed    = errors.New("failed to connect to vector database")
	ErrConnectionTimeout   = errors.New("connection timeout")
	ErrConnectionClosed    = errors.New("connection is closed")
	
	// 索引相关错误
	ErrIndexNotFound       = errors.New("index not found")
	ErrIndexAlreadyExists  = errors.New("index already exists")
	ErrIndexCreationFailed = errors.New("failed to create index")
	ErrIndexDeletionFailed = errors.New("failed to delete index")
	ErrInvalidIndexName    = errors.New("invalid index name")
	
	// 向量相关错误
	ErrVectorNotFound      = errors.New("vector not found")
	ErrInvalidVector       = errors.New("invalid vector")
	ErrInvalidDimension    = errors.New("invalid vector dimension")
	ErrEmptyVector         = errors.New("vector cannot be empty")
	ErrVectorInsertFailed  = errors.New("failed to insert vector")
	ErrVectorUpdateFailed  = errors.New("failed to update vector")
	ErrVectorDeleteFailed  = errors.New("failed to delete vector")
	
	// 搜索相关错误
	ErrSearchFailed        = errors.New("vector search failed")
	ErrInvalidSearchParams = errors.New("invalid search parameters")
	ErrInvalidTopK         = errors.New("invalid topK value")
	
	// 配置相关错误
	ErrInvalidConfig       = errors.New("invalid configuration")
	ErrMissingConfig       = errors.New("missing required configuration")
	ErrUnsupportedProvider = errors.New("unsupported provider")
	
	// 操作相关错误
	ErrOperationTimeout    = errors.New("operation timeout")
	ErrOperationCancelled  = errors.New("operation cancelled")
	ErrBatchSizeExceeded   = errors.New("batch size exceeded")
	ErrQuotaExceeded       = errors.New("quota exceeded")
)

// VectorError 向量操作错误
type VectorError struct {
	Op       string // 操作名称
	Provider string // 提供商名称
	Index    string // 索引名称
	Err      error  // 原始错误
}

func (e *VectorError) Error() string {
	if e.Index != "" {
		return fmt.Sprintf("vector %s failed on %s[%s]: %v", e.Op, e.Provider, e.Index, e.Err)
	}
	return fmt.Sprintf("vector %s failed on %s: %v", e.Op, e.Provider, e.Err)
}

func (e *VectorError) Unwrap() error {
	return e.Err
}

// NewVectorError 创建向量错误
func NewVectorError(op, provider, index string, err error) *VectorError {
	return &VectorError{
		Op:       op,
		Provider: provider,
		Index:    index,
		Err:      err,
	}
}

// IsConnectionError 检查是否为连接错误
func IsConnectionError(err error) bool {
	return errors.Is(err, ErrConnectionFailed) ||
		   errors.Is(err, ErrConnectionTimeout) ||
		   errors.Is(err, ErrConnectionClosed)
}

// IsIndexError 检查是否为索引错误
func IsIndexError(err error) bool {
	return errors.Is(err, ErrIndexNotFound) ||
		   errors.Is(err, ErrIndexAlreadyExists) ||
		   errors.Is(err, ErrIndexCreationFailed) ||
		   errors.Is(err, ErrIndexDeletionFailed) ||
		   errors.Is(err, ErrInvalidIndexName)
}

// IsVectorError 检查是否为向量错误
func IsVectorError(err error) bool {
	return errors.Is(err, ErrVectorNotFound) ||
		   errors.Is(err, ErrInvalidVector) ||
		   errors.Is(err, ErrInvalidDimension) ||
		   errors.Is(err, ErrEmptyVector) ||
		   errors.Is(err, ErrVectorInsertFailed) ||
		   errors.Is(err, ErrVectorUpdateFailed) ||
		   errors.Is(err, ErrVectorDeleteFailed)
}

// IsSearchError 检查是否为搜索错误
func IsSearchError(err error) bool {
	return errors.Is(err, ErrSearchFailed) ||
		   errors.Is(err, ErrInvalidSearchParams) ||
		   errors.Is(err, ErrInvalidTopK)
}

// IsConfigError 检查是否为配置错误
func IsConfigError(err error) bool {
	return errors.Is(err, ErrInvalidConfig) ||
		   errors.Is(err, ErrMissingConfig) ||
		   errors.Is(err, ErrUnsupportedProvider)
}

// IsRetryableError 检查错误是否可重试
func IsRetryableError(err error) bool {
	return errors.Is(err, ErrConnectionTimeout) ||
		   errors.Is(err, ErrOperationTimeout) ||
		   errors.Is(err, ErrConnectionFailed)
}