package storage

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/google/uuid"
)

// OSSClient OSS客户端接口
type OSSClient interface {
	// UploadFile 上传文件
	UploadFile(ctx context.Context, file multipart.File, filename string, projectID uuid.UUID) (*UploadResult, error)
	// DeleteFile 删除文件
	DeleteFile(ctx context.Context, ossPath string) error
	// GetFileURL 获取文件访问URL
	GetFileURL(ctx context.Context, ossPath string, expiry time.Duration) (string, error)
	// CheckFileExists 检查文件是否存在
	CheckFileExists(ctx context.Context, ossPath string) (bool, error)
}

// UploadResult 上传结果
type UploadResult struct {
	OSSPath     string `json:"oss_path"`     // OSS路径
	SHA256      string `json:"sha256"`       // 文件SHA256哈希
	Size        int64  `json:"size"`         // 文件大小
	ContentType string `json:"content_type"` // 文件MIME类型
}

// OSSConfig OSS配置
type OSSConfig struct {
	Endpoint        string `json:"endpoint"`
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	BucketName      string `json:"bucket_name"`
	Region          string `json:"region"`
}

// ossClient OSS客户端实现
type ossClient struct {
	client *oss.Client
	config *OSSConfig
}

// NewOSSClient 创建OSS客户端
func NewOSSClient(config *OSSConfig) (OSSClient, error) {
	// 使用V2 SDK创建OSS客户端
	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewStaticCredentialsProvider(config.AccessKeyID, config.AccessKeySecret)).
		WithRegion(config.Region)

	// 如果指定了Endpoint，则使用自定义Endpoint
	if config.Endpoint != "" {
		cfg = cfg.WithEndpoint(config.Endpoint)
	}

	client := oss.NewClient(cfg)

	return &ossClient{
		client: client,
		config: config,
	}, nil
}

// NewOSSClientFromEnv 从环境变量创建OSS客户端
func NewOSSClientFromEnv(bucketName, region string) (OSSClient, error) {
	// 使用环境变量凭证提供者
	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewEnvironmentVariableCredentialsProvider()).
		WithRegion(region)

	client := oss.NewClient(cfg)

	config := &OSSConfig{
		BucketName: bucketName,
		Region:     region,
	}

	return &ossClient{
		client: client,
		config: config,
	}, nil
}

// UploadFile 上传文件
func (c *ossClient) UploadFile(ctx context.Context, file multipart.File, filename string, projectID uuid.UUID) (*UploadResult, error) {
	// 重置文件指针到开始位置
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("重置文件指针失败: %w", err)
	}

	// 读取文件内容并计算SHA256
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("读取文件内容失败: %w", err)
	}

	// 计算SHA256哈希
	hash := sha256.Sum256(content)
	sha256Hash := fmt.Sprintf("%x", hash)

	// 生成OSS路径
	ossPath := c.generateOSSPath(projectID, filename, sha256Hash)

	// 检查文件是否已存在（去重）
	exists, err := c.CheckFileExists(ctx, ossPath)
	if err != nil {
		return nil, fmt.Errorf("检查文件是否存在失败: %w", err)
	}

	if !exists {
		// 文件不存在，执行上传
		request := &oss.PutObjectRequest{
			Bucket: oss.Ptr(c.config.BucketName),
			Key:    oss.Ptr(ossPath),
			Body:   strings.NewReader(string(content)),
		}

		_, err = c.client.PutObject(ctx, request)
		if err != nil {
			return nil, fmt.Errorf("上传文件到OSS失败: %w", err)
		}
	}

	// 检测文件MIME类型
	contentType := c.detectContentType(filename, content)

	return &UploadResult{
		OSSPath:     ossPath,
		SHA256:      sha256Hash,
		Size:        int64(len(content)),
		ContentType: contentType,
	}, nil
}

// DeleteFile 删除文件
func (c *ossClient) DeleteFile(ctx context.Context, ossPath string) error {
	request := &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(c.config.BucketName),
		Key:    oss.Ptr(ossPath),
	}

	_, err := c.client.DeleteObject(ctx, request)
	if err != nil {
		return fmt.Errorf("删除OSS文件失败: %w", err)
	}
	return nil
}

