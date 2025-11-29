# QuickJS + React Router SSR 트러블슈팅

## 증상

```tsx
// 작동함
renderToString(<App {...props} />);

// 빈 문자열 반환
renderToString(<StaticRouter location={url}><App /></StaticRouter>);
```

- esbuild 번들링 성공
- QuickJS 실행 시 에러 없음
- 결과가 빈 문자열 `""`

## 원인

**QuickJS에 `URL` Web API가 없음**

React Router v6의 `react-router-dom/server`에 있는 `encodeLocation` 함수가 `new URL()` constructor를 사용:

```javascript
// react-router-dom/server.js
function encodeLocation(to) {
  let href = typeof to === "string" ? to : createPath(to);
  // 여기서 URL constructor 사용 - QuickJS에서 실패!
  let encoded = new URL(href, "http://localhost");
  return {
    pathname: encoded.pathname,
    search: encoded.search,
    hash: encoded.hash
  };
}
```

이 함수는 `useRoutes`, `Routes`, `Route` 컴포넌트에서 route matching 시 호출됨.

## 진단 과정

1. 단순 컴포넌트 테스트 → 작동
2. `useLocation()` 테스트 → 작동
3. `useRoutes()` 테스트 → 실패
4. try-catch + console.error 캡처 추가
5. 에러 메시지: `at encodeLocation (<input>:52766:73)`
6. 원인 확인: `URL is not defined`

## 해결 방법

`internal/reactbuilder/build.go`에 URL polyfill 추가:

```go
var urlPolyfill = `if(typeof URL==="undefined"){function URL(u,b){if(b&&u.indexOf("://")===-1){u=b.replace(/\/$/,"")+"/"+u.replace(/^\//,"")}var m=u.match(/^(([^:/?#]+):)?(\/\/([^/?#]*))?([^?#]*)(\?([^#]*))?(#(.*))?/);this.href=u;this.protocol=(m[2]||"")+ ":";this.host=m[4]||"";this.hostname=this.host.split(":")[0];this.port=this.host.split(":")[1]||"";this.pathname=m[5]||"/";this.search=m[6]||"";this.hash=m[8]||"";this.origin=this.protocol+"//"+this.host}URL.prototype.toString=function(){return this.href}}`
```

Banner에 추가:

```go
Banner: map[string]string{
    "js": globalThisPolyfill + urlPolyfill + textEncoderPolyfill + processPolyfill + consolePolyfill,
},
```

## 디버깅 팁

### 1. console.error 캡처

```go
var consolePolyfill = `globalThis.__ssr_errors=[];var console = {
  log: function(){},
  warn: function(){},
  error: function(){
    var a=Array.prototype.slice.call(arguments);
    globalThis.__ssr_errors.push(a.map(function(x){
      return x&&x.stack?x.stack:String(x)
    }).join(' '));
  }
};`
```

### 2. Footer에 에러 출력

```go
Footer: map[string]string{
    "js": "globalThis.__ssr_result+(globalThis.__ssr_errors&&globalThis.__ssr_errors.length?'<!-- SSR_ERRORS: '+globalThis.__ssr_errors.join(' | ')+' -->':'')",
},
```

### 3. render 함수에 try-catch

```go
var serverSPARouterRenderFunction = `try {
  globalThis.__ssr_result = renderToString(<StaticRouter location={props.__requestPath}><App /></StaticRouter>);
} catch(e) {
  globalThis.__ssr_errors.push('RENDER_ERROR: ' + (e.stack || e.message || String(e)));
  globalThis.__ssr_result = '';
}`
```

### 4. minification 임시 비활성화

에러 위치를 정확히 파악하려면:

```go
MinifyWhitespace:  false,
MinifyIdentifiers: false,
MinifySyntax:      false,
```

## QuickJS에 없는 다른 Web API들

향후 비슷한 문제가 발생할 수 있는 API들:

- `URL` ← 이 문서에서 해결됨
- `URLSearchParams`
- `fetch`
- `AbortController`
- `Blob`
- `FormData`
- `Headers`
- `Request`/`Response`

필요시 polyfill 추가 필요.

## 참고

- QuickJS: https://bellard.org/quickjs/
- React Router v6 SSR: https://reactrouter.com/en/main/guides/ssr
- URL Standard: https://url.spec.whatwg.org/
