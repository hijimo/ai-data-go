package flows

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/service/auth"
	"genkit-ai-service/internal/service/session"
)

// MockSummaryService 是SummaryService的mock实现
type MockSummaryService struct {
	mock.Mock
}

func (m *MockSummaryService) GenerateSummary(ctx context.Context, req *session.GenerateSummaryRequest) (*model.ConversationSummary, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ConversationSummary), args.Error(1)
}

func (m *MockSummaryService) CheckSummaryTrigger(ctx context.Context, tenantID, sessionID uuid.UUID) (*session.SummaryTriggerResult, error) {
	args := m.Called(ctx, tenantID, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.SummaryTriggerResult), args.Error(1)
}

func (m *MockSummaryService) EvaluateSummaryQuality(ctx context.Context, req *session.EvaluateSummaryRequest) (*session.SummaryQualityResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.SummaryQualityResult), args.Error(1)
}

func (m *MockSummaryService) GetSummary(ctx context.Context, tenantID, summaryID uuid.UUID) (*model.ConversationSummary, error) {
	args := m.Called(ctx, tenantID, summaryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ConversationSummary), args.Error(1)
}

func (m *MockSummaryService) ListSummaries(ctx context.Context, tenantID, sessionID uuid.UUID, limit int) ([]*model.ConversationSummary, error) {
	args := m.Called(ctx, tenantID, sessionID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ConversationSummary), args.Error(1)
}

// createTestContext 创建带有JWT声明的测试上下文
func createTestContext(tenantID uuid.UUID) context.Context {
	ctx := context.Background()
	claims := &model.JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: uuid.New().String(),
		},
		TenantID: tenantID.String(),
		Roles:    []string{"tenant_admin"},
	}
	return context.WithValue(ctx, auth.JWTClaimsContextKey, claims)
}

// TestSummaryGenerateFlow 测试摘要生成Flow
func TestSummaryGenerateFlow(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New()
	sessionID := uuid.New()
	summaryID := uuid.New()
	startMessageID := uuid.New()
	endMessageID := uuid.New()

	qualityScore := 0.85
	compressionRate := 0.65

	// 创建mock服务
	mockSvc := new(MockSummaryService)

	// 设置mock期望
	mockSvc.On("GenerateSummary", mock.Anything, mock.MatchedBy(func(req *session.GenerateSummaryRequest) bool {
		return req.SessionID == sessionID && req.SummaryType == "full"
	})).Return(&model.ConversationSummary{
		ID:              summaryID,
		TenantID:        tenantID,
		SessionID:       sessionID,
		SummaryType:     "full",
		Content:         "这是一个测试摘要，包含了对话的主要内容和关键信息。",
		TokenCount:      50,
		MessageCount:    10,
		StartMessageID:  &startMessageID,
		EndMessageID:    &endMessageID,
		QualityScore:    &qualityScore,
		CompressionRate: &compressionRate,
		KeyTopics:       []string{"测试", "摘要", "对话"},
	}, nil)

	// 创建带有JWT声明的上下文
	ctx := createTestContext(tenantID)

	// 创建Flow函数
	flowFunc := summaryGenerateFlow(mockSvc)

	// 准备输入
	input := SummaryGenerateInput{
		SessionID:    sessionID.String(),
		SummaryType:  "full",
		TargetLength: 100,
	}

	// 执行Flow
	output, err := flowFunc(ctx, input)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, summaryID.String(), output.SummaryID)
	assert.Equal(t, "这是一个测试摘要，包含了对话的主要内容和关键信息。", output.Summary)
	assert.Equal(t, 50, output.TokenCount)
	assert.Equal(t, 10, output.MessageCount)
	assert.Equal(t, startMessageID.String(), output.StartMessageID)
	assert.Equal(t, endMessageID.String(), output.EndMessageID)
	assert.Equal(t, 0.85, output.QualityScore)
	assert.Equal(t, 0.65, output.CompressionRate)
	assert.Equal(t, []string{"测试", "摘要", "对话"}, output.KeyTopics)
	assert.GreaterOrEqual(t, output.GenerationTime, int64(0))

	// 验证mock调用
	mockSvc.AssertExpectations(t)
}

