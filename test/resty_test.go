package test

import (
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"

	"golang.org/x/net/proxy"
	"resty.dev/v3"
)

func TestResty(t *testing.T) {

	// SOCKS5 代理地址（例如本地 v2ray、shadowsocks、trojan 等）
	proxyAddr := "127.0.0.1:1080"

	// 如果代理需要用户名密码认证（可选）
	// auth := &proxy.Auth{User: "your_username", Password: "your_password"}
	// auth := (*proxy.Auth)(nil) // 无认证设为 nil

	// 创建 SOCKS5 dialer
	// 使用 proxy.Direct 作为 forward dialer → 实现真正的 socks5h（域名解析在代理端进行）
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
	if err != nil {
		log.Fatalf("创建 SOCKS5 dialer 失败: %v", err)
	}

	// 创建自定义 Transport
	transport := &http.Transport{
		Dial:                dialer.Dial, // 关键：使用 SOCKS5 dialer
		TLSHandshakeTimeout: 10 * time.Second,
		// 其他可选优化
		DisableKeepAlives: false,
		MaxIdleConns:      100,
		IdleConnTimeout:   90 * time.Second,
	}

	// 创建底层 http.Client
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second, // 整体请求超时
	}

	// 使用 resty v3 创建客户端，并传入自定义 http.Client
	client := resty.NewWithClient(httpClient)

	// 可选：设置重试、调试等
	client.
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second).
		SetRetryMaxWaitTime(10 * time.Second).
		// 如果需要看详细请求日志，可以打开调试
		SetDebug(true)

	// 示例请求：检查代理是否生效
	resp, err := client.R().
		Get("https://gamma-api.polymarket.com")

	if err != nil {
		log.Fatalf("请求失败: %v", err)
	}

	fmt.Println("状态码:", resp.Status())
	fmt.Println("代理出口 IP:", resp.String())
}
