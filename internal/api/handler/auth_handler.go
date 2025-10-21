package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/service/auth"
	"genkit-ai-service/pkg/errors"
	"genkit-ai-service/pkg/response"
	"genkit-ai-service/pkg/validator"
)

// AuthHandler 认证处理器
// 提供用户注册、登录、Token 刷新、注销和密码修改等功能
type AuthHandler struct {
	authService  auth.AuthService
	emailService auth.EmailService
	logger       logger.Logger
	validator    *validator.Validator
}

// NewAuthHandler 创建认证处理器实例
// 参数：
//   - authService: 认证服务接口
//   - emailService: 邮箱服务接口
//   - log: 日志记录器
// 返回：
//   - *AuthHandler: 认证处理器实例
func NewAuthHandler(authService auth.AuthService, emailService auth.EmailService, log logger.Logger) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		emailService: emailService,
		logger:       log,
		validator:    validator.New(),
	}
}

// RegisterRequest 用户注册请求（用于 Swagger）
// @name RegisterRequest
type RegisterRequest struct {
	TenantID    string `json:"tenantId" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email       string `json:"email" validate:"required,email" example:"user@example.com"`
	Password    string `json:"password" validate:"required,min=8" example:"password123"`
	DisplayName string `json:"displayName" example:"张三"`
	Phone       string `json:"phone" example:"13800138000"`
}

// LoginRequest 用户登录请求（用于 Swagger）
// @name LoginRequest
type LoginRequest struct {
	TenantID string `json:"tenantId" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email    string `json:"email" validate:"required,email" example:"user@example.com"`
	Password string `json:"password" validate:"required" example:"password123"`
}

// RefreshRequest Token 刷新请求（用于 Swagger）
// @name RefreshRequest
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// LogoutRequest 用户注销请求（用于 Swagger）
// @name LogoutRequest
type LogoutRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// ChangePasswordRequest 修改密码请求（用于 Swagger）
// @name ChangePasswordRequest
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" validate:"required" example:"oldpassword123"`
	NewPassword string `json:"newPassword" validate:"required,min=8" example:"newpassword123"`
}

// HandleRegister 处理用户注册
// @Summary 用户注册
// @Description 注册新用户账户
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "注册请求"
// @Success 201 {object} genkit-ai-service_internal_model.UserDataResponse "注册成功"
// @Failure 400 {object} genkit-ai-service_internal_model.ErrorResponse "请求参数错误"
// @Failure 422 {object} genkit-ai-service_internal_model.ErrorResponse "参数验证失败"
// @Failure 500 {object} genkit-ai-service_internal_model.ErrorResponse "服务器内部错误"
// @Router /auth/register [post]
func (h *AuthHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 解析请求参数
	var req auth.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析注册请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 2. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("注册请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, validationErrors)
		return
	}

	// 3. 记录请求日志
	h.logger.Info("收到用户注册请求", logger.Fields{
		"tenantId": req.TenantID,
		"email":    req.Email,
	})

	// 4. 调用服务层注册用户
	user, err := h.authService.Register(ctx, req)
	if err != nil {
		h.logger.Error("用户注册失败", logger.Fields{
			"error":    err,
			"tenantId": req.TenantID,
			"email":    req.Email,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewInternalError(err))
		}
		return
	}

	// 5. 记录响应日志
	h.logger.Info("用户注册成功", logger.Fields{
		"userId":   user.ID,
		"tenantId": user.TenantID,
		"email":    user.Email,
	})

	// 6. 返回成功响应（HTTP 201 Created）
	resp := response.SuccessWithMessage("注册成功", user)
	h.writeJSONResponse(w, http.StatusCreated, resp)
}

