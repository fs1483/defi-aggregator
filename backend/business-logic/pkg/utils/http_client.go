// Package utils HTTP客户端工具
// 提供调用外部服务的HTTP客户端功能
// 支持超时控制、重试机制、错误处理等企业级特性
package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// HTTPClient HTTP客户端接口
// 定义外部服务调用的标准接口
type HTTPClient interface {
	// 基础HTTP方法
	Get(ctx context.Context, url string, headers map[string]string) ([]byte, error)
	Post(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, error)
	Put(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, error)
	Delete(ctx context.Context, url string, headers map[string]string) ([]byte, error)

	// JSON便捷方法
	GetJSON(ctx context.Context, url string, result interface{}) error
	PostJSON(ctx context.Context, url string, body interface{}, result interface{}) error
}

// DefaultHTTPClient 默认HTTP客户端实现
type DefaultHTTPClient struct {
	client  *http.Client   // 底层HTTP客户端
	timeout time.Duration  // 请求超时时间
	retries int            // 重试次数
	logger  *logrus.Logger // 日志记录器
}

// NewHTTPClient 创建HTTP客户端实例
// 配置超时、重试、连接池等参数
func NewHTTPClient(timeout time.Duration, retries int, logger *logrus.Logger) HTTPClient {
	return &DefaultHTTPClient{
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				DisableKeepAlives:   false,
			},
		},
		timeout: timeout,
		retries: retries,
		logger:  logger,
	}
}

// ========================================
// 基础HTTP方法实现
// ========================================

// Get 发送GET请求
func (c *DefaultHTTPClient) Get(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	return c.doRequest(ctx, "GET", url, nil, headers)
}

// Post 发送POST请求
func (c *DefaultHTTPClient) Post(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, error) {
	return c.doRequest(ctx, "POST", url, body, headers)
}

// Put 发送PUT请求
func (c *DefaultHTTPClient) Put(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, error) {
	return c.doRequest(ctx, "PUT", url, body, headers)
}

// Delete 发送DELETE请求
func (c *DefaultHTTPClient) Delete(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	return c.doRequest(ctx, "DELETE", url, nil, headers)
}

// ========================================
// JSON便捷方法实现
// ========================================

// GetJSON 发送GET请求并解析JSON响应
func (c *DefaultHTTPClient) GetJSON(ctx context.Context, url string, result interface{}) error {
	responseBody, err := c.Get(ctx, url, map[string]string{
		"Accept": "application/json",
	})
	if err != nil {
		return err
	}

	return json.Unmarshal(responseBody, result)
}

// PostJSON 发送POST请求并解析JSON响应
func (c *DefaultHTTPClient) PostJSON(ctx context.Context, url string, body interface{}, result interface{}) error {
	responseBody, err := c.Post(ctx, url, body, map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	})
	if err != nil {
		return err
	}

	return json.Unmarshal(responseBody, result)
}

// ========================================
// 核心请求方法
// ========================================

// doRequest 执行HTTP请求
// 包含重试逻辑、错误处理、日志记录等
func (c *DefaultHTTPClient) doRequest(ctx context.Context, method, url string, body interface{}, headers map[string]string) ([]byte, error) {
	// 准备请求体
	var requestBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		requestBody = bytes.NewReader(jsonData)
	}

	var lastErr error

	// 重试循环
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			// 重试前等待
			waitTime := time.Duration(attempt) * 100 * time.Millisecond
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(waitTime):
			}

			c.logger.Debugf("HTTP请求重试: attempt=%d, url=%s", attempt, url)

			// 重新准备请求体（Reader可能已被消费）
			if body != nil {
				jsonData, err := json.Marshal(body)
				if err != nil {
					return nil, fmt.Errorf("重试时序列化请求体失败: %w", err)
				}
				requestBody = bytes.NewReader(jsonData)
			}
		}

		// 创建HTTP请求
		req, err := http.NewRequestWithContext(ctx, method, url, requestBody)
		if err != nil {
			lastErr = fmt.Errorf("创建HTTP请求失败: %w", err)
			continue
		}

		// 设置请求头
		req.Header.Set("User-Agent", "DeFi-Aggregator-Business-Logic/1.0")
		for key, value := range headers {
			req.Header.Set(key, value)
		}

		// 发送请求
		startTime := time.Now()
		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("HTTP请求失败: %w", err)
			continue
		}

		// 读取响应体
		responseBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = fmt.Errorf("读取响应体失败: %w", err)
			continue
		}

		// 记录请求日志
		duration := time.Since(startTime)
		c.logger.Debugf("HTTP请求完成: method=%s, url=%s, status=%d, duration=%v",
			method, url, resp.StatusCode, duration)

		// 检查HTTP状态码
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			// 成功响应
			return responseBody, nil
		} else if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			// 客户端错误，不重试
			return nil, fmt.Errorf("HTTP客户端错误: status=%d, body=%s", resp.StatusCode, string(responseBody))
		} else {
			// 服务器错误，可以重试
			lastErr = fmt.Errorf("HTTP服务器错误: status=%d, body=%s", resp.StatusCode, string(responseBody))
			continue
		}
	}

	return nil, fmt.Errorf("HTTP请求最终失败: %w", lastErr)
}
