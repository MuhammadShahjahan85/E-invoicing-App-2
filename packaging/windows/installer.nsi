; Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
; Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.
;
; Windows installer (NSIS 3, 64-bit Windows 10 or 11).
;
; scripts/build-release.sh builds it after the programs, or by hand from the
; repository root:
;   makensis -DVERSION=1.2.3 -DNUMVER=1.2.3.0 packaging/windows/installer.nsi
; Output: dist/VeridianEInvoicingPakistan-Setup-<VERSION>.exe
;
; Setup installs the server (einvoice.exe) as the Windows service
; "VeridianEInvoicingPakistan", which keeps running in the background, and the
; desktop launcher (VeridianEInvoicing.exe) that the shortcuts open: it shows
; the app in its own window, without a command window. Data lives in
; %ProgramData%\VeridianEInvoicingPakistan (database, master.key, certificates,
; logs and backups) and is never deleted by the uninstaller: sales tax records
; must be kept for six years.
;
; Upgrades: setup stops the service, replaces the programs and starts it again;
; it also removes an installation made with the earlier Inno Setup installer
; and the service of builds released as "Veridian E-invoicing PK" (whose data
; folder %ProgramData%\VeridianEInvoicingPK stays in use).
;
; Silent install: Setup.exe /S [/D=C:\Program Files\Veridian E-invoicing Pakistan]
; Silent uninstall: "C:\Program Files\Veridian E-invoicing Pakistan\Uninstall.exe" /S

Unicode true
ManifestDPIAware true
ManifestSupportedOS all
RequestExecutionLevel admin
SetCompressor /SOLID lzma
SetCompressorDictSize 32

!ifndef VERSION
  !define VERSION "1.0.0"
!endif
!ifndef NUMVER
  !define NUMVER "1.0.0.0"
!endif
!define ROOT "..\.."
!ifndef DIST
  !define DIST "${ROOT}\dist"
!endif

!define APPNAME "Veridian E-invoicing Pakistan"
!define COMPANY "Veridian Partners Consultancy Private Limited"
!define SUPPORT "muhammadshahjahan.audit@gmail.com"
!define SERVICE "VeridianEInvoicingPakistan"
!define LEGACYSERVICE "VeridianEInvoicingPK"
!define SERVER "einvoice.exe"
!define LAUNCHER "VeridianEInvoicing.exe"
!define GUIDE "Veridian-E-invoicing-Pakistan-User-Guide.pdf"
!define UNINSTKEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${SERVICE}"
; The earlier Inno Setup installer (AppId).
!define INNOKEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\{6F1C2B9E-4D3A-4E7B-9C51-2A8D7E3F4B10}_is1"

Name "${APPNAME}"
Caption "${APPNAME} ${VERSION} Setup"
UninstallCaption "Remove ${APPNAME}"
OutFile "${DIST}\VeridianEInvoicingPakistan-Setup-${VERSION}.exe"
InstallDir "$PROGRAMFILES64\${APPNAME}"
BrandingText "${COMPANY}"
ShowInstDetails hide
ShowUninstDetails hide

VIProductVersion "${NUMVER}"
VIFileVersion "${NUMVER}"
VIAddVersionKey /LANG=1033 "ProductName" "${APPNAME}"
VIAddVersionKey /LANG=1033 "CompanyName" "${COMPANY}"
VIAddVersionKey /LANG=1033 "LegalCopyright" "© 2026 ${COMPANY}. All rights reserved."
VIAddVersionKey /LANG=1033 "FileDescription" "${APPNAME} Setup"
VIAddVersionKey /LANG=1033 "FileVersion" "${VERSION}"
VIAddVersionKey /LANG=1033 "ProductVersion" "${VERSION}"
VIAddVersionKey /LANG=1033 "Comments" "Support: ${SUPPORT}"

!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "x64.nsh"
!include "WinVer.nsh"
!include "FileFunc.nsh"

Var DataDir

; ---------------------------------------------------------------- Appearance

