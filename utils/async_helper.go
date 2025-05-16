package utils

import (
	"bytes"
	"dify-upload-workflow/model"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AsyncRequestStore 异步请求存储，用于跟踪请求状态
var AsyncRequestStore = struct {
	sync.RWMutex
	requests map[string]model.AsyncResponse
}{
	requests: make(map[string]model.AsyncResponse),
}

// GenerateRequestID 生成唯一请求ID
func GenerateRequestID() string {
	return uuid.New().String()
}

// InitAsyncRequest 初始化异步请求并返回响应
func InitAsyncRequest(asyncReq *model.AsyncRequest) model.AsyncResponse {
	requestID := asyncReq.RequestID
	if requestID == "" {
		// 如果没有提供请求ID，则生成一个
		requestID = GenerateRequestID()
	}

	// 创建响应
	response := model.AsyncResponse{
		RequestID: requestID,
		Status:    "pending",
		Message:   "请求已接收，正在处理中",
	}

	// 保存到存储
	AsyncRequestStore.Lock()
	AsyncRequestStore.requests[requestID] = response
	AsyncRequestStore.Unlock()

	return response
}

// UpdateAsyncRequestStatus 更新异步请求状态
func UpdateAsyncRequestStatus(requestID string, status string, message string) {
	AsyncRequestStore.Lock()
	defer AsyncRequestStore.Unlock()

	if resp, exists := AsyncRequestStore.requests[requestID]; exists {
		resp.Status = status
		resp.Message = message
		AsyncRequestStore.requests[requestID] = resp
	}
}

// GetAsyncRequestStatus 获取异步请求状态
func GetAsyncRequestStatus(requestID string) (model.AsyncResponse, bool) {
	AsyncRequestStore.RLock()
	defer AsyncRequestStore.RUnlock()

	resp, exists := AsyncRequestStore.requests[requestID]
	return resp, exists
}

// CallbackResult 回调结果到指定URL
func CallbackResult(callbackURL string, requestID string, result interface{}) error {
	// 构建回调数据
	callbackData := struct {
		RequestID string      `json:"request_id"`
		Result    interface{} `json:"result"`
		Timestamp int64       `json:"timestamp"`
	}{
		RequestID: requestID,
		Result:    result,
		Timestamp: time.Now().Unix(),
	}

	// 序列化数据
	jsonData, err := json.Marshal(callbackData)
	if err != nil {
		log.Printf("序列化回调数据失败: %v", err)
		return fmt.Errorf("序列化回调数据失败: %w", err)
	}

	// 发送HTTP请求
	req, err := http.NewRequest("POST", callbackURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("创建回调请求失败: %v", err)
		return fmt.Errorf("创建回调请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", requestID)

	// 发送请求
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("发送回调请求失败: %v", err)
		return fmt.Errorf("发送回调请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("回调请求返回错误状态码: %d", resp.StatusCode)
		return fmt.Errorf("回调请求返回错误状态码: %d", resp.StatusCode)
	}

	return nil
}
