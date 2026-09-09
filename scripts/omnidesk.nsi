!include "MUI2.nsh"
!include "FileFunc.nsh"

Name "OmniDesk"
OutFile "../dist/OmniDesk-Setup-0.2.0-x64.exe"
InstallDir "$LOCALAPPDATA\Programs\OmniDesk"
InstallDirRegKey HKCU "Software\OmniDesk" "InstallDir"
RequestExecutionLevel user

; Interface Settings
!define MUI_ABORTWARNING
!define MUI_ICON "../assets/omnidesk.ico"
!define MUI_UNICON "../assets/omnidesk.ico"

; Setup Pages
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_RUN "$INSTDIR\omnidesk.exe"
!define MUI_FINISHPAGE_RUN_TEXT "Iniciar o OmniDesk agora"
!insertmacro MUI_PAGE_FINISH

; Uninstaller Pages
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_UNPAGE_FINISH

; Language
!insertmacro MUI_LANGUAGE "PortugueseBR"

Section "MainSection" SEC01
  SetOutPath "$INSTDIR"
  SetOverwrite on
  
  File "../dist/omnidesk.exe"
  File "../assets/omnidesk.ico"
  
  ; Write uninstaller
  WriteUninstaller "$INSTDIR\uninstall.exe"
  
  ; Create Shortcuts in Start Menu and Desktop
  CreateDirectory "$SMPROGRAMS\OmniDesk"
  CreateShortcut "$SMPROGRAMS\OmniDesk\OmniDesk.lnk" "$INSTDIR\omnidesk.exe" "" "$INSTDIR\omnidesk.ico"
  CreateShortcut "$SMPROGRAMS\OmniDesk\Desinstalar.lnk" "$INSTDIR\uninstall.exe"
  CreateShortcut "$DESKTOP\OmniDesk.lnk" "$INSTDIR\omnidesk.exe" "" "$INSTDIR\omnidesk.ico"
  
  ; Autostart via Registry Run Key
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "OmniDesk" '"$INSTDIR\omnidesk.exe" daemon'
  
  ; Register in Windows "Add/Remove Programs"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\OmniDesk" "DisplayName" "OmniDesk"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\OmniDesk" "UninstallString" '"$INSTDIR\uninstall.exe"'
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\OmniDesk" "DisplayIcon" "$INSTDIR\omnidesk.ico"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\OmniDesk" "DisplayVersion" "0.2.0"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\OmniDesk" "Publisher" "OmniDesk"
SectionEnd

Section "Uninstall"
  ; Terminate running instance if active
  nsExec::Exec 'taskkill /F /IM omnidesk.exe'
  
  ; Remove Registry entries
  DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "OmniDesk"
  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\OmniDesk"
  DeleteRegKey HKCU "Software\OmniDesk"
  
  ; Remove Shortcuts
  Delete "$DESKTOP\OmniDesk.lnk"
  Delete "$SMPROGRAMS\OmniDesk\OmniDesk.lnk"
  Delete "$SMPROGRAMS\OmniDesk\Desinstalar.lnk"
  RMDir "$SMPROGRAMS\OmniDesk"
  
  ; Remove Files
  Delete "$INSTDIR\omnidesk.exe"
  Delete "$INSTDIR\omnidesk.ico"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"
SectionEnd
