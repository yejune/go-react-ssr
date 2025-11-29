# JavaScript 런타임 선택 가이드

> 2025-11-30 | gotossr V8 기본 런타임 전환 기록

---

## TL;DR

**V8이 기본 런타임. 2.4배 빠름.**

QuickJS 쓰려면: `go build -tags=use_quickjs`

---

## 벤치마크 결과

| 항목 | V8 (기본) | QuickJS |
|------|----------|---------|
| **SSR 지연시간** | **53ms** | 128ms |
| 바이너리 크기 | 64MB | 32MB |
| 메모리 (RSS) | 117MB | 104MB |
| 빌드 시간 | 느림 (C++ 컴파일) | 빠름 |
| URL API | 내장 | polyfill 필요 |
| React Router | 네이티브 지원 | polyfill로 지원 |

### 테스트 환경
- macOS Darwin 25.0.0
- Go 1.24
- React 18 + React Router 6
- 10회 요청 평균

### 테스트 명령어
```bash
# V8 빌드 (기본)
go build -o /tmp/app .

# QuickJS 빌드
go build -tags=use_quickjs -o /tmp/app_qjs .

# 벤치마크
for i in {1..10}; do
  curl -s -o /dev/null -w "%{time_total}\n" http://localhost:8080/
done | awk '{sum+=$1} END {print "Average:", sum/NR, "seconds"}'
```

---

## 런타임 선택 기준

### V8 선택 (기본, 권장)
- 성능이 중요한 프로덕션 환경
- React Router 사용하는 SPA
- Web API를 많이 사용하는 코드
- SSR 응답 시간이 중요한 경우

### QuickJS 선택
- 바이너리 크기가 중요한 경우 (32MB vs 64MB)
- 메모리가 제한된 환경
- CGO 없이 빌드해야 하는 경우 (QuickJS는 pure Go 바인딩)
- 빌드 시간이 중요한 CI/CD 환경

---

## 빌드 방법

### V8 (기본)
```bash
# 일반 빌드
go build -o myapp .

# Wails 제외
go build -tags='!wails' -o myapp .
```

### QuickJS
```bash
# QuickJS로 빌드
go build -tags=use_quickjs -o myapp .

# Wails 제외 + QuickJS
go build -tags='!wails,use_quickjs' -o myapp .
```

---

## 런타임 확인

서버 시작 로그에서 확인:
```
level=DEBUG msg="Initialized JS runtime pool" runtime=v8 pool_size=10
```
또는
```
level=DEBUG msg="Initialized JS runtime pool" runtime=quickjs pool_size=10
```

---

## V8 빌드 시 주의사항

### macOS
V8 빌드 시 C++ 경고가 발생할 수 있음 (무시해도 됨):
```
v8go.cc:502:24: warning: variable length arrays in C++ are a Clang extension
```

### Linux
V8 의존성 설치 필요할 수 있음:
```bash
# Ubuntu/Debian
sudo apt-get install build-essential

# Alpine (Docker)
apk add --no-cache build-base
```

### Cross-compilation
V8는 크로스 컴파일이 어려움. 타겟 플랫폼에서 직접 빌드 권장.

---

## QuickJS 제한사항

QuickJS에서는 다음 Web API가 없어서 polyfill 필요:

| API | 상태 |
|-----|------|
| `URL` | ✅ polyfill 추가됨 (React Router 지원) |
| `TextEncoder/Decoder` | ✅ polyfill 있음 |
| `URLSearchParams` | ❌ 필요시 추가 |
| `fetch` | ❌ SSR에서 안 씀 |
| `AbortController` | ❌ |
| `Blob`, `FormData` | ❌ |

---

## 트러블슈팅

### V8 빌드 실패
```
# error: undefined reference to `v8::...`
```
→ CGO가 활성화되어 있는지 확인: `CGO_ENABLED=1`

### QuickJS에서 React Router 빈 결과
→ URL polyfill이 적용되었는지 확인 (gotossr v0.0.0-20251130 이후 포함)

상세 디버깅은 [TROUBLESHOOTING-QUICKJS-URL.md](./TROUBLESHOOTING-QUICKJS-URL.md) 참고

---

## 성능 튜닝

### 런타임 풀 크기
```go
gossr.Init(gossr.Config{
    JSRuntimePoolSize: 20,  // 기본값: 10
})
```

- CPU 코어 수 * 2 정도 권장
- 동시 요청이 많으면 늘림
- 메모리가 부족하면 줄임

### 캐싱
SSR 결과는 자동 캐싱되지 않음. 필요시 Redis 등 외부 캐시 사용.

---

## 커밋 히스토리

```
159e08c - Switch default JS runtime to V8 for better performance
ba01063 - Add URL polyfill and SPA CSS caching for QuickJS SSR
```

---

## 관련 파일

- `internal/jsruntime/v8.go` - V8 런타임 구현
- `internal/jsruntime/quickjs.go` - QuickJS 런타임 구현
- `internal/jsruntime/runtime.go` - 런타임 풀 관리
- `internal/reactbuilder/build.go` - polyfill 정의
