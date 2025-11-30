package reactbuilder

import (
	"encoding/json"
	"fmt"
	"strings"

	esbuildApi "github.com/evanw/esbuild/pkg/api"
	"github.com/yejune/gotossr/internal/utils"
)

var loaders = map[string]esbuildApi.Loader{
	".png":   esbuildApi.LoaderFile,
	".svg":   esbuildApi.LoaderFile,
	".jpg":   esbuildApi.LoaderFile,
	".jpeg":  esbuildApi.LoaderFile,
	".gif":   esbuildApi.LoaderFile,
	".bmp":   esbuildApi.LoaderFile,
	".woff2": esbuildApi.LoaderFile,
	".woff":  esbuildApi.LoaderFile,
	".ttf":   esbuildApi.LoaderFile,
	".eot":   esbuildApi.LoaderFile,
}

var globalThisPolyfill = `var globalThis=typeof globalThis!=="undefined"?globalThis:this;`
var textEncoderPolyfill = `function TextEncoder(){}TextEncoder.prototype.encode=function(string){var octets=[];var length=string.length;var i=0;while(i<length){var codePoint=string.codePointAt(i);var c=0;var bits=0;if(codePoint<=0x0000007F){c=0;bits=0x00}else if(codePoint<=0x000007FF){c=6;bits=0xC0}else if(codePoint<=0x0000FFFF){c=12;bits=0xE0}else if(codePoint<=0x001FFFFF){c=18;bits=0xF0}octets.push(bits|(codePoint>>c));c-=6;while(c>=0){octets.push(0x80|((codePoint>>c)&0x3F));c-=6}i+=codePoint>=0x10000?2:1}return octets};function TextDecoder(){}TextDecoder.prototype.decode=function(octets){var string="";var i=0;while(i<octets.length){var octet=octets[i];var bytesNeeded=0;var codePoint=0;if(octet<=0x7F){bytesNeeded=0;codePoint=octet&0xFF}else if(octet<=0xDF){bytesNeeded=1;codePoint=octet&0x1F}else if(octet<=0xEF){bytesNeeded=2;codePoint=octet&0x0F}else if(octet<=0xF4){bytesNeeded=3;codePoint=octet&0x07}if(octets.length-i-bytesNeeded>0){var k=0;while(k<bytesNeeded){octet=octets[i+k+1];codePoint=(codePoint<<6)|(octet&0x3F);k+=1}}else{codePoint=0xFFFD;bytesNeeded=octets.length-i}string+=String.fromCodePoint(codePoint);i+=bytesNeeded+1}return string};`
var processPolyfill = `var process = {env: {NODE_ENV: "production"}};`
var consolePolyfill = `globalThis.__ssr_errors=[];var console = {log: function(){},warn: function(){},error: function(){var a=Array.prototype.slice.call(arguments);globalThis.__ssr_errors.push(a.map(function(x){return x&&x.stack?x.stack:String(x)}).join(' '));}};`
var urlPolyfill = `if(typeof URL==="undefined"){function URL(u,b){if(b&&u.indexOf("://")===-1){u=b.replace(/\/$/,"")+"/"+u.replace(/^\//,"")}var m=u.match(/^(([^:/?#]+):)?(\/\/([^/?#]*))?([^?#]*)(\?([^#]*))?(#(.*))?/);this.href=u;this.protocol=(m[2]||"")+ ":";this.host=m[4]||"";this.hostname=this.host.split(":")[0];this.port=this.host.split(":")[1]||"";this.pathname=m[5]||"/";this.search=m[6]||"";this.hash=m[8]||"";this.origin=this.protocol+"//"+this.host}URL.prototype.toString=function(){return this.href}}`

type BuildResult struct {
	JS           string
	CSS          string
	Dependencies []string
}

func BuildServer(buildContents, frontendDir, assetRoute string) (BuildResult, error) {
	opts := esbuildApi.BuildOptions{
		Stdin: &esbuildApi.StdinOptions{
			Contents:   buildContents,
			Loader:     esbuildApi.LoaderTSX,
			ResolveDir: frontendDir,
		},
		Platform:          esbuildApi.PlatformNode,
		Bundle:            true,
		Write:             false,
		Outdir:            "/",
		Metafile:          false,
		AssetNames:        fmt.Sprintf("%s/[name]", strings.TrimPrefix(assetRoute, "/")),
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		Loader:            loaders,
		// Remove legal comments so they don't interfere with eval result
		LegalComments: esbuildApi.LegalCommentsNone,
		// We can inject the polyfills at the top of the generated js
		Banner: map[string]string{
			"js": globalThisPolyfill + urlPolyfill + textEncoderPolyfill + processPolyfill + consolePolyfill,
		},
		// Footer returns globalThis.__ssr_result - never affected by minification
		// Also prepends any console.error messages as HTML comment for debugging
		Footer: map[string]string{
			"js": "globalThis.__ssr_result+(globalThis.__ssr_errors&&globalThis.__ssr_errors.length?'<!-- SSR_ERRORS: '+globalThis.__ssr_errors.join(' | ')+' -->':'')",
		},
	}
	return build(opts, false)
}

func BuildClient(buildContents, frontendDir, assetRoute string, minify bool) (BuildResult, error) {
	opts := esbuildApi.BuildOptions{
		Stdin: &esbuildApi.StdinOptions{
			Contents:   buildContents,
			Loader:     esbuildApi.LoaderTSX,
			ResolveDir: frontendDir,
		},
		Bundle:            true,
		Write:             false,
		Outdir:            "/",
		Metafile:          true,
		AssetNames:        fmt.Sprintf("%s/[name]", strings.TrimPrefix(assetRoute, "/")),
		MinifyWhitespace:  minify,
		MinifyIdentifiers: minify,
		MinifySyntax:      minify,
		Loader:            loaders,
	}
	return build(opts, true)
}

func build(buildOptions esbuildApi.BuildOptions, isClient bool) (BuildResult, error) {
	result := esbuildApi.Build(buildOptions)
	if len(result.Errors) > 0 {
		fileLocation := "unknown"
		lineNum := "unknown"
		if result.Errors[0].Location != nil {
			fileLocation = result.Errors[0].Location.File
			lineNum = result.Errors[0].Location.LineText
		}
		return BuildResult{}, fmt.Errorf("%s <br>in %s <br>at %s", result.Errors[0].Text, fileLocation, lineNum)
	}

	var br BuildResult
	for _, file := range result.OutputFiles {
		if strings.HasSuffix(file.Path, "stdin.js") {
			br.JS = string(file.Contents)
		} else if strings.HasSuffix(file.Path, "stdin.css") {
			br.CSS = string(file.Contents)
		}
	}
	if isClient {
		br.Dependencies = getDependencyPathsFromMetafile(result.Metafile)
	}
	return br, nil
}

// metafileSchema represents the structure of esbuild metafile
type metafileSchema struct {
	Inputs map[string]interface{} `json:"inputs"`
}

// getDependencyPathsFromMetafile parses dependencies from esbuild metafile and returns the paths of the dependencies
func getDependencyPathsFromMetafile(metafile string) []string {
	var meta metafileSchema
	if err := json.Unmarshal([]byte(metafile), &meta); err != nil {
		return nil
	}

	var dependencyPaths []string
	// Ignore dependencies in node_modules
	for key := range meta.Inputs {
		if !strings.Contains(key, "/node_modules/") {
			dependencyPaths = append(dependencyPaths, utils.GetFullFilePath(key))
		}
	}
	return dependencyPaths
}
