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

// This file holds the messages of the pairing page, the connection states above the
// tabs and the phone card of the settings. Its screens' group owns it, and no other
// group edits it. Add the messages in an init function, with Register:
//
//	func init() {
//		Register(map[string][2]string{
//			"someKey": {"English text.", "中文文本。"},
//		})
//	}
//
// Every key is checked by the tests: a key used in the code must be registered.

package i18n

func init() {
	Register(map[string][2]string{
		"pairPaste": {
			"Paste",
			"粘贴",
		},
		"pairPhoto": {
			"Photo",
			"拍照",
		},
		"pairPicture": {
			"Picture",
			"相册",
		},
		"pairHow": {
			"How to pair",
			"如何配对",
		},
		"pairHowTitle": {
			"Pair this phone",
			"配对这台手机",
		},
		"pairHowBody": {
			"1. On your computer, open the vfox+ settings and turn on Other devices.\n2. Under Pair a phone, choose Show code. The computer shows a link and a QR code.\n3. Paste the link here, or take a photo of the code, or choose its picture.\n4. Compare the fingerprint shown here with the one under Other devices on your computer, and tap Trust and pair when they match.",
			"1. 在电脑上打开 vfox+ 的设置，打开“其他设备”。\n2. 在“配对手机”下选择“显示二维码”，电脑会显示一个链接和一个二维码。\n3. 在这里粘贴链接，或拍下二维码，或从相册选择二维码图片。\n4. 把这里显示的指纹与电脑“其他设备”中的指纹对比，一致后点击“信任并配对”。",
		},
		"pairConfirmTitle": {
			"Trust this computer?",
			"信任这台电脑吗？",
		},
		"pairConfirmBody": {
			"Compare this fingerprint with the one under Other devices in the settings of your computer. Trust the computer only when they match.",
			"请把这里的指纹与电脑设置页“其他设备”中显示的指纹对比。两者一致时才信任这台电脑。",
		},
		"pairProbing": {
			"Checking the computer…",
			"正在检查电脑…",
		},
		"pairPairing": {
			"Pairing…",
			"正在配对…",
		},
		"pairCertDiffers": {
			"The certificate of this computer is not the one paired before. Check the fingerprint carefully.",
			"这台电脑的证书与上次配对时不同，请仔细核对指纹。",
		},
		"trustAndPair": {
			"Trust and pair",
			"信任并配对",
		},
		"pairAgain": {
			"Pair again",
			"重新配对",
		},
		"pairCameraTitle": {
			"Camera access is off",
			"相机权限未开启",
		},
		"pairCameraBody": {
			"vfox+ reads the pairing code with the camera. Allow the camera in the system settings, or choose a picture of the code instead.",
			"vfox+ 需要用相机读取配对二维码。请在系统设置中允许相机，也可以改为从相册选择二维码图片。",
		},
		"openSettings": {
			"Open settings",
			"打开设置",
		},
		"pairCameraHint": {
			"Point the camera at the QR code, then take the photo.",
			"将相机对准二维码，然后拍照。",
		},
		"pairShoot": {
			"Take the photo",
			"拍摄",
		},
		"pairCameraUnavailable": {
			"The camera is not available.",
			"相机不可用。",
		},
		"pairPhotoFailed": {
			"The photo could not be read.",
			"无法读取照片。",
		},
		"pairPickFailed": {
			"The picture could not be chosen.",
			"无法选择图片。",
		},
		"pairNoQRCode": {
			"No QR code found in the picture.",
			"未找到二维码。",
		},
		"pairNotPicture": {
			"That file is not a PNG or JPEG picture.",
			"这个文件不是 PNG 或 JPEG 图片。",
		},
		"pairPictureTooLarge": {
			"The picture is too large to read. Choose a smaller one.",
			"图片太大，无法读取。请选择较小的图片。",
		},
		"pairClipboardEmpty": {
			"The clipboard holds no text.",
			"剪贴板里没有文字。",
		},
		"pairNeedsTitle": {
			"This phone must pair again",
			"需要重新配对",
		},
		"pairCertTitle": {
			"The computer's certificate changed",
			"电脑的证书已经改变",
		},
		"pairLastComputer": {
			"Last computer",
			"上次的电脑",
		},
		"sessionConnecting": {
			"Connecting to the computer…",
			"正在连接电脑…",
		},
	})
}