// TestSummaryGenerateFlowWithMessageIDs 测试带消息ID列表的摘要生成
func TestSummaryGenerateFlowWithMessageIDs(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New()
	sessionID := uuid.New()
	summaryID := uuid.New()
	messageID1 := uuid.New()
	messageID2 := uuid.New()
	messageID3 := uuid.New()

	qualityScore := 0.90
	compressionRate := 0.70

	// 创建mock服务
	mockSvc := new(MockSummaryService)

	// 设置mock期望
	mockSvc.On("GenerateSummary", mock.Anything, mock.MatchedBy(func(req *session.GenerateSummaryRequest) bool {
		return req.SessionID == sessionID &&
			req.SummaryType == "incremental" &&
			len(req.MessageIDs) == 3
	})).Return(&model.ConversationSummary{
		ID:              summaryID,
		TenantID:        tenantID,
		SessionID:       sessionID,
		SummaryType:     "incremental",
		Content:         "增量摘要内容",
		TokenCount:      30,
		MessageCount:    3,
		QualityScore:    &qualityScore,
		CompressionRate: &compressionRate,
		KeyTopics:       []string{"增量", "更新"},
	}, nil)

	// 创建带有JWT声明的上下文
	ctx := createTestContext(tenantID)

	// 创建Flow函数
	flowFunc := summaryGenerateFlow(mockSvc)

	// 准备输入
	input := SummaryGenerateInput{
		SessionID: sessionID.String(),
		MessageIDs: []string{
			messageID1.String(),
			messageID2.String(),
			messageID3.String(),
		},
		PreviousSummary: "之前的摘要内容",
		SummaryType:     "incremental",
		TargetLength:    50,
	}

	// 执行Flow
	output, err := flowFunc(ctx, input)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, summaryID.String(), output.SummaryID)
	assert.Equal(t, "增量摘要内容", output.Summary)
	assert.Equal(t, 30, output.TokenCount)
	assert.Equal(t, 3, output.MessageCount)
	assert.Equal(t, 0.90, output.QualityScore)
	assert.Equal(t, 0.70, output.CompressionRate)

	// 验证mock调用
	mockSvc.AssertExpectations(t)
}

// TestSummaryGenerateFlowInvalidInput 测试无效输入
func TestSummaryGenerateFlowInvalidInput(t *testing.T) {
	mockSvc := new(MockSummaryService)
	flowFunc := summaryGenerateFlow(mockSvc)

	tenantID := uuid.New()
	ctx := createTestContext(tenantID)

	tests := []struct {
		name  string
		input SummaryGenerateInput
	}{
		{
			name: "空会话ID",
			input: SummaryGenerateInput{
				SessionID:    "",
				SummaryType:  "full",
				TargetLength: 100,
			},
		},
		{
			name: "无效的会话ID格式",
			input: SummaryGenerateInput{
				SessionID:    "invalid-uuid",
				SummaryType:  "full",
				TargetLength: 100,
			},
		},
		{
			name: "无效的摘要类型",
			input: SummaryGenerateInput{
				SessionID:    uuid.New().String(),
				SummaryType:  "invalid",
				TargetLength: 100,
			},
		},
		{
			name: "目标长度过小",
			input: SummaryGenerateInput{
				SessionID:    uuid.New().String(),
				SummaryType:  "full",
				TargetLength: 10,
			},
		},
		{
			name: "目标长度过大",
			input: SummaryGenerateInput{
				SessionID:    uuid.New().String(),
				SummaryType:  "full",
				TargetLength: 2000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := flowFunc(ctx, tt.input)
			assert.Error(t, err)
		})
	}
}

// TestSummaryGenerateFlowUnauthorized 测试未认证请求
func TestSummaryGenerateFlowUnauthorized(t *testing.T) {
	mockSvc := new(MockSummaryService)
	flowFunc := summaryGenerateFlow(mockSvc)

	// 创建没有JWT声明的上下文
	ctx := context.Background()

	input := SummaryGenerateInput{
		SessionID:    uuid.New().String(),
		SummaryType:  "full",
		TargetLength: 100,
	}

	_, err := flowFunc(ctx, input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "未认证")
}

