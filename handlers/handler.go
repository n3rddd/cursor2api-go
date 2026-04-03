// Copyright (c) 2025-2026 libaxuan
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package handlers

import (
	"cursor2api-go/config"
	"cursor2api-go/middleware"
	"cursor2api-go/models"
	"cursor2api-go/services"
	"cursor2api-go/utils"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Handler 处理器结构
type Handler struct {
	cfg           *config.Config
	cursorService *services.CursorService
	docsContent   []byte
	docsMimeType  string
}

// NewHandler 创建新的处理器
func NewHandler(cfg *config.Config) *Handler {
	cursorService := services.NewCursorService(cfg)

	// 预加载文档内容
	docsContent, docsMimeType := loadDocs("static/docs.html")

	return &Handler{
		cfg:           cfg,
		cursorService: cursorService,
		docsContent:   docsContent,
		docsMimeType:  docsMimeType,
	}
}

// loadDocs 预加载文档内容
func loadDocs(path string) ([]byte, string) {
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		return data, "text/html; charset=utf-8"
	}

	// 如果文件不存在，使用默认的简单HTML页面
	return []byte(defaultDocs()), "text/html; charset=utf-8"
}

func defaultDocs() string {
	return `<!DOCTYPE html>
<html lang="zh-CN"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Cursor2API</title>
<style>body{font-family:'PingFang SC','Microsoft YaHei','Segoe UI',system-ui,sans-serif;max-width:800px;margin:50px auto;padding:20px;background:#f5f5f5}
.container{background:#fff;padding:30px;border-radius:10px;box-shadow:0 2px 10px rgba(0,0,0,.1)}
h1{color:#333;border-bottom:2px solid #007bff;padding-bottom:10px}
.info{background:#f8f9fa;padding:20px;border-radius:8px;margin:20px 0;border-left:4px solid #007bff}
code{background:#e9ecef;padding:2px 6px;border-radius:4px;font-family:'Courier New',monospace}
.endpoint{background:#e3f2fd;padding:10px;margin:10px 0;border-radius:5px;border-left:3px solid #2196f3}
.ok{color:#28a745;font-weight:700}</style></head>
<body><div class="container">
<h1>🚀 Cursor2API — Go 版本</h1>
<div class="info"><p>状态：<span class="ok">✅ 运行中</span></p>
<p>OpenAI 兼容的 AI 聊天代理网关（Cursor AI 桥接）</p></div>
<div class="info"><h3>📡 可用接口</h3>
<div class="endpoint"><code>GET /v1/models</code> — 列出可用模型</div>
<div class="endpoint"><code>POST /v1/chat/completions</code> — 聊天补全（支持流式 + 非流式 + 工具调用）</div>
<div class="endpoint"><code>GET /health</code> — 健康检查</div></div>
<div class="info"><h3>🔐 认证方式</h3>
<p>Bearer Token 认证：<code>Authorization: Bearer YOUR_API_KEY</code></p>
<p>默认密钥：<code>0000</code>（建议通过 <code>API_KEY</code> 环境变量修改）</p></div>
<div class="info"><h3>💻 使用示例</h3>
<pre><code>curl -X POST http://localhost:8002/v1/chat/completions \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer 0000" \\
  -d '{"model":"gemini-3-flash","messages":[{"role":"user","content":"你好"}],"stream":true}'</code></pre>
</div>
</div></body></html>`
}

// ListModels 列出可用模型
func (h *Handler) ListModels(c *gin.Context) {
	modelNames := h.cfg.GetModels()
	modelList := make([]models.Model, 0, len(modelNames))

	for _, modelID := range modelNames {
		m := models.Model{
			ID:      modelID,
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "cursor2api",
		}
		if mc, ok := models.GetModelConfig(modelID); ok {
			m.MaxTokens = mc.MaxTokens
			m.ContextWindow = mc.ContextWindow
		}
		modelList = append(modelList, m)
	}

	c.JSON(http.StatusOK, models.ModelsResponse{Object: "list", Data: modelList})
}

// ChatCompletions 处理聊天完成请求
func (h *Handler) ChatCompletions(c *gin.Context) {
	var req models.ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Debug("Failed to bind request")
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid request body: must be valid JSON",
			"invalid_request_error",
			"invalid_json",
		))
		return
	}

	// Validate model
	if req.Model == "" {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Missing required field: model",
			"invalid_request_error",
			"missing_model",
		))
		return
	}
	if !h.cfg.IsValidModel(req.Model) {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid model: "+req.Model,
			"invalid_request_error",
			"model_not_found",
		))
		return
	}

	// Validate messages
	if len(req.Messages) == 0 {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Messages cannot be empty",
			"invalid_request_error",
			"missing_messages",
		))
		return
	}

	// Validate and cap max_tokens
	req.MaxTokens = models.ValidateMaxTokens(req.Model, req.MaxTokens)

	logger := logrus.WithField("model", req.Model).WithField("stream", req.Stream)
	logger.Debug("Processing chat completion")

	if req.Stream {
		gen, err := h.cursorService.ChatCompletion(c.Request.Context(), &req)
		if err != nil {
			logger.WithError(err).Error("Chat completion stream error")
			middleware.HandleError(c, err)
			return
		}
		utils.SafeStreamWrapper(utils.StreamChatCompletion, c, gen, req.Model)
	} else {
		resp, err := h.cursorService.ChatCompletionNonStream(c.Request.Context(), &req)
		if err != nil {
			logger.WithError(err).Error("Non-stream chat completion error")
			middleware.HandleError(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

// ServeDocs 服务API文档页面
func (h *Handler) ServeDocs(c *gin.Context) {
	c.Data(http.StatusOK, h.docsMimeType, h.docsContent)
}

// Health 健康检查
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
	})
}
