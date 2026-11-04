// Copyright 2026 Han Li and contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gateway

import (
	"html/template"
	"net/http"

	"github.com/version-fox/vfox-plus/mygo/pair"
)

// The pages the gateway serves itself, one template with a part for each. They are forms that post to
// the gateway and need no script, in both languages as the intro page of the servers of the application
// is, and the look of a page of the settings: a card, the colours of the system, light and dark.
const (
	pageIntro  = "intro"  // a browser that is not paired with the gateway.
	pageHome   = "home"   // a paired browser: to which instance the gateway is paired, and the form to pair it.
	pageTrust  = "trust"  // the fingerprint of a certificate that nobody vouches for, to be compared.
	pageInvite = "invite" // the pairing link of another browser.
	pageFailed = "failed" // the instance cannot be reached.
)

// view is what a page shows.
type view struct {
	Page    string
	Favicon template.HTML
	// Fingerprint is the certificate of the gateway (the intro) or of the instance (the page to trust it).
	Fingerprint string
	// Address is the instance that the gateway is paired with or is to be paired with.
	Address string
	// Connected says that the gateway holds a session at the instance.
	Connected bool
	// Trusted is the fingerprint of the certificate of the instance that was trusted.
	Trusted string
	// Changed says that the certificate at the address is not the one that was trusted before.
	Changed bool
	// Link is the pairing link of the instance, carried by the page that asks for the fingerprint to be
	// compared; or, in the page of a link for another browser, that link.
	Link string
	// Error is what went wrong with the last thing asked.
	Error string
}