// HandleLogin 处理用户登录
// @Summary 用户登录
// @Description 使用邮箱和密码登录系统
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body LoginRequest true "登录请求"
// @Success 200 {object} genkit-ai-service_internal_model.LoginDataResponse "登录成功"
// @Failure 400 {object} genkit-ai-service_internal_model.ErrorResponse "请求参数错误"
// @Failure 401 {object} genkit-ai-service_internal_model.ErrorResponse "认证失败"
// @Failure 422 {object} genkit-ai-service_internal_model.ErrorResponse "参数验证失败"
// @Failure 500 {object} genkit-ai-service_internal_model.ErrorResponse "服务器内部错误"
// @Router /auth/login [post]
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 解析请求参数
	var req auth.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析登录请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 2. 如果请求中没有 TenantID，尝试从上下文获取
	if req.TenantID == "" {
		if tenantID, ok := ctx.Value("tenant_id").(string); ok {
			req.TenantID = tenantID
		}
	}

	// 3. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("登录请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, validationErrors)
		return
	}

	// 4. 记录请求日志
	h.logger.Info("收到用户登录请求", logger.Fields{
		"tenantId": req.TenantID,
		"email":    req.Email,
		"ip":       h.getClientIP(r),
	})

	// 5. 调用服务层登录
	loginResp, err := h.authService.Login(ctx, req)
	if err != nil {
		h.logger.Error("用户登录失败", logger.Fields{
			"error":    err,
			"tenantId": req.TenantID,
			"email":    req.Email,
		})
		// 登录失败返回 401 Unauthorized
		h.writeErrorResponse(w, errors.NewUnauthorizedError("邮箱或密码错误"))
		return
	}

	// 6. 记录响应日志
	h.logger.Info("用户登录成功", logger.Fields{
		"userId":   loginResp.User.ID,
		"tenantId": loginResp.User.TenantID,
		"email":    loginResp.User.Email,
	})

	// 7. 返回成功响应
	resp := response.SuccessWithMessage("登录成功", loginResp)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// HandleRefresh 处理 Token 刷新
// @Summary 刷新访问令牌
// @Description 使用 Refresh Token 获取新的 Access Token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "刷新请求"
// @Success 200 {object} genkit-ai-service_internal_model.LoginDataResponse "刷新成功"
// @Failure 400 {object} genkit-ai-service_internal_model.ErrorResponse "请求参数错误"
// @Failure 401 {object} genkit-ai-service_internal_model.ErrorResponse "Token 无效或已过期"
// @Failure 422 {object} genkit-ai-service_internal_model.ErrorResponse "参数验证失败"
// @Failure 500 {object} genkit-ai-service_internal_model.ErrorResponse "服务器内部错误"
// @Router /auth/refresh [post]
func (h *AuthHandler) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 解析请求参数
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析刷新请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 2. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("刷新请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, validationErrors)
		return
	}

	// 3. 记录请求日志
	h.logger.Info("收到 Token 刷新请求", logger.Fields{
		"ip": h.getClientIP(r),
	})

	// 4. 调用服务层刷新 Token
	loginResp, err := h.authService.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		h.logger.Error("Token 刷新失败", logger.Fields{
			"error": err,
		})
		// Token 无效返回 401 Unauthorized
		h.writeErrorResponse(w, errors.NewUnauthorizedError("刷新令牌无效或已过期"))
		return
	}

	// 5. 记录响应日志
	h.logger.Info("Token 刷新成功", logger.Fields{
		"userId":   loginResp.User.ID,
		"tenantId": loginResp.User.TenantID,
	})

	// 6. 返回成功响应
	resp := response.SuccessWithMessage("刷新成功", loginResp)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// HandleLogout 处理用户注销
// @Summary 用户注销
// @Description 注销用户并撤销 Refresh Token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body LogoutRequest true "注销请求"
// @Success 200 {object} genkit-ai-service_internal_model.SuccessResponse "注销成功"
// @Failure 400 {object} genkit-ai-service_internal_model.ErrorResponse "请求参数错误"
// @Failure 401 {object} genkit-ai-service_internal_model.ErrorResponse "Token 无效"
// @Failure 422 {object} genkit-ai-service_internal_model.ErrorResponse "参数验证失败"
// @Failure 500 {object} genkit-ai-service_internal_model.ErrorResponse "服务器内部错误"
// @Router /auth/logout [post]
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 解析请求参数
	var req LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析注销请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 2. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("注销请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, validationErrors)
		return
	}

	// 3. 记录请求日志
	h.logger.Info("收到用户注销请求", logger.Fields{
		"ip": h.getClientIP(r),
	})

	// 4. 从 Authorization 头提取 access token
	accessToken := ""
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			accessToken = parts[1]
		}
	}

	// 5. 调用服务层注销
	err := h.authService.Logout(ctx, accessToken, req.RefreshToken)
	if err != nil {
		h.logger.Error("用户注销失败", logger.Fields{
			"error": err,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewInternalError(err))
		}
		return
	}

	// 6. 记录响应日志
	h.logger.Info("用户注销成功", logger.Fields{})

	// 7. 返回成功响应
	emptyData := struct{}{}
	resp := response.SuccessWithMessage("注销成功", &emptyData)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// HandleChangePassword 处理密码修改
