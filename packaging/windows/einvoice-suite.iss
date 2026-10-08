; Inno Setup 6 script — Veridian E-invoicing Pakistan (Windows x64)
;
; 1. Build the binary:   sh scripts/build-release.sh   (creates dist\windows-amd64\einvoice.exe)
; 2. Compile installer:  ISCC.exe /DAppVersion=1.0.0 packaging\windows\einvoice-suite.iss
;    Output:             dist\VeridianEInvoicingPakistan-Setup-<version>.exe
;
; The installer registers the Windows service "VeridianEInvoicingPakistan", which stores its data in
; %ProgramData%\VeridianEInvoicingPakistan (database, master.key, TLS certificate, logs, backups).
; Uninstalling never deletes that folder: records must be kept for six years (rule 150S).
; Upgrading a computer that runs a build released as "Veridian E-invoicing PK" removes the old
; service "VeridianEInvoicingPK"; the program keeps using its data folder
; %ProgramData%\VeridianEInvoicingPK, so no invoices, settings or keys are lost.

#ifndef AppVersion
  #define AppVersion "1.0.0"
#endif
#define AppName "Veridian E-invoicing Pakistan"
#define AppPublisher "Veridian Partners Consultancy Private Limited"
#define AppExe "einvoice.exe"
#define ServiceName "VeridianEInvoicingPakistan"
#define LegacyServiceName "VeridianEInvoicingPK"
#define LegacyAppName "Veridian E-invoicing PK"
#define Port "8443"

[Setup]
AppId={{6F1C2B9E-4D3A-4E7B-9C51-2A8D7E3F4B10}
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher={#AppPublisher}
AppCopyright=Copyright (C) 2026 {#AppPublisher}. All rights reserved.
AppSupportURL=mailto:muhammadshahjahan.audit@gmail.com
VersionInfoCompany={#AppPublisher}
VersionInfoCopyright=Copyright (C) 2026 {#AppPublisher}
VersionInfoProductName={#AppName}
VersionInfoVersion={#AppVersion}
LicenseFile=..\..\LICENSE
DefaultDirName={autopf}\{#AppName}
DefaultGroupName={#AppName}
DisableProgramGroupPage=yes
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
PrivilegesRequired=admin
OutputDir=..\..\dist
OutputBaseFilename=VeridianEInvoicingPakistan-Setup-{#AppVersion}
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
UninstallDisplayIcon={app}\{#AppExe}
SetupLogging=yes

[Tasks]
Name: "firewall"; Description: "Allow other computers on the office network to use the system (Windows Firewall, TCP port {#Port})"
Name: "desktopicon"; Description: "Create a desktop shortcut"; Flags: unchecked

[Files]
Source: "..\..\dist\windows-amd64\einvoice.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\LICENSE"; DestDir: "{app}"; DestName: "LICENSE.txt"; Flags: ignoreversion
Source: "..\..\THIRD-PARTY-NOTICES.txt"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\docs\*.md"; DestDir: "{app}\docs"; Flags: ignoreversion

[Icons]
Name: "{group}\Open {#AppName}"; Filename: "https://localhost:{#Port}/"
Name: "{group}\Documentation"; Filename: "{app}\docs"
Name: "{group}\Uninstall {#AppName}"; Filename: "{uninstallexe}"
Name: "{commondesktop}\{#AppName}"; Filename: "https://localhost:{#Port}/"; Tasks: desktopicon

[Run]
Filename: "{app}\{#AppExe}"; Parameters: "service install"; Flags: runhidden waituntilterminated; StatusMsg: "Installing the Windows service..."; Check: not ServiceExists
Filename: "{sys}\netsh.exe"; Parameters: "advfirewall firewall add rule name=""{#AppName}"" dir=in action=allow protocol=TCP localport={#Port}"; Flags: runhidden waituntilterminated; Tasks: firewall
Filename: "{app}\{#AppExe}"; Parameters: "service start"; Flags: runhidden waituntilterminated; StatusMsg: "Starting the service..."
Filename: "https://localhost:{#Port}/"; Description: "Open {#AppName} now (first-run setup wizard)"; Flags: postinstall shellexec nowait skipifsilent

[UninstallRun]
Filename: "{app}\{#AppExe}"; Parameters: "service stop"; Flags: runhidden waituntilterminated; RunOnceId: "StopService"
Filename: "{app}\{#AppExe}"; Parameters: "service uninstall"; Flags: runhidden waituntilterminated; RunOnceId: "RemoveService"
Filename: "{sys}\netsh.exe"; Parameters: "advfirewall firewall delete rule name=""{#AppName}"""; Flags: runhidden waituntilterminated; RunOnceId: "RemoveFirewallRule"

[Code]
function NamedServiceExists(Name: String): Boolean;
var
  ResultCode: Integer;
begin
  Result := Exec(ExpandConstant('{sys}\sc.exe'), 'query ' + Name, '', SW_HIDE, ewWaitUntilTerminated, ResultCode) and (ResultCode = 0);
end;

function ServiceExists(): Boolean;
begin
  Result := NamedServiceExists('{#ServiceName}');
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
var
  ResultCode: Integer;
begin
  // Upgrade: stop the running service so that einvoice.exe can be replaced.
  if ServiceExists() then
  begin
    Exec(ExpandConstant('{sys}\sc.exe'), 'stop {#ServiceName}', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
    Sleep(5000);
  end;
  // Upgrade from a build released as "{#LegacyAppName}": remove its service and
  // firewall rule. The new service is installed afterwards and keeps using the
  // existing data folder.
  if NamedServiceExists('{#LegacyServiceName}') then
  begin
    Exec(ExpandConstant('{sys}\sc.exe'), 'stop {#LegacyServiceName}', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
    Sleep(5000);
    Exec(ExpandConstant('{sys}\sc.exe'), 'delete {#LegacyServiceName}', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
    Exec(ExpandConstant('{sys}\netsh.exe'), 'advfirewall firewall delete rule name="{#LegacyAppName}"', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  end;
  Result := '';
end;

// DataFolder mirrors config.DefaultDataDir: the current folder, or the folder of
// an installation upgraded from "{#LegacyAppName}".
function DataFolder(): String;
begin
  Result := ExpandConstant('{commonappdata}') + '\{#ServiceName}';
  if (not DirExists(Result)) and FileExists(ExpandConstant('{commonappdata}') + '\{#LegacyServiceName}\config.json') then
    Result := ExpandConstant('{commonappdata}') + '\{#LegacyServiceName}';
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usPostUninstall then
    MsgBox('The data folder ' + DataFolder() + ' was NOT deleted.' + #13#10 + #13#10 +
      'It contains the invoice database, master.key, certificates and backups. Keep it (records must be retained for six years) ' +
      'or copy it to safe storage before removing it manually.', mbInformation, MB_OK);
end;
