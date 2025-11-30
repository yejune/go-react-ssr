# gotossr

Go로 React SSR을 간단하게. **"go to SSR"** — 그냥 가면 됩니다.

---

[![Go Report](https://goreportcard.com/badge/github.com/yejune/gotossr)](https://goreportcard.com/report/github.com/yejune/gotossr)
[![GoDoc](http://img.shields.io/badge/GoDoc-Reference-blue.svg)](https://pkg.go.dev/github.com/yejune/gotossr?tab=doc)
[![MIT License](https://img.shields.io/badge/License-MIT%202.0-blue.svg)](https://github.com/yejune/gotossr/blob/master/LICENSE)

gotossr은 **기존 Go 웹 프레임워크**에 드롭인 방식으로 추가하여 [React](https://react.dev/)를 **서버 사이드 렌더링**할 수 있게 해주는 플러그인입니다. [esbuild](https://esbuild.github.io/)로 구동되며 Go에서 React로 **타입 안전한** props 전달을 지원합니다.

gotossr은 기존 Go 생태계에서 풀스택 React 앱을 쉽게 구축할 수 있는 도구가 부족하여 개발되었습니다. [Remix](https://remix.run/)와 [Next.JS](https://nextjs.org/)에서 영감을 받았지만, 프레임워크가 아닌 플러그인을 목표로 합니다.

# 주요 기능

- [esbuild](https://esbuild.github.io/)를 활용한 초고속 컴파일
- **V8 JavaScript 엔진 지원**으로 고성능 SSR
- **런타임 풀링**으로 최적의 리소스 사용
- Props를 위한 TypeScript 타입 자동 생성
- 개발 모드에서 Hot reloading
- 간단한 에러 리포팅
- Build tag를 활용한 프로덕션 최적화
- 기존 Go 웹 서버에 드롭인 방식 적용
- 최소 의존성 (프로덕션 모드에서 2개)

# 시작하기

gotossr은 설치가 매우 간단하도록 설계되었습니다.

## CLI 도구 사용

가장 쉽게 프로젝트를 시작하는 방법입니다.

```console
$ go install github.com/yejune/gotossr/gossr-cli@latest
$ gossr-cli create
```

프로젝트 경로, 웹 프레임워크, Tailwind 사용 여부를 선택할 수 있습니다.

## 기존 웹 서버에 추가

```console
$ go get -u github.com/yejune/gotossr
```

메인 파일에 import 추가:

```go
import (
    gossr "github.com/yejune/gotossr"
)
```

엔진 초기화:

```go
engine, err := gossr.New(gossr.Config{
    AppEnv:             "development", // 또는 "production"
    AssetRoute:         "/assets",
    FrontendDir:        "./frontend/src",
    GeneratedTypesPath: "./frontend/src/generated.d.ts",
    PropsStructsPath:   "./models/props.go",
})
```

### 설정 옵션

| 옵션 | 타입 | 기본값 | 설명 |
|------|------|--------|------|
| `AppEnv` | string | `"development"` | `"development"` 또는 `"production"` |
| `AssetRoute` | string | - | 에셋 제공 경로 (예: `"/assets"`) |
| `FrontendDir` | string | - | React 소스 디렉토리 경로 |
| `GeneratedTypesPath` | string | - | TypeScript 타입 생성 경로 |
| `PropsStructsPath` | string | - | Go props 구조체 파일 경로 |
| `LayoutFilePath` | string | - | 레이아웃 파일 경로 (선택) |
| `LayoutCSSFilePath` | string | - | 전역 CSS 파일 경로 (선택) |
| `TailwindConfigPath` | string | - | Tailwind 설정 경로 (선택) |
| `HotReloadServerPort` | int | `3001` | Hot reload WebSocket 포트 |
| `JSRuntimePoolSize` | int | `10` | JS 런타임 풀 크기 |

라우트 렌더링:

```go
g.GET("/", func(c *gin.Context) {
    renderedResponse := engine.RenderRoute(gossr.RenderConfig{
        File:  "Home.tsx",
        Title: "예제 앱",
        MetaTags: map[string]string{
            "og:title":    "예제 앱",
            "description": "Hello world!",
        },
        Props: &models.IndexRouteProps{
            InitialCount: rand.Intn(100),
        },
    })
    c.Writer.Write(renderedResponse)
})
```

# 성능

gotossr은 두 가지 JavaScript 런타임을 지원합니다:

| 런타임 | Build Tag | 성능 | 사용 케이스 |
|--------|-----------|------|------------|
| QuickJS | (기본값) | 양호 | 개발, 저메모리 환경 |
| V8 | `-tags=use_v8` | **70-85% 빠름** | 프로덕션, 고트래픽 |

### 벤치마크 (Apple M4)

```
                          QuickJS      V8         개선율
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
단순 렌더링               225μs       199μs      +12%
복잡한 렌더링             234μs       137μs      +70%
병렬 처리 (10 cores)      49μs        26μs       +85%
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

# Build Tags

gotossr은 build tag를 사용하여 프로덕션에서 의존성을 최소화합니다:

| 빌드 명령 | 런타임 | Dev 기능 | 의존성 |
|-----------|--------|----------|--------|
| `go build` | QuickJS | Hot reload, 타입생성 | 5개 |
| `go build -tags=use_v8` | V8 | Hot reload, 타입생성 | 5개 |
| `go build -tags=prod` | QuickJS | 없음 | **2개** |
| `go build -tags="prod,use_v8"` | V8 | 없음 | **2개** |

### 권장 프로덕션 빌드

```bash
# V8을 사용한 최고 성능 프로덕션 빌드
go build -tags="prod,use_v8" -ldflags "-w -s" -o main .
```

# 프로덕션 배포

### V8 사용 Dockerfile (최고 성능)

```Dockerfile
# V8 런타임으로 백엔드 빌드
FROM golang:1.24-alpine as build-backend
RUN apk add --no-cache git build-base
ADD . /build
WORKDIR /build

RUN go mod download
RUN CGO_ENABLED=1 GOOS=linux go build -tags="prod,use_v8" -ldflags "-w -s" -o main .

# 프론트엔드 빌드
FROM node:20-alpine as build-frontend
ADD ./frontend /frontend
WORKDIR /frontend
RUN npm install

# 최종 이미지
FROM alpine:latest
RUN apk add --no-cache libstdc++ libgcc
COPY --from=build-backend /build/main ./app/main
COPY --from=build-frontend /frontend ./app/frontend

WORKDIR /app
RUN chmod +x ./main
EXPOSE 8080
CMD ["./main"]
```

### 경량 빌드 (QuickJS)

더 작은 이미지를 원하는 경우:

```Dockerfile
FROM golang:1.24-alpine as build-backend
ADD . /build
WORKDIR /build
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -tags=prod -ldflags "-w -s" -o main .

FROM node:20-alpine as build-frontend
ADD ./frontend /frontend
WORKDIR /frontend
RUN npm install

FROM alpine:latest
COPY --from=build-backend /build/main ./app/main
COPY --from=build-frontend /frontend ./app/frontend
WORKDIR /app
EXPOSE 8080
CMD ["./main"]
```

### 배포 테스트 완료 플랫폼

- [Fly.io](https://fly.io/)
- [Render](https://render.com/)
- [Hop.io](https://hop.io/)

# 아키텍처

```
┌─────────────────────────────────────────────────────────────────┐
│                        HTTP 요청                                 │
└─────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Go 웹 프레임워크                              │
│              (Fiber / Gin / Echo / net/http)                     │
└─────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                      gotossr 엔진                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   esbuild   │  │  JS 런타임  │  │       캐시 매니저        │  │
│  │  (번들러)   │  │  풀 (V8/    │  │  (인메모리, 라우트별)    │  │
│  │             │  │   QuickJS)  │  │                         │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                     렌더링된 HTML 응답                           │
└─────────────────────────────────────────────────────────────────┘
```

### 요청 흐름

1. **첫 요청 (캐시 미스)**
   - esbuild가 React 컴포넌트 번들링 (~100-500ms)
   - JS 런타임 풀이 `renderToString()` 실행 (~10-50ms)
   - 결과가 메모리에 캐시됨

2. **이후 요청 (캐시 히트)**
   - Props가 캐시된 JS 번들에 주입됨
   - JS 런타임이 즉시 실행 (~10-25ms)
   - 번들링 오버헤드 없음

# Next.js + Go vs gotossr 비교

### 아키텍처 비교

**Next.js + Go (전통적):**
```
브라우저 ──▶ Next.js (SSR) ──▶ Go API ──▶ DB
            Port 3000         Port 8080
            ~500MB RAM        ~50MB RAM
```
- 서버 2개 관리 필요
- 요청당 네트워크 홉 4회
- 높은 인프라 비용

**gotossr (단일 서버):**
```
브라우저 ──▶ Go 서버 (SSR + API) ──▶ DB
            Port 8080
            ~100-200MB RAM
```
- 서버 1개만 관리
- 요청당 네트워크 홉 2회
- 낮은 인프라 비용

### 성능 비교

| 메트릭 | Next.js + Go | gotossr (V8) |
|--------|-------------|-------------------|
| SSR 레이턴시 | 5-20ms | 10-30ms |
| 내부 API 호출 | 5-10ms | 0ms (불필요) |
| **총 레이턴시** | **35-55ms** | **35-55ms** |
| 메모리 (유휴) | 550MB | 100MB |
| 메모리 (1000 연결) | 900MB | 200MB |
| 처리량 | 500-1000 req/s | 200-500 req/s |

### 월간 인프라 비용 (AWS 기준)

| 항목 | Next.js + Go | gotossr |
|------|-------------|--------------|
| EC2 인스턴스 | $50-80 (2대) | $20-30 (1대) |
| 로드밸런서 | $40 (2개) | $20 (1개) |
| **월 합계** | **$90-120** | **$40-50** |

**연간 절감액: $600-840**

### 기능 비교

| 기능 | gotossr | Next.js |
|------|--------------|---------|
| SSR | ✅ | ✅ |
| SSG/ISR | ❌ | ✅ |
| 스트리밍 SSR | ❌ | ✅ |
| 이미지 최적화 | ❌ | ✅ |
| 타입 안전성 | ✅ 자동생성 | 수동 |
| 단일 배포 | ✅ | ❌ |
| 메모리 효율 | ✅ | ❌ |

### 사용 권장

**gotossr 권장:**
- SSR이 필요한 기존 Go 백엔드
- 내부 도구 / 관리자 대시보드
- B2B 애플리케이션
- 비용 최적화 우선
- 트래픽 일 10만 요청 이하
- Go 전문 팀

**Next.js + Go 권장:**
- 고트래픽 소비자 앱 (일 10만+ 요청)
- 복잡한 인터랙션의 React 앱
- Node.js 전문 팀

### 프로덕션 준비도

| 기능 | 상태 |
|------|------|
| SSR 렌더링 | ✅ 안정적 |
| 타입 안전성 | ✅ 자동 생성 |
| 런타임 풀링 | ✅ V8/QuickJS |
| Graceful Shutdown | ✅ 내장 |
| 캐싱 | ✅ 로컬 (기본) / Redis (선택) |

**결론: 저~중간 트래픽 애플리케이션에 프로덕션 사용 가능**

### 캐싱 옵션

gotossr은 esbuild 번들 결과를 캐시하여 매 요청마다 재번들링을 방지합니다.

| 캐시 타입 | 사용 케이스 | 설정 |
|----------|------------|------|
| 로컬 (기본) | 단일 서버, 간단한 구성 | 설정 불필요 |
| Redis | 다중 서버, 캐시 공유 | `RedisAddr` 설정 |

**로컬 캐시 (기본):**
- 각 서버가 독립적으로 캐시
- 서버당 첫 요청: ~200ms (번들링)
- 이후 요청: ~0ms (캐시 히트)
- 대부분의 경우 충분함

**Redis 캐시 (선택):**
- 모든 서버가 하나의 캐시 공유
- 전체에서 첫 요청만: ~200ms
- 나머지 모든 요청: ~1ms (Redis 조회)
- 서버가 많을 때 콜드 스타트 단축에 유용

```go
// Redis 캐시 설정 (선택)
engine, _ := gossr.New(gossr.Config{
    // ... 기타 설정
    CacheType:     "redis",              // "local" (기본) 또는 "redis"
    RedisAddr:     "localhost:6379",
    RedisPassword: "",                   // 선택
    RedisDB:       0,                    // 선택
    RedisTLS:      true,                 // 선택, TLS 연결용
})
```

### Graceful Shutdown 예제

```go
package main

import (
    "context"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    gossr "github.com/yejune/gotossr"
)

func main() {
    engine, _ := gossr.New(gossr.Config{
        AppEnv:      "production",
        FrontendDir: "./frontend/src",
    })

    srv := &http.Server{Addr: ":8080"}

    // 라우트 핸들러
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        response := engine.RenderRoute(gossr.RenderConfig{
            File: "Home.tsx",
        })
        w.Write(response)
    })

    // 서버 시작
    go func() {
        srv.ListenAndServe()
    }()

    // 종료 시그널 대기
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
    <-quit

    // 10초 타임아웃으로 graceful shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    srv.Shutdown(ctx)
    engine.Shutdown(ctx)  // 런타임 풀 정리, 캐시 클리어
}
```

# 마이그레이션 가이드: gotossr → Next.js

트래픽이 일 10만 요청을 초과하거나 SSG, ISR, 스트리밍 SSR 등 고급 기능이 필요할 때 Next.js로 마이그레이션하세요.

### 1단계: Next.js 프로젝트 생성

```bash
# 기존 Go 프로젝트 옆에 Next.js 앱 생성
npx create-next-app@latest frontend-nextjs --typescript --app

# 새로운 구조:
your-project/
├── go-server/           # 기존 Go 백엔드 (API로 유지)
│   ├── main.go
│   └── handlers/
├── frontend-nextjs/     # 새 Next.js 프론트엔드
│   ├── app/
│   └── package.json
└── frontend/            # 기존 gotossr 프론트엔드 (마이그레이션 대상)
```

### 2단계: React 컴포넌트 복사

```bash
# React 컴포넌트를 그대로 복사 (수정 불필요!)
cp -r frontend/src/components frontend-nextjs/components/
cp -r frontend/src/*.tsx frontend-nextjs/app/

# 컴포넌트는 호환됨 - 변경 없이 사용 가능
```

### 3단계: Props를 데이터 페칭으로 변환

**변경 전 (gotossr):**
```go
// Go 핸들러
func HomeHandler(c *gin.Context) {
    data := getDataFromDB()
    response := engine.RenderRoute(gossr.RenderConfig{
        File:  "Home.tsx",
        Props: &HomeProps{Data: data},
    })
    c.Writer.Write(response)
}
```

**변경 후 (Next.js + Go API):**
```typescript
// app/page.tsx
async function HomePage() {
  // Go API에서 데이터 가져오기
  const res = await fetch('http://go-server:8080/api/home', {
    cache: 'no-store' // SSR
  });
  const data = await res.json();

  return <Home data={data} />;
}

// 또는 ISR로 성능 향상
async function HomePage() {
  const res = await fetch('http://go-server:8080/api/home', {
    next: { revalidate: 60 } // ISR: 60초마다 재생성
  });
  const data = await res.json();
  return <Home data={data} />;
}
```

### 4단계: Go API 엔드포인트 추가

```go
// Go 서버에 API 엔드포인트 추가
func main() {
    r := gin.Default()

    // 신규: Next.js용 API 엔드포인트
    api := r.Group("/api")
    {
        api.GET("/home", func(c *gin.Context) {
            data := getDataFromDB()
            c.JSON(200, data)
        })
        api.GET("/products", getProducts)
        api.GET("/users/:id", getUser)
    }

    // 기존: 마이그레이션 중 gotossr 라우트 유지
    r.GET("/", homeHandler)

    r.Run(":8080")
}
```

### 5단계: 리버스 프록시로 점진적 마이그레이션

```nginx
# nginx.conf - 경로별 라우팅
upstream nextjs {
    server localhost:3000;
}
upstream go {
    server localhost:8080;
}

server {
    listen 80;

    # 신규 페이지 → Next.js
    location /new/ {
        proxy_pass http://nextjs;
    }

    # 마이그레이션 완료 페이지 → Next.js
    location /products {
        proxy_pass http://nextjs;
    }

    # 기존 페이지 → gotossr (마이그레이션 전까지)
    location / {
        proxy_pass http://go;
    }

    # API → Go
    location /api/ {
        proxy_pass http://go;
    }
}
```

### 6단계: Docker Compose 업데이트

```yaml
# docker-compose.yml
services:
  go-api:
    build: ./go-server
    ports:
      - "8080:8080"
    environment:
      - APP_ENV=production

  nextjs:
    build: ./frontend-nextjs
    ports:
      - "3000:3000"
    environment:
      - API_URL=http://go-api:8080
    depends_on:
      - go-api

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
    depends_on:
      - go-api
      - nextjs
```

### 7단계: 타입 공유 (선택)

```bash
# OpenAPI 생성기로 타입 동기화
npm install @openapitools/openapi-generator-cli -D

# Go API에서 OpenAPI 스펙 생성 (swaggo/swag 사용)
go install github.com/swaggo/swag/cmd/swag@latest
swag init

# TypeScript 클라이언트 생성
npx openapi-generator-cli generate \
  -i http://localhost:8080/swagger/doc.json \
  -g typescript-fetch \
  -o frontend-nextjs/lib/api
```

### 마이그레이션 체크리스트

```markdown
## 페이지 마이그레이션 추적

| 페이지 | gotossr | Next.js | 검증 |
|--------|-------------|---------|------|
| / (홈) | ✅ | ⬜ | ⬜ |
| /products | ✅ | ⬜ | ⬜ |
| /dashboard | ✅ | ⬜ | ⬜ |
| /settings | ✅ | ⬜ | ⬜ |

## API 엔드포인트

| 엔드포인트 | 생성 | 테스트 |
|-----------|------|--------|
| GET /api/home | ⬜ | ⬜ |
| GET /api/products | ⬜ | ⬜ |
| GET /api/user/:id | ⬜ | ⬜ |
```

### 예상 일정

| 단계 | 기간 | 작업 |
|------|------|------|
| 설정 | 1-2일 | Next.js 프로젝트, Docker, Nginx |
| 단순 페이지 | 1주 | 정적 페이지, 기본 데이터 페칭 |
| 복잡한 페이지 | 2-3주 | 폼, 인증, 실시간 기능 |
| 테스트 | 1주 | E2E 테스트, 성능 비교 |
| 전환 | 1일 | DNS 변경, 모니터링 |

**총 소요: 일반적인 프로젝트 4-6주**

### 롤백 계획

문제 발생 시 롤백은 간단합니다:

```nginx
# nginx.conf - gotossr로 롤백
location / {
    proxy_pass http://go;  # 모든 트래픽을 Go로
}
```

# CSS 프레임워크 지원

gotossr은 다양한 CSS 프레임워크를 지원합니다:

### Tailwind CSS v4

```bash
npm install tailwindcss @tailwindcss/cli
```

```css
/* src/Main.css */
@import "tailwindcss";

@theme {
  --color-primary: #3b82f6;
}
```

### Bootstrap 5

```bash
npm install bootstrap@5.1.3 react-bootstrap
```

```tsx
import { Button, Container } from "react-bootstrap";
import "bootstrap/dist/css/bootstrap.min.css";

function Home() {
  return (
    <Container>
      <Button variant="primary">클릭</Button>
    </Container>
  );
}
```

### 예제 프로젝트

| 프레임워크 | 디렉토리 | 설명 |
|-----------|----------|------|
| Plain CSS | `examples/frontend/` | 기본 CSS |
| Tailwind v4 | `examples/frontend-tailwind/` | @theme 사용 |
| Bootstrap 5 | `examples/frontend-bootstrap/` | react-bootstrap |
| MUI | `examples/frontend-mui/` | Material UI |

# 프로젝트 구조

```
your-project/
├── main.go                 # Go 진입점
├── models/
│   └── props.go            # Props 구조체 (자동으로 TS 변환)
├── frontend/
│   ├── src/
│   │   ├── Home.tsx        # React 컴포넌트
│   │   ├── Layout.tsx      # 레이아웃 (선택)
│   │   └── generated.d.ts  # 자동 생성 타입
│   └── package.json
└── go.mod
```

# 기여

기여를 환영합니다! Pull Request를 자유롭게 제출해주세요.

# 라이선스

MIT License - 자세한 내용은 [LICENSE](../LICENSE)를 참조하세요.