// @Summary 修改密码
// @Description 修改当前用户的密码
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ChangePasswordRequest true "修改密码请求"
// @Success 200 {object} genkit-ai-service_internal_model.SuccessResponse "修改成功"
// @Failure 400 {object} genkit-ai-service_internal_model.ErrorResponse "请求参数错误"
// @Failure 401 {object} genkit-ai-service_internal_model.ErrorResponse "未认证或旧密码错误"
// @Failure 422 {object} genkit-ai-service_internal_model.ErrorResponse "参数验证失败"
// @Failure 500 {object} genkit-ai-service_internal_model.ErrorResponse "服务器内部错误"
// @Router /auth/change-password [post]
func (h *AuthHandler) HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从上下文获取用户信息（由 JWT 中间件注入）
	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		h.logger.Warn("未找到用户ID")
		h.writeErrorResponse(w, errors.NewUnauthorizedError("未认证"))
		return
	}

	tenantID, ok := ctx.Value("tenant_id").(string)
	if !ok || tenantID == "" {
		h.logger.Warn("未找到租户ID")
		h.writeErrorResponse(w, errors.NewUnauthorizedError("未认证"))
		return
	}

	// 2. 解析请求参数
	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析修改密码请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 3. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("修改密码请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, validationErrors)
		return
	}

	// 4. 记录请求日志
	h.logger.Info("收到修改密码请求", logger.Fields{
		"userId":   userID,
		"tenantId": tenantID,
	})

	// 5. 调用服务层修改密码
	err := h.authService.ChangePassword(ctx, tenantID, userID, req.OldPassword, req.NewPassword)
	if err != nil {
		h.logger.Error("修改密码失败", logger.Fields{
			"error":    err,
			"userId":   userID,
			"tenantId": tenantID,
		})
		// 旧密码错误返回 401
		h.writeErrorResponse(w, errors.NewUnauthorizedError("旧密码错误"))
		return
	}

	// 6. 记录响应日志
	h.logger.Info("修改密码成功", logger.Fields{
		"userId":   userID,
		"tenantId": tenantID,
	})

	// 7. 返回成功响应
	emptyData := struct{}{}
	resp := response.SuccessWithMessage("密码修改成功，请重新登录", &emptyData)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// UnlockAccountRequest 解锁账户请求（用于 Swagger）
