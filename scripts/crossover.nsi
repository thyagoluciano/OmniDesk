!include "MUI2.nsh"
!include "FileFunc.nsh"

Name "Crossover"
OutFile "../dist/Crossover-Setup-0.1.0-x64.exe"
InstallDir "$LOCALAPPDATA\Programs\Crossover"
InstallDirRegKey HKCU "Software\Crossover" "InstallDir"
RequestExecutionLevel user

; Interface Settings
!define MUI_ABORTWARNING
!define MUI_ICON "../assets/crossover.ico"
!define MUI_UNICON "../assets/crossover.ico"

; Setup Pages
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_RUN "$INSTDIR\crossover.exe"
!define MUI_FINISHPAGE_RUN_TEXT "Iniciar o Crossover agora"
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
  
  File "../dist/crossover.exe"
  File "../assets/crossover.ico"
  
  ; Write uninstaller
  WriteUninstaller "$INSTDIR\uninstall.exe"
  
  ; Create Shortcuts in Start Menu and Desktop
  CreateDirectory "$SMPROGRAMS\Crossover"
  CreateShortcut "$SMPROGRAMS\Crossover\Crossover.lnk" "$INSTDIR\crossover.exe" "" "$INSTDIR\crossover.ico"
  CreateShortcut "$SMPROGRAMS\Crossover\Desinstalar.lnk" "$INSTDIR\uninstall.exe"
  CreateShortcut "$DESKTOP\Crossover.lnk" "$INSTDIR\crossover.exe" "" "$INSTDIR\crossover.ico"
  
  ; Autostart via Registry Run Key
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "Crossover" '"$INSTDIR\crossover.exe" daemon'
  
  ; Register in Windows "Add/Remove Programs"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Crossover" "DisplayName" "Crossover"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Crossover" "UninstallString" '"$INSTDIR\uninstall.exe"'
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Crossover" "DisplayIcon" "$INSTDIR\crossover.ico"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Crossover" "DisplayVersion" "0.1.0"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Crossover" "Publisher" "Crossover"
SectionEnd

Section "Uninstall"
  ; Terminate running instance if active
  nsExec::Exec 'taskkill /F /IM crossover.exe'
  
  ; Remove Registry entries
  DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "Crossover"
  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Crossover"
  DeleteRegKey HKCU "Software\Crossover"
  
  ; Remove Shortcuts
  Delete "$DESKTOP\Crossover.lnk"
  Delete "$SMPROGRAMS\Crossover\Crossover.lnk"
  Delete "$SMPROGRAMS\Crossover\Desinstalar.lnk"
  RMDir "$SMPROGRAMS\Crossover"
  
  ; Remove Files
  Delete "$INSTDIR\crossover.exe"
  Delete "$INSTDIR\crossover.ico"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"
SectionEnd