// TestSummaryTriggerCheckFlow 测试摘要触发检查Flow
func TestSummaryTriggerCheckFlow(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New()
	sessionID := uuid.New()
	messageID1 := uuid.New()
	messageID2 := uuid.New()

	// 创建mock服务
	mockSvc := new(MockSummaryService)

	// 设置mock期望
	mockSvc.On("CheckSummaryTrigger", mock.Anything, tenantID, sessionID).Return(&session.SummaryTriggerResult{
		ShouldSummarize:      true,
		TriggerReason:        "消息数量达到阈值",
		MessageIDs:           []uuid.UUID{messageID1, messageID2},
		MessageCount:         20,
		EstimatedTokenSaving: 500,
		Urgency:              0.75,
		RecommendedType:      "incremental",
	}, nil)

	// 创建带有JWT声明的上下文
	ctx := createTestContext(tenantID)

	// 创建Flow函数
	flowFunc := summaryTriggerCheckFlow(mockSvc)

	// 准备输入
	input := SummaryTriggerCheckInput{
		SessionID: sessionID.String(),
		CheckMode: "auto",
	}

	// 执行Flow
	output, err := flowFunc(ctx, input)

	// 验证结果
	assert.NoError(t, err)
	assert.True(t, output.ShouldSummarize)
	assert.Equal(t, "消息数量达到阈值", output.TriggerReason)
	assert.Equal(t, 2, len(output.MessageIDs))
	assert.Equal(t, messageID1.String(), output.MessageIDs[0])
	assert.Equal(t, messageID2.String(), output.MessageIDs[1])
	assert.Equal(t, 20, output.MessageCount)
	assert.Equal(t, 500, output.EstimatedTokenSaving)
	assert.Equal(t, 0.75, output.Urgency)
	assert.Equal(t, "incremental", output.RecommendedType)
	assert.GreaterOrEqual(t, output.CheckTime, int64(0))

	// 验证mock调用
	mockSvc.AssertExpectations(t)
}

// TestSummaryTriggerCheckFlowForceMode 测试强制模式
func TestSummaryTriggerCheckFlowForceMode(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建mock服务
	mockSvc := new(MockSummaryService)

	// 设置mock期望 - 即使服务返回不需要摘要，强制模式也会覆盖
	mockSvc.On("CheckSummaryTrigger", mock.Anything, tenantID, sessionID).Return(&session.SummaryTriggerResult{
		ShouldSummarize:      false,
		TriggerReason:        "未达到触发条件",
		MessageIDs:           []uuid.UUID{},
		MessageCount:         5,
		EstimatedTokenSaving: 0,
		Urgency:              0.1,
		RecommendedType:      "full",
	}, nil)

	// 创建带有JWT声明的上下文
	ctx := createTestContext(tenantID)

	// 创建Flow函数
	flowFunc := summaryTriggerCheckFlow(mockSvc)

	// 准备输入 - 使用强制模式
	input := SummaryTriggerCheckInput{
		SessionID: sessionID.String(),
		CheckMode: "force",
	}

	// 执行Flow
	output, err := flowFunc(ctx, input)

	// 验证结果 - 强制模式应该覆盖服务返回的结果
	assert.NoError(t, err)
	assert.True(t, output.ShouldSummarize) // 强制模式下应该为true
	assert.Equal(t, "强制触发", output.TriggerReason)
	assert.Equal(t, 1.0, output.Urgency) // 强制模式下紧急程度为1.0

	// 验证mock调用
	mockSvc.AssertExpectations(t)
}