// @name UnlockAccountRequest
type UnlockAccountRequest struct {
	TenantID string `json:"tenantId" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID   string `json:"userId" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// HandleUnlockAccount 处理账户解锁
// @Summary 解锁账户
// @Description 解锁被锁定的用户账户（需要管理员权限）
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UnlockAccountRequest true "解锁请求"
// @Success 200 {object} genkit-ai-service_internal_model.SuccessResponse "解锁成功"
// @Failure 400 {object} genkit-ai-service_internal_model.ErrorResponse "请求参数错误"
// @Failure 401 {object} genkit-ai-service_internal_model.ErrorResponse "未认证"
// @Failure 403 {object} genkit-ai-service_internal_model.ErrorResponse "权限不足"
// @Failure 422 {object} genkit-ai-service_internal_model.ErrorResponse "参数验证失败"
// @Failure 500 {object} genkit-ai-service_internal_model.ErrorResponse "服务器内部错误"
// @Router /auth/unlock-account [post]
func (h *AuthHandler) HandleUnlockAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 解析请求参数
	var req UnlockAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析解锁账户请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 2. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("解锁账户请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, validationErrors)
		return
	}

	// 3. 记录请求日志
	h.logger.Info("收到解锁账户请求", logger.Fields{
		"tenantId": req.TenantID,
		"userId":   req.UserID,
	})

	// 4. 调用服务层解锁账户
	err := h.authService.UnlockAccount(ctx, req.TenantID, req.UserID)
	if err != nil {
		h.logger.Error("解锁账户失败", logger.Fields{
			"error":    err,
			"tenantId": req.TenantID,
			"userId":   req.UserID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewInternalError(err))
		}
		return
	}

	// 5. 记录响应日志
	h.logger.Info("解锁账户成功", logger.Fields{
		"tenantId": req.TenantID,
		"userId":   req.UserID,
	})

	// 6. 返回成功响应
	emptyData := struct{}{}
	resp := response.SuccessWithMessage("账户解锁成功", &emptyData)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// HandleMe 获取当前用户信息
// @Summary 获取当前用户信息
// @Description 获取当前登录用户的详细信息
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} genkit-ai-service_internal_model.UserDataResponse "获取成功"
// @Failure 401 {object} genkit-ai-service_internal_model.ErrorResponse "未认证"
// @Failure 404 {object} genkit-ai-service_internal_model.ErrorResponse "用户不存在"
// @Failure 500 {object} genkit-ai-service_internal_model.ErrorResponse "服务器内部错误"
// @Router /auth/me [get]
func (h *AuthHandler) HandleMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从上下文获取用户信息（由 JWT 中间件注入）
	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		h.logger.Warn("未找到用户ID")
		h.writeErrorResponse(w, errors.NewUnauthorizedError("未认证"))
		return
	}

	tenantID, ok := ctx.Value("tenant_id").(string)
	if !ok || tenantID == "" {
		h.logger.Warn("未找到租户ID")
		h.writeErrorResponse(w, errors.NewUnauthorizedError("未认证"))
		return
	}

	// 2. 记录请求日志
	h.logger.Info("收到获取当前用户信息请求", logger.Fields{
		"userId":   userID,
		"tenantId": tenantID,
	})

	// 3. 这里需要通过 UserService 获取用户信息
	// 由于当前 AuthHandler 只依赖 AuthService，我们可以：
	// 选项1：在 AuthService 中添加 GetUser 方法
	// 选项2：在 AuthHandler 中注入 UserService
	// 选项3：从 JWT claims 中返回基本用户信息（临时方案）
	
	// 临时实现：返回错误提示需要实现
	h.logger.Warn("HandleMe 方法需要实现 UserService 集成")
	h.writeErrorResponse(w, errors.NewInternalError(nil))
}

// getClientIP 获取客户端 IP 地址
func (h *AuthHandler) getClientIP(r *http.Request) string {
	// 尝试从 X-Forwarded-For 头获取
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	// 尝试从 X-Real-IP 头获取
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	// 使用 RemoteAddr
	return r.RemoteAddr
}

// writeJSONResponse 写入 JSON 响应
func (h *AuthHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("写入响应失败", logger.Fields{"error": err})
	}
}

// writeErrorResponse 写入错误响应
func (h *AuthHandler) writeErrorResponse(w http.ResponseWriter, appErr *errors.AppError) {
	resp := response.Error[any](appErr.Code, appErr.Message)

	// 根据错误码确定 HTTP 状态码
	statusCode := http.StatusInternalServerError
	switch appErr.Code {
	case errors.CodeBadRequest:
		statusCode = http.StatusBadRequest
	case errors.CodeValidationError:
		statusCode = http.StatusUnprocessableEntity
	case errors.CodeNotFound:
		statusCode = http.StatusNotFound
	case errors.CodeUnauthorized:
		statusCode = http.StatusUnauthorized
	case errors.CodeForbidden:
		statusCode = http.StatusForbidden
	case errors.CodeServiceUnavailable:
		statusCode = http.StatusServiceUnavailable
	}

	h.writeJSONResponse(w, statusCode, resp)
}

