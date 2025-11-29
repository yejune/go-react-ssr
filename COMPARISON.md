# Next.js + Go vs go-react-ssr 비교 분석

## 1. 아키텍처 비교

### Option A: Next.js + Go (전통적 마이크로서비스)

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Client    │────▶│   Next.js   │────▶│  Go API     │
│   Browser   │◀────│   (SSR)     │◀────│  Server     │
└─────────────┘     └─────────────┘     └─────────────┘
                    Port: 3000          Port: 8080

요청 흐름:
1. 브라우저 → Next.js (SSR 요청)
2. Next.js → Go API (데이터 요청)
3. Go API → Next.js (데이터 응답)
4. Next.js → 브라우저 (렌더링된 HTML)

총 네트워크 홉: 4
```

### Option B: go-react-ssr (단일 서버)

```
┌─────────────┐     ┌─────────────────────────────┐
│   Client    │────▶│         Go Server           │
│   Browser   │◀────│  ┌─────────┐  ┌──────────┐  │
└─────────────┘     │  │ go-ssr  │  │ Business │  │
                    │  │ Engine  │  │  Logic   │  │
                    │  └─────────┘  └──────────┘  │
                    └─────────────────────────────┘
                    Port: 8080 (단일)

요청 흐름:
1. 브라우저 → Go 서버 (요청)
2. Go 서버 내부: 데이터 조회 → SSR → 응답
3. Go 서버 → 브라우저 (렌더링된 HTML)

총 네트워크 홉: 2
```

## 2. 성능 비교

### 단일 요청 레이턴시

| 단계 | Next.js + Go | go-react-ssr |
|------|-------------|--------------|
| 네트워크 (브라우저→서버) | 10ms | 10ms |
| 데이터 조회 | 5ms | 5ms |
| 내부 API 호출 | 5-10ms | 0ms (불필요) |
| SSR 렌더링 | 5-20ms | 10-30ms |
| 네트워크 (서버→브라우저) | 10ms | 10ms |
| **총 레이턴시** | **35-55ms** | **35-55ms** |

➡️ **결론: 비슷함** (go-react-ssr의 느린 SSR이 네트워크 홉 제거로 상쇄)

### 처리량 (Throughput)

| 메트릭 | Next.js + Go | go-react-ssr (V8) |
|--------|-------------|-------------------|
| 단일 서버 req/s | 500-1000 | 200-500 |
| 수평 확장 시 | 선형 증가 | 선형 증가 |
| 메모리 효율 | 낮음 | 높음 |

### 메모리 사용량

| 구성 | Next.js + Go | go-react-ssr |
|------|-------------|--------------|
| 유휴 상태 | 500MB + 50MB = 550MB | 100MB |
| 1000 동시 연결 | 800MB + 100MB = 900MB | 200MB |
| 스케일 팩터 | 2x 서버 필요 | 1x 서버 |

## 3. 운영 비교

### 배포 복잡도

**Next.js + Go:**
```yaml
# docker-compose.yml
services:
  nextjs:
    build: ./frontend
    ports: ["3000:3000"]
    environment:
      - API_URL=http://go-api:8080
    depends_on:
      - go-api

  go-api:
    build: ./backend
    ports: ["8080:8080"]
```
- 2개 컨테이너 관리
- 서비스 간 네트워크 설정
- 헬스체크 2개
- 로그 수집 2곳

**go-react-ssr:**
```yaml
services:
  app:
    build: .
    ports: ["8080:8080"]