!define MUI_ICON "assets\veridian.ico"
!define MUI_UNICON "assets\veridian.ico"
!define MUI_WELCOMEFINISHPAGE_BITMAP "assets\wizard.bmp"
!define MUI_UNWELCOMEFINISHPAGE_BITMAP "assets\wizard.bmp"
!define MUI_HEADERIMAGE
!define MUI_HEADERIMAGE_RIGHT
!define MUI_HEADERIMAGE_BITMAP "assets\header.bmp"
!define MUI_HEADERIMAGE_UNBITMAP "assets\header.bmp"
!define MUI_ABORTWARNING
!define MUI_UNABORTWARNING
!define MUI_COMPONENTSPAGE_SMALLDESC
!define MUI_FINISHPAGE_NOAUTOCLOSE
!define MUI_UNFINISHPAGE_NOAUTOCLOSE

; ---------------------------------------------------------------- Pages

!define MUI_WELCOMEPAGE_TITLE "Welcome to ${APPNAME}"
!define MUI_WELCOMEPAGE_TEXT "Setup will install ${APPNAME} ${VERSION} on this computer.$\r$\n$\r$\nIssue sales tax invoices, report them to FBR Digital Invoicing and keep your sales register, Annexure-C and sales tax return in step, all in one place.$\r$\n$\r$\nThe program runs quietly in the background. After setup, open it from its desktop or Start menu icon: no command window is needed.$\r$\n$\r$\nClick Next to continue."
!insertmacro MUI_PAGE_WELCOME

!define MUI_PAGE_HEADER_TEXT "Licence Agreement"
!define MUI_PAGE_HEADER_SUBTEXT "Please read the licence before you install."
!define MUI_LICENSEPAGE_TEXT_TOP "Scroll down to read all of it."
!define MUI_LICENSEPAGE_TEXT_BOTTOM "If you accept the licence, click I Agree to continue. You must accept it to install ${APPNAME}."
!insertmacro MUI_PAGE_LICENSE "${ROOT}\LICENSE"

!define MUI_PAGE_HEADER_TEXT "Choose Components"
!define MUI_PAGE_HEADER_SUBTEXT "Choose what to set up on this computer."
!define MUI_COMPONENTSPAGE_TEXT_TOP "The recommended choices suit most offices."
!insertmacro MUI_PAGE_COMPONENTS

!define MUI_PAGE_HEADER_TEXT "Choose Install Location"
!define MUI_PAGE_HEADER_SUBTEXT "Choose the folder for the program files."
!define MUI_DIRECTORYPAGE_TEXT_TOP "Setup will install the program in the folder below. Your data is kept separately, in $DataDir. Click Install to start."
!insertmacro MUI_PAGE_DIRECTORY

!define MUI_PAGE_HEADER_TEXT "Installing"
!define MUI_PAGE_HEADER_SUBTEXT "Please wait while ${APPNAME} is set up."
!define MUI_INSTFILESPAGE_FINISHHEADER_TEXT "Installation Complete"
!define MUI_INSTFILESPAGE_FINISHHEADER_SUBTEXT "${APPNAME} is installed and running."
!define MUI_INSTFILESPAGE_ABORTHEADER_TEXT "Installation Stopped"
!define MUI_INSTFILESPAGE_ABORTHEADER_SUBTEXT "Setup did not finish. Click Show details for the reason."
!insertmacro MUI_PAGE_INSTFILES

!define MUI_FINISHPAGE_TITLE "${APPNAME} is ready"
!define MUI_FINISHPAGE_TEXT "${APPNAME} is installed and running in the background.$\r$\n$\r$\nOpen it any time from its desktop or Start menu icon. A short wizard sets up your business the first time."
!define MUI_FINISHPAGE_RUN
!define MUI_FINISHPAGE_RUN_TEXT "Open ${APPNAME} now"
!define MUI_FINISHPAGE_RUN_FUNCTION OpenApp
!define MUI_FINISHPAGE_LINK "Support: ${SUPPORT}"
!define MUI_FINISHPAGE_LINK_LOCATION "mailto:${SUPPORT}"
!insertmacro MUI_PAGE_FINISH