var pageTemplate = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><meta name="color-scheme" content="light dark">
<title>vfox+ gateway</title>{{.Favicon}}
<style>
body{font:16px/1.5 system-ui,-apple-system,"Segoe UI","Noto Sans","Noto Sans CJK SC","PingFang SC","Microsoft YaHei",sans-serif;margin:0;padding:24px 16px;background:Canvas;color:CanvasText}
main{max-width:40rem;margin:0 auto}
h1{font-size:1.5rem;margin:0 0 .5rem} h1 small{font-weight:400;opacity:.7;font-size:1rem}
.card{border:1px solid color-mix(in srgb,CanvasText 20%,transparent);border-radius:16px;padding:16px 20px;margin:16px 0}
.zh{opacity:.85}
code{font:.9rem ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;word-break:break-all;background:color-mix(in srgb,CanvasText 8%,transparent);padding:2px 6px;border-radius:6px}
input[type=text]{width:100%;box-sizing:border-box;font:inherit;padding:10px 12px;border-radius:10px;border:1px solid color-mix(in srgb,CanvasText 35%,transparent);background:Canvas;color:CanvasText}
button,.button{font:inherit;padding:8px 18px;border-radius:999px;border:1px solid color-mix(in srgb,CanvasText 35%,transparent);background:Canvas;color:CanvasText;cursor:pointer;text-decoration:none;display:inline-block}
button.primary,.button.primary{background:#6750a4;color:#fff;border-color:transparent;background:AccentColor;color:AccentColorText}
.row{display:flex;gap:8px;flex-wrap:wrap;margin-top:12px}
.error{border-color:#b3261e;color:#b3261e} .warn{border-color:#b26a00}
form{margin:0}
</style></head>
<body><main>
<h1>vfox+ <small>gateway · 网关</small></h1>
{{if .Error}}<div class="card error" role="alert"><p>{{.Error}}</p></div>{{end}}
{{if eq .Page "intro"}}
<div class="card">
<p>This is the gateway of vfox+. It shows the web interface of one vfox+ on your network in this browser. To use it, pair this browser with the one-time link that the gateway printed when it started (in a container: <code>docker logs</code>, and <code>docker restart</code> makes a new link). Links work once and expire after five minutes.</p>
<p class="zh">这是 vfox+ 的网关。它把网络中某一台 vfox+ 的网页界面显示在这个浏览器里。要使用它，请先用网关启动时打印的一次性链接配对此浏览器（容器里用 <code>docker logs</code> 查看，<code>docker restart</code> 会生成新链接）。链接一次性有效，五分钟后过期。</p>
{{if .Fingerprint}}<p>Certificate fingerprint of the gateway: <code>{{.Fingerprint}}</code></p>
<p class="zh">网关的证书指纹：<code>{{.Fingerprint}}</code>（信任自签名证书前请与日志里的核对）。</p>{{end}}
</div>
{{else if eq .Page "home"}}
<div class="card">
{{if .Connected}}
<p>Paired with vfox+ at <code>{{.Address}}</code>.</p>
<p class="zh">已与 <code>{{.Address}}</code> 的 vfox+ 配对。</p>
{{if .Trusted}}<p>Trusted certificate: <code>{{.Trusted}}</code></p>{{end}}
<div class="row"><a class="button primary" href="/app">Open vfox+ · 打开 vfox+</a></div>
{{else if .Address}}
<p>The session at <code>{{.Address}}</code> has ended: a session lasts a day and ends with vfox+. Pair the gateway again with a new link.</p>
<p class="zh">与 <code>{{.Address}}</code> 的会话已结束：会话最长一天，vfox+ 退出时也会结束。请用新的配对链接重新配对。</p>
{{else}}
<p>The gateway is not paired with a vfox+ yet.</p>
<p class="zh">网关还没有与任何 vfox+ 配对。</p>
{{end}}
</div>
<div class="card">
<form method="post" action="/gateway/connect">
<p><label for="link"><strong>Pairing link of vfox+ · vfox+ 的配对链接</strong></label></p>
<input type="text" id="link" name="link" required autocomplete="off" autocapitalize="off" spellcheck="false" placeholder="https://192.168.1.20:8369/auth/connect/…">
<p>On the computer that runs vfox+: Settings → Web interface → Other devices (allow them), then Pair a phone → Show code → Copy link. It works once and expires after five minutes.</p>
<p class="zh">在运行 vfox+ 的电脑上：设置 → 网页界面 → 其他设备（允许），再点 配对手机 → 显示二维码 → 复制链接。链接一次性有效，五分钟后过期。</p>
<div class="row"><button class="primary" type="submit">Pair · 配对</button></div>
</form>
</div>
<div class="card">
<div class="row" style="margin-top:0">
<form method="post" action="/gateway/invite"><button type="submit">Pair another browser · 配对另一个浏览器</button></form>
{{if .Address}}<form method="post" action="/gateway/forget"><button type="submit">Forget this vfox+ · 忘记此 vfox+</button></form>{{end}}
</div>
</div>
{{else if eq .Page "trust"}}
<div class="card {{if .Changed}}warn{{end}}">
{{if .Changed}}<p><strong>The certificate at this address is not the one you trusted before.</strong> If you did not change the certificate of vfox+, someone else answers here.</p>
<p class="zh"><strong>此地址上的证书与你之前信任的不是同一个。</strong>如果你没有更换 vfox+ 的证书，说明有别的主机在应答。</p>{{end}}
<p>The certificate of vfox+ at <code>{{.Address}}</code> is not signed by an authority, as the one that vfox+ makes for itself is not. Before the gateway trusts it, compare its fingerprint with the one in the settings of vfox+ (Settings → Web interface → Certificate):</p>
<p class="zh"><code>{{.Address}}</code> 上 vfox+ 的证书没有机构签名，vfox+ 为自己生成的证书都是这样。信任它之前，请把它的指纹与 vfox+ 设置里的核对（设置 → 网页界面 → 证书）：</p>
<p><code>{{.Fingerprint}}</code></p>
<p>If they are the same, trust it. If they are not, do not: someone else answers at this address.</p>
<p class="zh">一致就信任它；不一致就不要信任，说明有别的主机在这个地址上应答。</p>
<div class="row">
<form method="post" action="/gateway/trust"><input type="hidden" name="link" value="{{.Link}}"><input type="hidden" name="fingerprint" value="{{.Fingerprint}}"><button class="primary" type="submit">They are the same: trust and pair · 一致，信任并配对</button></form>
<a class="button" href="/">Cancel · 取消</a>
</div>
</div>
{{else if eq .Page "invite"}}
<div class="card">
<p>Open this link in the other browser. It works once and expires after five minutes.</p>
<p class="zh">在另一个浏览器里打开这个链接。链接一次性有效，五分钟后过期。</p>
<p><code>{{.Link}}</code></p>
<div class="row"><a class="button" href="/">Back · 返回</a></div>
</div>
{{else if eq .Page "failed"}}
<div class="card">
<p>vfox+ at <code>{{.Address}}</code> cannot be reached just now.</p>
<p class="zh">暂时连不上 <code>{{.Address}}</code> 的 vfox+。</p>
<div class="row"><a class="button primary" href="/app">Try again · 重试</a><a class="button" href="/">Pair again · 重新配对</a></div>
</div>
{{end}}
</main></body></html>`))

// render writes a page. The pages carry nothing that may be kept, and say what they may not be framed by.
func render(writer http.ResponseWriter, status int, page view) {
	page.Favicon = template.HTML(pair.FaviconLink()) //nolint:gosec // The favicon link is a constant of this program.
	header := writer.Header()
	header.Set("Content-Type", "text/html; charset=utf-8")
	header.Set("Cache-Control", "no-store")
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; img-src data:; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
	writer.WriteHeader(status)
	_ = pageTemplate.Execute(writer, page)
}