// writeValidationErrorResponse 写入验证错误响应
func (h *AuthHandler) writeValidationErrorResponse(w http.ResponseWriter, validationErrors []validator.ValidationError) {
	// 构建验证错误详情
	errorData := map[string]interface{}{
		"errors": validationErrors,
	}

	resp := response.ErrorWithData(
		errors.CodeValidationError,
		errors.MsgValidationError,
		&errorData,
	)

	h.writeJSONResponse(w, http.StatusUnprocessableEntity, resp)
}

// VerifyEmailRequest 邮箱验证请求（用于 Swagger）
// @name VerifyEmailRequest
type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// ResendVerificationRequest 重新发送验证邮件请求（用于 Swagger）
// @name ResendVerificationRequest
type ResendVerificationRequest struct {
	TenantID string `json:"tenantId" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID   string `json:"userId" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// HandleVerifyEmail 处理邮箱验证
// @Summary 验证邮箱
// @Description 使用验证令牌验证用户邮箱
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body VerifyEmailRequest true "验证请求"
// @Success 200 {object} genkit-ai-service_internal_model.SuccessResponse "验证成功"
// @Failure 400 {object} genkit-ai-service_internal_model.ErrorResponse "请求参数错误"
// @Failure 422 {object} genkit-ai-service_internal_model.ErrorResponse "参数验证失败"
// @Failure 500 {object} genkit-ai-service_internal_model.ErrorResponse "服务器内部错误"
// @Router /auth/verify-email [post]
func (h *AuthHandler) HandleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 解析请求参数
	var req VerifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析邮箱验证请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 2. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("邮箱验证请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, validationErrors)
		return
	}

	// 3. 记录请求日志
	h.logger.Info("收到邮箱验证请求", logger.Fields{
		"token": req.Token[:8] + "...", // 只记录部分token
	})

	// 4. 调用服务层验证邮箱
	if err := h.emailService.VerifyEmail(ctx, req.Token); err != nil {
		h.logger.Error("邮箱验证失败", logger.Fields{
			"error": err,
			"token": req.Token[:8] + "...",
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewBadRequestError("验证令牌无效或已过期"))
		}
		return
	}

	// 5. 记录响应日志
	h.logger.Info("邮箱验证成功", logger.Fields{
		"token": req.Token[:8] + "...",
	})

	// 6. 返回成功响应
	var emptyData *interface{}
	resp := response.SuccessWithMessage("邮箱验证成功", emptyData)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// HandleResendVerification 处理重新发送验证邮件
// @Summary 重新发送验证邮件
// @Description 重新发送邮箱验证邮件
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body ResendVerificationRequest true "重发请求"
// @Success 200 {object} genkit-ai-service_internal_model.SuccessResponse "发送成功"
// @Failure 400 {object} genkit-ai-service_internal_model.ErrorResponse "请求参数错误"
// @Failure 422 {object} genkit-ai-service_internal_model.ErrorResponse "参数验证失败"
// @Failure 500 {object} genkit-ai-service_internal_model.ErrorResponse "服务器内部错误"
// @Router /auth/resend-verification [post]
func (h *AuthHandler) HandleResendVerification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 解析请求参数
	var req ResendVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析重发验证邮件请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 2. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("重发验证邮件请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, validationErrors)
		return
	}

	// 3. 记录请求日志
	h.logger.Info("收到重发验证邮件请求", logger.Fields{
		"tenantId": req.TenantID,
		"userId":   req.UserID,
	})

	// 4. 调用服务层重新发送验证邮件
	if err := h.emailService.ResendVerificationEmail(ctx, req.TenantID, req.UserID); err != nil {
		h.logger.Error("重发验证邮件失败", logger.Fields{
			"error":    err,
			"tenantId": req.TenantID,
			"userId":   req.UserID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewBadRequestError("重发验证邮件失败"))
		}
		return
	}

	// 5. 记录响应日志
	h.logger.Info("重发验证邮件成功", logger.Fields{
		"tenantId": req.TenantID,
		"userId":   req.UserID,
	})

	// 6. 返回成功响应
	var emptyData *interface{}
	resp := response.SuccessWithMessage("验证邮件已发送", emptyData)
	h.writeJSONResponse(w, http.StatusOK, resp)
}
