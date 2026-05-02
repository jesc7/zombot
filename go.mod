module github.com/jesc7/zombot

go 1.25.7

require (
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/jesc7/zombot/cmd/zspy v0.0.0-20260428070659-40a8823595c4
	github.com/kardianos/service v1.2.4
	github.com/max-messenger/max-bot-api-client-go v1.6.14
	github.com/mymmrac/telego v1.8.0
	golang.org/x/sys v0.43.0
	golang.org/x/text v0.36.0
	golang.org/x/time v0.15.0
)

require (
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/bytedance/gopkg v0.1.3 // indirect
	github.com/bytedance/sonic v1.15.0 // indirect
	github.com/bytedance/sonic/loader v0.5.0 // indirect
	github.com/cloudwego/base64x v0.1.6 // indirect
	github.com/grbit/go-json v0.11.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.2 // indirect
	github.com/klauspost/cpuid/v2 v2.2.9 // indirect
	github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421 // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.69.0 // indirect
	github.com/valyala/fastjson v1.6.10 // indirect
	golang.org/x/arch v0.0.0-20210923205945-b76863e36670 // indirect
)

replace github.com/jesc7/zombot/cmd/zspy => ./cmd/zspy
