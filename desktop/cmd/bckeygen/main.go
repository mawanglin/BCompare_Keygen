// bckeygen 是 Beyond Compare 5 注册机的桌面窗口版（Fyne）。
//
// 两步合一的纯桌面窗口：
//
//	① 修补程序：把 Beyond Compare 可执行文件里内置的 RSA 公钥改成与注册机匹配的版本；
//	② 生成注册码：填入信息生成注册码，一键复制，并展示解析详情。
//
// 作者/维护者：Cob
package main

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/mawanglin/BCompare_Keygen/desktop/internal/keygen"
	"github.com/mawanglin/BCompare_Keygen/desktop/internal/patcher"
)

func main() {
	a := app.NewWithID("com.cob.bckeygen")
	a.Settings().SetTheme(theme.DefaultTheme())
	w := a.NewWindow("BCompare Keygen · by Cob")
	w.Resize(fyne.NewSize(580, 720))

	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("① 修补程序", theme.DocumentCreateIcon(), patchTab(w)),
		container.NewTabItemWithIcon("② 生成注册码", theme.ConfirmIcon(), keygenTab(w)),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	w.SetContent(tabs)
	w.ShowAndRun()
}

// ---- ① 修补程序 ----

func patchTab(w fyne.Window) fyne.CanvasObject {
	pathEntry := widget.NewEntry()
	pathEntry.SetPlaceHolder("Beyond Compare 可执行文件路径")
	if p := patcher.FindExecutable(); p != "" {
		pathEntry.SetText(p)
	}

	status := widget.NewLabel("请选择 Beyond Compare 可执行文件后点击「检测」。")
	status.Wrapping = fyne.TextWrapWord

	refresh := func() {
		p := strings.TrimSpace(pathEntry.Text)
		if p == "" {
			status.SetText("尚未选择文件。")
			return
		}
		st, oc, nc, err := patcher.Status(p)
		if err != nil {
			status.SetText("读取失败：" + err.Error())
			return
		}
		status.SetText(fmt.Sprintf("状态：%s\n原始公钥 %d 处，已改公钥 %d 处。", st, oc, nc))
	}

	browse := widget.NewButtonWithIcon("浏览…", theme.FolderOpenIcon(), func() {
		dlg := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
			if err != nil || rc == nil {
				return
			}
			defer rc.Close()
			pathEntry.SetText(rc.URI().Path())
			refresh()
		}, w)
		dlg.Resize(fyne.NewSize(700, 500))
		// 默认定位到常见安装目录
		if p := patcher.FindExecutable(); p != "" {
			if lister, e := storage.ListerForURI(storage.NewFileURI(parentDir(p))); e == nil {
				dlg.SetLocation(lister)
			}
		}
		dlg.Show()
	})

	detectBtn := widget.NewButtonWithIcon("检测", theme.SearchIcon(), refresh)

	patchBtn := widget.NewButtonWithIcon("修补", theme.ConfirmIcon(), func() {
		p := strings.TrimSpace(pathEntry.Text)
		if p == "" {
			dialog.ShowError(fmt.Errorf("请先选择 Beyond Compare 可执行文件"), w)
			return
		}
		dialog.ShowConfirm("确认修补",
			fmt.Sprintf("将修改：\n%s\n\n会先自动备份为同名 .bak 文件。继续？", p),
			func(ok bool) {
				if !ok {
					return
				}
				n, err := patcher.Patch(p, true)
				if err != nil {
					dialog.ShowError(fmt.Errorf("修补失败：%w", err), w)
					return
				}
				if n == 0 {
					dialog.ShowInformation("无需修补", "该文件已是修补后的公钥。", w)
					refresh()
					return
				}
				msg := fmt.Sprintf("修补成功：替换 %d 处公钥。\n备份：%s.bak", n, p)
				if runtime.GOOS == "darwin" {
					if e := patcher.ResignMacho(p); e != nil {
						msg += "\n\n⚠️ 自动重签名失败：" + e.Error() +
							"\n请手动执行：codesign -f -s - \"" + p + "\""
					} else {
						msg += "\n已自动 ad-hoc 重签名。"
					}
				}
				dialog.ShowInformation("完成", msg, w)
				refresh()
			}, w)
	})
	patchBtn.Importance = widget.HighImportance

	note := widget.NewLabelWithStyle(
		"说明：修补会改动可执行文件，签名随之失效（Windows 仍可运行；macOS 已自动 ad-hoc 重签名）。"+
			"修补前请关闭 Beyond Compare。",
		fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
	note.Wrapping = fyne.TextWrapWord

	form := container.NewBorder(nil, nil, nil, browse, pathEntry)
	content := container.NewVBox(
		widget.NewLabelWithStyle("第一步：处理 RSA 公钥", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabel("可执行文件："),
		form,
		container.NewHBox(detectBtn, patchBtn),
		widget.NewSeparator(),
		widget.NewCard("检测状态", "", status),
		note,
	)
	return container.NewVScroll(content)
}

func parentDir(p string) string {
	i := strings.LastIndexAny(p, `/\`)
	if i <= 0 {
		return p
	}
	return p[:i]
}

// ---- ② 生成注册码 ----

func keygenTab(w fyne.Window) fyne.CanvasObject {
	username := widget.NewEntry()
	username.SetText("Test")
	organization := widget.NewEntry()
	organization.SetText("Test Studio")
	serial := widget.NewEntry()
	serial.SetText("Abcd-1234")
	serial.Validator = func(s string) error {
		if !keygen.CheckSerial(s) {
			return fmt.Errorf("格式须为 xxxx-xxxx（4 位+4 位字母数字）")
		}
		return nil
	}
	quantity := widget.NewEntry()
	quantity.SetText("1")
	quantity.Validator = func(s string) error {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil || n <= 0 {
			return fmt.Errorf("请输入正整数")
		}
		return nil
	}

	form := widget.NewForm(
		widget.NewFormItem("用户名", username),
		widget.NewFormItem("组织名", organization),
		widget.NewFormItem("序列号", serial),
		widget.NewFormItem("数量", quantity),
	)

	keyOutput := widget.NewMultiLineEntry()
	keyOutput.Wrapping = fyne.TextWrapBreak
	keyOutput.SetMinRowsVisible(7)
	keyOutput.Disable()

	copyBtn := widget.NewButtonWithIcon("复制", theme.ContentCopyIcon(), func() {
		if strings.TrimSpace(keyOutput.Text) == "" {
			return
		}
		w.Clipboard().SetContent(keyOutput.Text)
		dialog.ShowInformation("已复制", "注册码已复制到剪贴板", w)
	})

	details := widget.NewLabel("")
	details.Wrapping = fyne.TextWrapWord

	generate := func() {
		if err := serial.Validate(); err != nil {
			dialog.ShowError(fmt.Errorf("序列号错误：%w", err), w)
			return
		}
		n, err := strconv.Atoi(strings.TrimSpace(quantity.Text))
		if err != nil || n <= 0 {
			n = 1
			quantity.SetText("1")
		}
		req := keygen.Request{
			Username:     fallback(username.Text, "Test"),
			Organization: fallback(organization.Text, "Test Studio"),
			SerialNumber: fallback(serial.Text, "Abcd-1234"),
			Quantity:     n,
		}
		key, err := keygen.Encode(req)
		if err != nil {
			dialog.ShowError(fmt.Errorf("生成失败：%w", err), w)
			return
		}
		keyOutput.SetText(key)

		info, derr := keygen.Decode(key)
		if derr != nil {
			details.SetText("（注册码已生成，但解析详情失败：" + derr.Error() + "）")
			return
		}
		details.SetText(fmt.Sprintf(
			"版本：%s\n用户名：%s\n组织名：%s\n序列号：%s\n数量：%d\n随机值：%s",
			info.Version, info.Username, info.Organization,
			info.SerialNumber, info.Quantity, info.Random))
	}

	genBtn := widget.NewButtonWithIcon("生成密钥", theme.ConfirmIcon(), generate)
	genBtn.Importance = widget.HighImportance

	resultCard := widget.NewCard("生成结果", "",
		container.NewVBox(keyOutput, container.NewHBox(copyBtn)))
	detailCard := widget.NewCard("注册码解析", "", details)

	content := container.NewVBox(
		widget.NewLabelWithStyle("第二步：生成注册码", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		form,
		genBtn,
		widget.NewSeparator(),
		resultCard,
		detailCard,
		widget.NewLabelWithStyle("仅供学习研究使用 · by Cob",
			fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
	)
	return container.NewVScroll(content)
}

func fallback(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}