// TestSummaryTriggerCheckFlowInvalidInput 测试无效输入
func TestSummaryTriggerCheckFlowInvalidInput(t *testing.T) {
	mockSvc := new(MockSummaryService)
	flowFunc := summaryTriggerCheckFlow(mockSvc)

	tenantID := uuid.New()
	ctx := createTestContext(tenantID)

	tests := []struct {
		name  string
		input SummaryTriggerCheckInput
	}{
		{
			name: "空会话ID",
			input: SummaryTriggerCheckInput{
				SessionID: "",
				CheckMode: "auto",
			},
		},
		{
			name: "无效的会话ID格式",
			input: SummaryTriggerCheckInput{
				SessionID: "invalid-uuid",
				CheckMode: "auto",
			},
		},
		{
			name: "无效的检查模式",
			input: SummaryTriggerCheckInput{
				SessionID: uuid.New().String(),
				CheckMode: "invalid",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := flowFunc(ctx, tt.input)
			assert.Error(t, err)
		})
	}
}

// TestValidateSummaryGenerateInput 测试摘要生成输入验证
func TestValidateSummaryGenerateInput(t *testing.T) {
	tests := []struct {
		name    string
		input   SummaryGenerateInput
		wantErr bool
	}{
		{
			name: "有效输入",
			input: SummaryGenerateInput{
				SessionID:    uuid.New().String(),
				SummaryType:  "full",
				TargetLength: 100,
			},
			wantErr: false,
		},
		{
			name: "有效的增量摘要",
			input: SummaryGenerateInput{
				SessionID:       uuid.New().String(),
				PreviousSummary: "之前的摘要",
				SummaryType:     "incremental",
				TargetLength:    200,
			},
			wantErr: false,
		},
		{
			name: "空会话ID",
			input: SummaryGenerateInput{
				SessionID:    "",
				SummaryType:  "full",
				TargetLength: 100,
			},
			wantErr: true,
		},
		{
			name: "无效的摘要类型",
			input: SummaryGenerateInput{
				SessionID:    uuid.New().String(),
				SummaryType:  "invalid",
				TargetLength: 100,
			},
			wantErr: true,
		},
		{
			name: "目标长度过小",
			input: SummaryGenerateInput{
				SessionID:    uuid.New().String(),
				SummaryType:  "full",
				TargetLength: 10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSummaryGenerateInput(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateSummaryTriggerCheckInput 测试摘要触发检查输入验证
func TestValidateSummaryTriggerCheckInput(t *testing.T) {
	tests := []struct {
		name    string
		input   SummaryTriggerCheckInput
		wantErr bool
	}{
		{
			name: "有效输入 - auto模式",
			input: SummaryTriggerCheckInput{
				SessionID: uuid.New().String(),
				CheckMode: "auto",
			},
			wantErr: false,
		},
		{
			name: "有效输入 - force模式",
			input: SummaryTriggerCheckInput{
				SessionID: uuid.New().String(),
				CheckMode: "force",
			},
			wantErr: false,
		},
		{
			name: "空会话ID",
			input: SummaryTriggerCheckInput{
				SessionID: "",
				CheckMode: "auto",
			},
			wantErr: true,
		},
		{
			name: "无效的检查模式",
			input: SummaryTriggerCheckInput{
				SessionID: uuid.New().String(),
				CheckMode: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSummaryTriggerCheckInput(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// BenchmarkSummaryGenerateFlow 性能基准测试
func BenchmarkSummaryGenerateFlow(b *testing.B) {
	tenantID := uuid.New()
	sessionID := uuid.New()
	summaryID := uuid.New()

	qualityScore := 0.85
	compressionRate := 0.65

	mockSvc := new(MockSummaryService)
	mockSvc.On("GenerateSummary", mock.Anything, mock.Anything).Return(&model.ConversationSummary{
		ID:              summaryID,
		TenantID:        tenantID,
		SessionID:       sessionID,
		SummaryType:     "full",
		Content:         "测试摘要内容",
		TokenCount:      50,
		MessageCount:    10,
		QualityScore:    &qualityScore,
		CompressionRate: &compressionRate,
		KeyTopics:       []string{"测试"},
	}, nil)

	ctx := createTestContext(tenantID)

	flowFunc := summaryGenerateFlow(mockSvc)

	input := SummaryGenerateInput{
		SessionID:    sessionID.String(),
		SummaryType:  "full",
		TargetLength: 100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = flowFunc(ctx, input)
	}
}