!define MUI_PAGE_HEADER_TEXT "Remove ${APPNAME}"
!define MUI_PAGE_HEADER_SUBTEXT "Remove the program from this computer."
!define MUI_UNCONFIRMPAGE_TEXT_TOP "The program and its background service will be removed. Your data folder, with your invoices, settings and backups, is kept."
!insertmacro MUI_UNPAGE_CONFIRM
!define MUI_PAGE_HEADER_TEXT "Removing"
!define MUI_PAGE_HEADER_SUBTEXT "Please wait while ${APPNAME} is removed."
!define MUI_INSTFILESPAGE_FINISHHEADER_TEXT "Removal Complete"
!define MUI_INSTFILESPAGE_FINISHHEADER_SUBTEXT "The program was removed. Your data was kept."
!insertmacro MUI_UNPAGE_INSTFILES
!define MUI_FINISHPAGE_TITLE "${APPNAME} was removed"
!define MUI_FINISHPAGE_TEXT "The program was removed from this computer.$\r$\n$\r$\nYour data folder was kept:$\r$\n$DataDir$\r$\n$\r$\nIt holds your invoices, settings, keys and backups. Sales tax records must be kept for six years, so keep it or copy it to safe storage. Installing again picks it up automatically."
!define MUI_FINISHPAGE_TEXT_LARGE
!insertmacro MUI_UNPAGE_FINISH

!insertmacro MUI_LANGUAGE "English"

; ---------------------------------------------------------------- Helpers

; FindDataDir mirrors config.DefaultDataDir: the current folder, or that of
; an installation released as "Veridian E-invoicing PK".
!macro FIND_DATA_DIR un
Function ${un}FindDataDir
  StrCpy $DataDir "$APPDATA\${SERVICE}"
  ${IfNot} ${FileExists} "$DataDir\*.*"
  ${AndIf} ${FileExists} "$APPDATA\${LEGACYSERVICE}\config.json"
    StrCpy $DataDir "$APPDATA\${LEGACYSERVICE}"
  ${EndIf}
FunctionEnd
!macroend
!insertmacro FIND_DATA_DIR ""
!insertmacro FIND_DATA_DIR "un."

; StopService stops the service and waits (up to 30 seconds) until
; einvoice.exe has exited, so that it can be replaced or removed.
!macro STOP_SERVICE un
Function ${un}StopService
  nsExec::Exec '"$SYSDIR\sc.exe" stop ${SERVICE}'
  Pop $0
  ${IfNot} ${FileExists} "$INSTDIR\${SERVER}"
    Return
  ${EndIf}
  StrCpy $1 0
  ${Do}
    ClearErrors
    FileOpen $2 "$INSTDIR\${SERVER}" a
    ${IfNot} ${Errors}
      FileClose $2
      ${Break}
    ${EndIf}
    IntOp $1 $1 + 1
    ${If} $1 >= 60
      ${Break}
    ${EndIf}
    Sleep 500
  ${Loop}
FunctionEnd
!macroend
!insertmacro STOP_SERVICE ""
!insertmacro STOP_SERVICE "un."

; Step runs a command of einvoice.exe, showing its output under Show details.
; $0 is its exit code (0 = success).
!macro STEP text args
  DetailPrint "${text}"
  nsExec::ExecToLog '"$INSTDIR\${SERVER}" ${args}'
  Pop $0
!macroend

Function .onInit
  ${IfNot} ${RunningX64}
  ${OrIfNot} ${AtLeastWin10}
    MessageBox MB_ICONSTOP|MB_OK "${APPNAME} needs 64-bit Windows 10 or Windows 11." /SD IDOK
    Abort
  ${EndIf}
  SetRegView 64
  SetShellVarContext all
  System::Call 'kernel32::CreateMutex(p 0, i 0, t "${SERVICE}Setup") p .r1 ?e'
  Pop $0
  ${If} $0 = 183
    MessageBox MB_ICONINFORMATION|MB_OK "Setup is already running." /SD IDOK
    Abort
  ${EndIf}
  ; Upgrade: keep the folder of the installed copy (unless /D= chose one).
  ${If} $INSTDIR == "$PROGRAMFILES64\${APPNAME}"
    ReadRegStr $0 HKLM "${UNINSTKEY}" "InstallLocation"
    ${If} $0 != ""
      StrCpy $INSTDIR $0
    ${EndIf}
  ${EndIf}
  Call FindDataDir
FunctionEnd

Function un.onInit
  SetRegView 64
  SetShellVarContext all
  Call un.FindDataDir
FunctionEnd

; OpenApp runs the launcher as the signed-in user rather than as administrator.
Function OpenApp
  Exec '"$WINDIR\explorer.exe" "$INSTDIR\${LAUNCHER}"'
FunctionEnd

; ---------------------------------------------------------------- Sections

