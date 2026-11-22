; BCompare Keygen Windows 安装包脚本（NSIS）
; 由 scripts/build-windows.sh 调用，变量经 -D 传入。
; 作者/维护者：Cob

!ifndef APPVER
  !define APPVER "1.1.0"
!endif
!ifndef SRCEXE
  !define SRCEXE "..\dist\windows\BCompareKeygen.exe"
!endif
!ifndef SRCCLI
  !define SRCCLI "..\dist\windows\BCompareKeygen-cli.exe"
!endif
!ifndef OUTFILE
  !define OUTFILE "..\dist\windows\BCompareKeygen-Setup-1.1.0.exe"
!endif
!ifndef ICON
  !define ICON "..\assets\icon.ico"
!endif

Name "BCompare Keygen"
OutFile "${OUTFILE}"
Unicode true
InstallDir "$PROGRAMFILES64\BCompare Keygen"
InstallDirRegKey HKLM "Software\BCompareKeygen" "InstallDir"
RequestExecutionLevel admin
Icon "${ICON}"
UninstallIcon "${ICON}"
BrandingText "BCompare Keygen · by Cob"

Page directory
Page instfiles
UninstPage uninstConfirm
UninstPage instfiles

Section "Install"
  SetOutPath "$INSTDIR"
  File "/oname=BCompareKeygen.exe" "${SRCEXE}"
  File "/oname=BCompareKeygen-cli.exe" "${SRCCLI}"
  File "/oname=icon.ico" "${ICON}"

  WriteRegStr HKLM "Software\BCompareKeygen" "InstallDir" "$INSTDIR"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\BCompareKeygen" "DisplayName" "BCompare Keygen"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\BCompareKeygen" "DisplayVersion" "${APPVER}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\BCompareKeygen" "Publisher" "Cob"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\BCompareKeygen" "DisplayIcon" "$INSTDIR\icon.ico"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\BCompareKeygen" "UninstallString" "$INSTDIR\uninstall.exe"

  CreateDirectory "$SMPROGRAMS\BCompare Keygen"
  CreateShortcut "$SMPROGRAMS\BCompare Keygen\BCompare Keygen.lnk" "$INSTDIR\BCompareKeygen.exe" "" "$INSTDIR\icon.ico"
  CreateShortcut "$DESKTOP\BCompare Keygen.lnk" "$INSTDIR\BCompareKeygen.exe" "" "$INSTDIR\icon.ico"

  WriteUninstaller "$INSTDIR\uninstall.exe"
SectionEnd

Section "Uninstall"
  Delete "$INSTDIR\BCompareKeygen.exe"
  Delete "$INSTDIR\BCompareKeygen-cli.exe"
  Delete "$INSTDIR\icon.ico"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"
  Delete "$SMPROGRAMS\BCompare Keygen\BCompare Keygen.lnk"
  RMDir "$SMPROGRAMS\BCompare Keygen"
  Delete "$DESKTOP\BCompare Keygen.lnk"
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\BCompareKeygen"
  DeleteRegKey HKLM "Software\BCompareKeygen"
SectionEnd