// GetFileURL 获取文件访问URL
func (c *ossClient) GetFileURL(ctx context.Context, ossPath string, expiry time.Duration) (string, error) {
	request := &oss.GetObjectRequest{
		Bucket: oss.Ptr(c.config.BucketName),
		Key:    oss.Ptr(ossPath),
	}

	presigner := oss.NewPresigner(c.client)
	result, err := presigner.PresignGetObject(ctx, request, func(options *oss.PresignOptions) {
		options.Expiry = expiry
	})
	if err != nil {
		return "", fmt.Errorf("生成文件访问URL失败: %w", err)
	}
	return result.URL, nil
}

// CheckFileExists 检查文件是否存在
func (c *ossClient) CheckFileExists(ctx context.Context, ossPath string) (bool, error) {
	request := &oss.HeadObjectRequest{
		Bucket: oss.Ptr(c.config.BucketName),
		Key:    oss.Ptr(ossPath),
	}

	_, err := c.client.HeadObject(ctx, request)
	if err != nil {
		// 检查是否是404错误（文件不存在）
		if strings.Contains(err.Error(), "NoSuchKey") || strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, fmt.Errorf("检查文件是否存在失败: %w", err)
	}
	return true, nil
}

// generateOSSPath 生成OSS路径
func (c *ossClient) generateOSSPath(projectID uuid.UUID, filename, sha256Hash string) string {
	// 使用项目ID和SHA256前8位作为目录结构
	prefix := sha256Hash[:8]
	ext := filepath.Ext(filename)
	
	// 格式: projects/{projectID}/files/{sha256前8位}/{sha256}{扩展名}
	return fmt.Sprintf("projects/%s/files/%s/%s%s", projectID.String(), prefix, sha256Hash, ext)
}

// detectContentType 检测文件MIME类型
func (c *ossClient) detectContentType(filename string, content []byte) string {
	// 根据文件扩展名判断MIME类型
	ext := strings.ToLower(filepath.Ext(filename))
	
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".txt":
		return "text/plain"
	case ".md":
		return "text/markdown"
	case ".html", ".htm":
		return "text/html"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".csv":
		return "text/csv"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	default:
		// 尝试从文件内容检测
		if len(content) > 0 {
			// 检查PDF文件头
			if len(content) >= 4 && string(content[:4]) == "%PDF" {
				return "application/pdf"
			}
			// 检查Office文档
			if len(content) >= 8 && string(content[:8]) == "\xd0\xcf\x11\xe0\xa1\xb1\x1a\xe1" {
				return "application/msword"
			}
			// 检查ZIP格式（可能是DOCX等）
			if len(content) >= 4 && string(content[:4]) == "PK\x03\x04" {
				return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
			}
		}
		return "application/octet-stream"
	}
}

// MockOSSClient 用于测试的Mock客户端
type MockOSSClient struct {
	files map[string][]byte
}

// NewMockOSSClient 创建Mock OSS客户端
func NewMockOSSClient() OSSClient {
	return &MockOSSClient{
		files: make(map[string][]byte),
	}
}

// UploadFile Mock上传文件
func (m *MockOSSClient) UploadFile(ctx context.Context, file multipart.File, filename string, projectID uuid.UUID) (*UploadResult, error) {
	// 重置文件指针
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	// 读取文件内容
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	// 计算SHA256
	hash := sha256.Sum256(content)
	sha256Hash := fmt.Sprintf("%x", hash)

	// 生成路径
	ossPath := fmt.Sprintf("projects/%s/files/%s/%s%s", 
		projectID.String(), 
		sha256Hash[:8], 
		sha256Hash, 
		filepath.Ext(filename))

	// 存储到内存
	m.files[ossPath] = content

	return &UploadResult{
		OSSPath:     ossPath,
		SHA256:      sha256Hash,
		Size:        int64(len(content)),
		ContentType: "application/octet-stream",
	}, nil
}

// DeleteFile Mock删除文件
func (m *MockOSSClient) DeleteFile(ctx context.Context, ossPath string) error {
	delete(m.files, ossPath)
	return nil
}

// GetFileURL Mock获取文件URL
func (m *MockOSSClient) GetFileURL(ctx context.Context, ossPath string, expiry time.Duration) (string, error) {
	if _, exists := m.files[ossPath]; !exists {
		return "", fmt.Errorf("文件不存在")
	}
	return fmt.Sprintf("http://mock-oss.com/%s", ossPath), nil
}

// CheckFileExists Mock检查文件是否存在
func (m *MockOSSClient) CheckFileExists(ctx context.Context, ossPath string) (bool, error) {
	_, exists := m.files[ossPath]
	return exists, nil
}