Section "${APPNAME}" SecApp
  SectionIn RO
  SetDetailsPrint both

  ; An installed copy is replaced: stop it first.
  DetailPrint "Stopping the running copy, if any"
  Call StopService

  ; Builds released as "Veridian E-invoicing PK" ran a service of that name.
  nsExec::Exec '"$SYSDIR\sc.exe" query ${LEGACYSERVICE}'
  Pop $0
  ${If} $0 = 0
    DetailPrint "Removing the service of Veridian E-invoicing PK (its data is kept)"
    nsExec::Exec '"$SYSDIR\sc.exe" stop ${LEGACYSERVICE}'
    Pop $0
    Sleep 3000
    nsExec::Exec '"$SYSDIR\sc.exe" delete ${LEGACYSERVICE}'
    Pop $0
  ${EndIf}

  ; Installations made with the earlier Inno Setup installer: remove them
  ; quietly (the data folder is kept).
  ReadRegStr $1 HKLM "${INNOKEY}" "UninstallString"
  ${If} $1 != ""
    DetailPrint "Removing the earlier installation (its data is kept)"
    ExecWait '$1 /VERYSILENT /SUPPRESSMSGBOXES /NORESTART' $0
  ${EndIf}

  DetailPrint "Copying the program files"
  SetOutPath "$INSTDIR"
  File "${DIST}\windows-amd64\${SERVER}"
  File "${DIST}\windows-amd64\${LAUNCHER}"
  File /oname=LICENSE.txt "${ROOT}\LICENSE"
  File "${ROOT}\THIRD-PARTY-NOTICES.txt"
  File "${ROOT}\README.md"
  SetOutPath "$INSTDIR\docs"
  File "${ROOT}\docs\*.md"
  !if /FileExists "${ROOT}\docs\${GUIDE}"
    File "${ROOT}\docs\${GUIDE}"
  !endif
  SetOutPath "$INSTDIR"

  !insertmacro STEP "Preparing the data folder $DataDir" 'tls init --data "$DataDir"'
  ${If} $0 != 0
    MessageBox MB_ICONSTOP|MB_OK "The data folder $DataDir could not be prepared.$\r$\n$\r$\nClick Show details for the reason, or contact ${SUPPORT}." /SD IDOK
    Abort "The data folder could not be prepared."
  ${EndIf}

  ; Only Windows itself (SYSTEM) and administrators may open the database,
  ; keys, logs and backups. Users may read the settings and the certificate
  ; authority, which the desktop launcher and browsers use.
  DetailPrint "Protecting the data folder"
  nsExec::Exec '"$SYSDIR\icacls.exe" "$DataDir" /inheritance:r /grant:r *S-1-5-18:(OI)(CI)F *S-1-5-32-544:(OI)(CI)F'
  Pop $0
  nsExec::Exec '"$SYSDIR\icacls.exe" "$DataDir\config.json" /grant:r *S-1-5-32-545:(R)'
  Pop $0
  nsExec::Exec '"$SYSDIR\icacls.exe" "$DataDir\tls\ca.pem" /grant:r *S-1-5-32-545:(R)'
  Pop $0

  !insertmacro STEP "Registering the background service" 'service install --data "$DataDir"'
  ${If} $0 != 0
    MessageBox MB_ICONSTOP|MB_OK "The background service could not be registered.$\r$\n$\r$\nClick Show details for the reason, or contact ${SUPPORT}." /SD IDOK
    Abort "The background service could not be registered."
  ${EndIf}

  WriteUninstaller "$INSTDIR\Uninstall.exe"
  WriteRegStr HKLM "${UNINSTKEY}" "DisplayName" "${APPNAME}"
  WriteRegStr HKLM "${UNINSTKEY}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKLM "${UNINSTKEY}" "Publisher" "${COMPANY}"
  WriteRegStr HKLM "${UNINSTKEY}" "DisplayIcon" "$INSTDIR\${LAUNCHER},0"
  WriteRegStr HKLM "${UNINSTKEY}" "InstallLocation" "$INSTDIR"
  WriteRegStr HKLM "${UNINSTKEY}" "UninstallString" '"$INSTDIR\Uninstall.exe"'
  WriteRegStr HKLM "${UNINSTKEY}" "QuietUninstallString" '"$INSTDIR\Uninstall.exe" /S'
  WriteRegStr HKLM "${UNINSTKEY}" "HelpLink" "mailto:${SUPPORT}"
  WriteRegStr HKLM "${UNINSTKEY}" "Comments" "FBR Digital Invoicing"
  WriteRegDWORD HKLM "${UNINSTKEY}" "NoModify" 1
  WriteRegDWORD HKLM "${UNINSTKEY}" "NoRepair" 1
  SetDetailsPrint none
  ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
  SetDetailsPrint both
  IntFmt $0 "0x%08X" $0
  WriteRegDWORD HKLM "${UNINSTKEY}" "EstimatedSize" $0