```
- 1개 컨테이너
- 단순한 배포
- 단일 로그 스트림

### 월간 인프라 비용 (AWS 기준)

| 항목 | Next.js + Go | go-react-ssr |
|------|-------------|--------------|
| EC2 (Next.js) | $30-50 (t3.medium) | - |
| EC2 (Go) | $20-30 (t3.small) | $20-30 (t3.small) |
| 로드밸런서 | $20 x 2 = $40 | $20 |
| **월 비용** | **$90-120** | **$40-50** |

**연간 절감액: $600-840**

## 4. 개발 경험 비교

### Next.js + Go

**장점:**
- Next.js의 풍부한 기능 (ISR, SSG, Image 최적화)
- 프론트엔드/백엔드 팀 분리 가능
- 대규모 React 생태계

**단점:**
- 타입 동기화 어려움 (Go ↔ TypeScript)
- API 스키마 관리 필요 (OpenAPI, gRPC 등)
- 두 언어/런타임 전문성 필요

### go-react-ssr

**장점:**
- 타입 자동 생성 (Go struct → TypeScript)
- 단일 코드베이스
- Go만 알면 됨 (React 기본 지식 + Go)

**단점:**
- Next.js 고급 기능 없음
- React 생태계 일부 제한
- 커뮤니티 작음

## 5. 프로덕션 체크리스트

### go-react-ssr 프로덕션 준비도

| 항목 | 상태 | 비고 |
|------|------|------|
| SSR 렌더링 | ✅ | 안정적 |
| 타입 안전성 | ✅ | Go→TS 자동 생성 |
| Hot Reload | ✅ | 개발 모드 |
| 런타임 풀링 | ✅ | V8/QuickJS 풀 |
| 캐싱 | ⚠️ | 인메모리만 (재시작 시 소멸) |
| 에러 핸들링 | ⚠️ | 기본 수준 |
| 메트릭 | ❌ | 직접 구현 필요 |
| 분산 캐시 | ❌ | 미지원 |
| 스트리밍 SSR | ❌ | 미지원 |
| graceful shutdown | ❌ | 직접 구현 필요 |

### 프로덕션 투입 전 권장 작업

```go
// 1. graceful shutdown 추가
func main() {
    engine, _ := gossr.New(config)

    srv := &http.Server{Addr: ":8080"}

    go func() {
        sigint := make(chan os.Signal, 1)
        signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
        <-sigint

        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        srv.Shutdown(ctx)
    }()

    srv.ListenAndServe()
}

// 2. 에러 복구 미들웨어
func recoveryMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                log.Printf("panic recovered: %v", err)
                http.Error(w, "Internal Server Error", 500)
            }
        }()
        next.ServeHTTP(w, r)
    })
}

// 3. 메트릭 추가 (Prometheus)
var (
    renderDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{Name: "ssr_render_duration_seconds"},
        []string{"route"},
    )
)
```

## 6. 결정 가이드

### go-react-ssr 선택

✅ **적합한 경우:**
- Go 백엔드가 이미 있고 SSR 추가 필요
- 내부 도구 / 관리자 대시보드
- B2B 애플리케이션
- 비용 최적화 우선
- 팀이 Go에 익숙
- 트래픽 일 10만 요청 이하

### Next.js + Go 선택

✅ **적합한 경우:**
- 고트래픽 소비자 서비스 (일 100만+ 요청)
- SEO 최적화 필수 (ISR/SSG 필요)
- 복잡한 React 앱 (많은 인터랙션)
- 프론트엔드/백엔드 팀 분리
- React 생태계 최대 활용

## 7. 마이그레이션 전략

### Next.js + Go → go-react-ssr

```
단계 1: 단순 페이지부터 마이그레이션
  - 정적 콘텐츠 위주 페이지
  - 데이터 의존성 적은 페이지

단계 2: API 라우트 통합
  - Next.js API 라우트 → Go 핸들러
  - 내부 API 호출 제거

단계 3: 복잡한 페이지
  - 점진적 마이그레이션
  - A/B 테스트로 성능 비교

단계 4: Next.js 서버 종료
  - 완전 마이그레이션 후
```

### go-react-ssr → Next.js + Go (롤백)

```
단계 1: Next.js 프로젝트 생성
단계 2: React 컴포넌트 복사 (그대로 사용 가능)
단계 3: Go API 엔드포인트 추가
단계 4: getServerSideProps로 데이터 페칭 추가
```

## 8. 최종 권장사항

### 현재 go-react-ssr 상태

**프로덕션 투입: 조건부 가능**

| 조건 | 판단 |
|------|------|
| 트래픽 < 10만/일 | ✅ 가능 |
| 복잡도 낮은 UI | ✅ 가능 |
| Go 팀 | ✅ 가능 |
| 비용 최적화 필요 | ✅ 강력 추천 |
| 트래픽 > 100만/일 | ❌ Next.js 권장 |
| 복잡한 React 앱 | ❌ Next.js 권장 |
| SSG/ISR 필요 | ❌ Next.js 권장 |

### 추천 시나리오

1. **내부 도구**: go-react-ssr ✅
2. **B2B SaaS 대시보드**: go-react-ssr ✅
3. **소규모 마케팅 사이트**: go-react-ssr ✅
4. **대규모 이커머스**: Next.js + Go ✅
5. **미디어/콘텐츠 사이트**: Next.js + Go ✅
