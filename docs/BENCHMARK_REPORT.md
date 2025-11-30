# 벤치마크 리포트

**테스트 일시**: 2025-11-30
**테스트 도구**: hey

## 테스트 대상

| 경로 | 스택 | DB |
|-----|------|-----|
| localhost:8080/ | Go + v8go SSR | 없음 |
| localhost:8080/board | Go + v8go SSR | SQLite (로컬) |
| hae.test | PHP-FPM + Nginx | MySQL (도커) |

## 결과

### c=10, n=100
| 경로 | RPS | 99% Latency |
|-----|-----|-------------|
| localhost/ | 1,533 | 12ms |
| localhost/board | 1,598 | 11ms |
| hae.test | 94 | 218ms |

### c=50, n=500
| 경로 | RPS | 99% Latency |
|-----|-----|-------------|
| localhost/ | 1,847 | 54ms |
| localhost/board | 3,524 | 35ms |
| hae.test | 107 | 652ms |

## 결론

| 항목 | Go+v8go | PHP-FPM |
|-----|---------|---------|
| RPS | 1,500~3,500 | ~100 |
| 99% Latency | 11~54ms | 218~652ms |
| 성능 배수 | **17~33x** | 기준 |

**Go+v8go SSR이 PHP-FPM보다 17~33배 빠름**

## 핵심 수정사항

Blocking Pool 적용 (V8 GC 스파이크 해결):
```go
// runtime.go:97-102
func (p *Pool) Get() JSRuntime {
    return <-p.pool  // 블로킹 방식으로 대기
}
```

상세: [V8_GC_OPTIMIZATION.md](V8_GC_OPTIMIZATION.md)