SectionEnd

Section "Trust the security certificate" SecTrust
  !insertmacro STEP "Trusting the security certificate on this computer" 'tls trust --data "$DataDir"'
SectionEnd

Section "Desktop icon" SecDesktop
  CreateShortcut "$DESKTOP\${APPNAME}.lnk" "$INSTDIR\${LAUNCHER}" "" "$INSTDIR\${LAUNCHER}" 0 SW_SHOWNORMAL "" "Open ${APPNAME}"
SectionEnd

Section "Office network access" SecFirewall
  !insertmacro STEP "Allowing computers on the office network (Windows Firewall)" 'firewall allow --data "$DataDir"'
SectionEnd

Section "-Finish"
  ${IfNot} ${SectionIsSelected} ${SecFirewall}
    nsExec::Exec '"$INSTDIR\${SERVER}" firewall remove'
    Pop $0
  ${EndIf}
  ${IfNot} ${SectionIsSelected} ${SecDesktop}
    Delete "$DESKTOP\${APPNAME}.lnk"
  ${EndIf}

  CreateDirectory "$SMPROGRAMS\${APPNAME}"
  CreateShortcut "$SMPROGRAMS\${APPNAME}\${APPNAME}.lnk" "$INSTDIR\${LAUNCHER}" "" "$INSTDIR\${LAUNCHER}" 0 SW_SHOWNORMAL "" "Open ${APPNAME}"
  !if /FileExists "${ROOT}\docs\${GUIDE}"
    CreateShortcut "$SMPROGRAMS\${APPNAME}\User guide.lnk" "$INSTDIR\docs\${GUIDE}"
  !endif
  CreateShortcut "$SMPROGRAMS\${APPNAME}\Help and error codes.lnk" "$INSTDIR\${LAUNCHER}" "help" "$INSTDIR\${LAUNCHER}" 0 SW_SHOWNORMAL "" "FBR error codes and how to fix them"

  !insertmacro STEP "Starting the background service" "service start"
  ${If} $0 != 0
    MessageBox MB_ICONEXCLAMATION|MB_OK "The background service did not start.$\r$\n$\r$\nThe desktop icon tries again. If it keeps failing, the log files in $DataDir\logs show why." /SD IDOK
  ${EndIf}
SectionEnd

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
  !insertmacro MUI_DESCRIPTION_TEXT ${SecApp} "The program, the background service that keeps it running, and the guides."
  !insertmacro MUI_DESCRIPTION_TEXT ${SecTrust} "Lets Microsoft Edge and Google Chrome on this computer open the app securely, without a certificate warning. Recommended."
  !insertmacro MUI_DESCRIPTION_TEXT ${SecDesktop} "An icon on the desktop that opens the app in its own window."
  !insertmacro MUI_DESCRIPTION_TEXT ${SecFirewall} "Lets other computers and phones on your office network use the app (Windows Firewall; local network only)."
!insertmacro MUI_FUNCTION_DESCRIPTION_END

; ---------------------------------------------------------------- Uninstall

Section "Uninstall"
  SetDetailsPrint both
  DetailPrint "Stopping the background service"
  Call un.StopService
  !insertmacro STEP "Removing the background service" "service uninstall"
  !insertmacro STEP "Removing the security certificate" 'tls untrust --data "$DataDir"'
  !insertmacro STEP "Removing the Windows Firewall rule" "firewall remove"

  Delete "$DESKTOP\${APPNAME}.lnk"
  RMDir /r "$SMPROGRAMS\${APPNAME}"

  Delete "$INSTDIR\${SERVER}"
  Delete "$INSTDIR\${LAUNCHER}"
  Delete "$INSTDIR\LICENSE.txt"
  Delete "$INSTDIR\THIRD-PARTY-NOTICES.txt"
  Delete "$INSTDIR\README.md"
  RMDir /r "$INSTDIR\docs"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"
  DeleteRegKey HKLM "${UNINSTKEY}"
  DetailPrint "Your data folder was kept: $DataDir"
SectionEnd
