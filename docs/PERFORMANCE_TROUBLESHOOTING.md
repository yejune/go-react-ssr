# gotossr 성능 트러블슈팅 기록

## 문제

벤치마크 테스트에서 99% latency가 ~2초 spike 발생

```
Response time histogram:
  0.019-0.323s [479개] ← 96% 정상
  1.233s       [15개]  ← spike
  2.143s       [3개]   ← spike
  3.053s       [2개]   ← spike
```

## 테스트 환경

- macOS Darwin 25.0.0
- Go 1.24
- V8: tommie/v8go v0.34.0
- Pool size: 10
- 벤치마크: `hey -n 500 -c 10 http://localhost:8080/`

## 기준 성능

| 지표 | 값 |
|-----|-----|
| RPS | ~76-90 |
| 50% latency | 27ms |
| 90% latency | 35ms |
| 99% latency | **2.0s** (문제) |

## 시도한 해결책

### 1. V8 라이브러리 교체

rogchap/v8go → tommie/v8go

**결과:** 효과 없음. 성능 동일.

### 2. V8 플래그 튜닝

```go
v8.SetFlags(
    "--max-semi-space-size=64",  // young gen 확대
    "--max-old-space-size=256",  // old gen 제한
)
```

**결과:** 효과 없음. 99% 여전히 2초.

### 3. V8 힙 제한 + 모니터링

```go
isolate := v8.NewIsolate(v8.WithResourceConstraints(16MB, 128MB))

// Reset()에서 힙 80% 초과 시 isolate 재생성
if usage > 0.80 {
    v.isolate.Dispose()
    v.isolate = v8.NewIsolate(...)
}
```

**결과:** 효과 없음.

### 4. Isolate 주기적 재생성

```go
const isolateRecycleInterval = 20 // 또는 50, 100

func (v *V8Runtime) Reset() {
    v.requestCount++
    if v.requestCount >= isolateRecycleInterval {
        v.isolate.Dispose()
        v.isolate = v8.NewIsolate()
        v.requestCount = 0
    }
}
```

**결과:** 불안정. 99%가 1초~2초 왔다갔다. RPS도 47까지 떨어지는 경우 있음.

### 5. Blocking Pool

```go
// 새 isolate 생성 대신 기존 것 대기
func (p *Pool) Get() JSRuntime {
    return <-p.pool  // blocking
}
```

**결과:** 더 나빠짐. Slowest 4.07s, RPS 65.6.

### 6. Go GC 튜닝 (GOGC)

```bash
GOGC=200 ./server  # GC 빈도 절반
GOGC=off ./server  # GC 완전 끔
```

**결과:** 둘 다 효과 없음. 99% 여전히 2초.

**중요 발견:** GOGC=off도 spike 발생 → **Go GC가 원인이 아님**

### 7. Pool 크기 증가

```go
JSRuntimePoolSize: 20  // 10 → 20
```

**결과:** 효과 없음. Pool > 동시성이어도 spike 발생.

## 동시성과 Spike 관계

| 동시성 | Slowest | spike 개수 |
|--------|---------|----------|
| c=5 | 1.04s | 9개 |
| c=10 | 3.05s | 20개 |
| c=20 | 4.0s+ | 더 많음 |

동시성이 높을수록 spike가 심해짐. 하지만 Pool 크기를 늘려도 해결 안 됨.

## 결론

### 원인 분석

spike는 다음이 **아님**:
- Go GC (GOGC=off도 동일)
- Pool 대기 (Pool > 동시성도 동일)
- V8 라이브러리 버전 (rogchap/tommie 동일)

spike 원인 추정:
- V8 내부 JIT 컴파일
- V8 내부 GC (Go GC와 별개)
- V8 isolate/context 생성 비용 누적
- 알 수 없는 V8 내부 동작

### localhost 환경 문제 아님

같은 localhost에서 PHP-FPM + nginx (hae.test) 테스트 결과:

| 서버 | RPS | 50% | 99% |
|------|-----|-----|-----|
| gotossr (V8) | 82 | 28ms | **2.0s** |
| PHP-FPM + nginx | 110 | 87ms | **153ms** |

PHP-FPM은 99% 153ms로 안정적. **localhost 환경 문제가 아니라 gotossr/V8 자체의 문제.**

### 해결되지 않은 문제

2초 spike는 gotossr/Go 레벨에서 해결 불가. V8 내부 동작 문제로 추정.

### 현재 상태 (2025-11-30)

- tommie/v8go v0.34.0 사용 (유지보수 활발)
- JSRuntimePoolSize: 10
- 모든 실험적 설정 제거, 원본 코드 유지

### V8 프로파일링 결과 (2025-11-30)

tommie/v8go의 CPUProfiler를 사용한 분석 결과:

**순차 실행 (단일 isolate):**
```
Avg execution: 282µs
Min execution: 204µs
Max execution: 3.2ms
Max/Min ratio: 15.80x
V8 GC: 6.3%
```

**동시 실행 (10개 isolate):**
```
Avg execution: 484µs
Min execution: 168µs
Max execution: 8.9ms
Max/Min ratio: 52.95x  ← 동시성이 spike를 3배 악화!
```

**결론:**
1. V8 실행 시간에 15~53배 변동이 존재
2. 동시성이 높을수록 spike가 심해짐
3. V8 GC가 6-10% CPU를 차지
4. 실제 React SSR 번들(~900KB)은 테스트 코드보다 100배 이상 큼

→ 동시성 + 큰 번들 + V8 GC 변동 = 2초 spike

### 가능한 개선 방향

1. **React 번들 크기 감소** - 불필요한 코드 제거, code splitting
2. **Preact 사용** - React 45KB → Preact 3KB
3. **SSR 캐싱** - 같은 경로는 HTML 캐시 (TTL 필요)
4. **동시성 제한** - Pool 크기를 줄여서 동시 isolate 수 제한
5. **Streaming SSR** - 전체 렌더링 대기 대신 스트리밍

### 현재 한계

V8의 실행 시간 변동은 Go/gotossr 레벨에서 제어 불가능. V8 내부 동작(JIT, GC)에 의존.

## 참고: V8 vs QuickJS 비교

| 런타임 | RPS | 50% | 99% |
|--------|-----|-----|-----|
| V8 | 82 | 28ms | 2.0s |
| QuickJS | 36 | 196ms | 1.9s |

V8이 평균 성능은 2배 이상 빠르지만, 99% spike는 둘 다 비슷.
