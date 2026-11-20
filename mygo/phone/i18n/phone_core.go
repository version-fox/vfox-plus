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

package i18n

// The messages of the connection that the whole app shows: the kinds of error
// that the service reports (app.KindOf), which the window says as a snack bar on
// whatever screen is open.
func init() {
	Register(map[string][2]string{
		"errNotLink": {
			"That is not a pairing link.",
			"这不是配对链接。",
		},
		"errPairingRefused": {
			"The pairing link is invalid, already used or expired. Make a new one on the computer.",
			"配对链接无效、已使用或已过期，请在电脑上重新生成一个。",
		},
		"errUnpaired": {
			"The computer does not accept this phone any more. Pair it again.",
			"电脑不再接受这台手机，请重新配对。",
		},
		"errCertificateChanged": {
			"The computer's certificate has changed, so the connection was refused. Compare its fingerprint with the one on the computer before pairing again.",
			"电脑的证书已经改变，连接已被拒绝。请先与电脑上的指纹核对，再重新配对。",
		},
		"errUnreachable": {
			"The computer did not answer. Check the network, and that vfox+ runs on the computer.",
			"电脑没有响应。请检查网络，并确认电脑上的 vfox+ 正在运行。",
		},
		"errUnavailable": {
			"vfox failed or did not answer in time.",
			"vfox 出错，或未能及时响应。",
		},
		"errNotFound": {
			"The computer no longer has this.",
			"电脑上已经没有这一项。",
		},
		"errRejected": {
			"The computer refused the request.",
			"电脑拒绝了这个请求。",
		},
		"errServer": {
			"The computer failed to answer the request.",
			"电脑处理这个请求时出错。",
		},
		"errOther": {
			"Something went wrong.",
			"出了点问题。",
		},
	})
}
