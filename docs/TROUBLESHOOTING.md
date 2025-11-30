# 트러블슈팅

## 1초 주기 Latency 스파이크
**증상**: 99% latency가 1-2초로 치솟음
**원인**: Pool에서 동적 Isolate 생성
**해결**: [V8_GC_OPTIMIZATION.md](V8_GC_OPTIMIZATION.md) 참고

## SSR 결과가 빈 HTML
**증상**: `<div id="root"></div>`가 비어있음
**확인**:
1. HTML 소스에서 `<!-- SSR_ERRORS: ... -->` 주석 확인
2. 서버 로그에서 에러 확인
3. `RequestPath` 설정 확인:
```go
engine.RenderRoute(gossr.RenderConfig{
    RequestPath: "/",  // 필수
})
```

## 메모리 누수
**증상**: 메모리 사용량 계속 증가
**확인**:
```bash
# 메모리 프로파일
curl http://localhost:6060/debug/pprof/heap -o heap.pprof
go tool pprof -http=:8081 heap.pprof
```

## 빌드 에러: v8go
```bash
go clean -cache
go mod tidy
CGO_ENABLED=1 go build
```

## Hot Reload 안됨
```bash
APP_ENV=development ./server  # production이면 안됨
```

## 디버깅 체크리스트
- [ ] `APP_ENV` 환경변수
- [ ] HTML 소스에서 `SSR_ERRORS` 주석
- [ ] `hey -n 100 -c 10` 벤치마크
- [ ] `GODEBUG=gctrace=1` Go GC 로그
