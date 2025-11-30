# V8 GC 스파이크 해결

## 문제
v8go SSR에서 1초 주기 latency 스파이크 발생.
```
정상: 20-30ms
스파이크: 1-2초
```

## 원인
Pool이 비었을 때 새 V8 Isolate를 동적 생성하면서 V8 내부 충돌 발생.

```go
// 문제 코드
func (p *Pool) Get() JSRuntime {
    select {
    case rt := <-p.pool:
        return rt
    default:
        return p.createRuntime()  // 동적 생성
    }
}
```

## 해결
Blocking Pool로 변경. 고정된 수의 Isolate만 사용.

```go
// runtime.go:97-102
func (p *Pool) Get() JSRuntime {
    return <-p.pool  // 대기
}
```

## 결과
| 지표 | 이전 | 이후 |
|-----|------|------|
| 99% Latency | 1-2초 | 50-100ms |
| 스파이크 | 1초 주기 | 없음 |

## 효과 없었던 시도
- V8 플래그 튜닝 (`--max-old-space-size`, `--gc-interval`, `--incremental-marking`)
- Go GC 튜닝 (`GOGC=200`, `GOGC=off`)
- `LowMemoryNotification()` 호출
- `RequestGarbageCollection(GCTypeMinor)` 주기적 호출
- Pool 크기 변경
- Isolate 주기적 재생성
