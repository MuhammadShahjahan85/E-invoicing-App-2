; Inno Setup 6 script — E-Invoicing Suite PK (Windows x64)
;
; 1. Build the binary:   sh scripts/build-release.sh   (creates dist\windows-amd64\einvoice.exe)
; 2. Compile installer:  ISCC.exe /DAppVersion=1.0.0 packaging\windows\einvoice-suite.iss
;    Output:             dist\EInvoicingSuitePK-Setup-<version>.exe
;
; The installer registers the Windows service "EInvoicingSuitePK", which stores its data in
; %ProgramData%\EInvoicingSuitePK (database, master.key, TLS certificate, logs, backups).
; Uninstalling never deletes that folder: records must be kept for six years (rule 150S).

#ifndef AppVersion
  #define AppVersion "1.0.0"
#endif
#define AppName "E-Invoicing Suite PK"
#define AppPublisher "Your Company (Pvt) Ltd"
#define AppExe "einvoice.exe"
#define ServiceName "EInvoicingSuitePK"
#define Port "8443"

[Setup]
AppId={{6F1C2B9E-4D3A-4E7B-9C51-2A8D7E3F4B10}
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher={#AppPublisher}
DefaultDirName={autopf}\{#AppName}
DefaultGroupName={#AppName}
DisableProgramGroupPage=yes
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
PrivilegesRequired=admin
OutputDir=..\..\dist
OutputBaseFilename=EInvoicingSuitePK-Setup-{#AppVersion}
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
function ServiceExists(): Boolean;
var
  ResultCode: Integer;
begin
  Result := Exec(ExpandConstant('{sys}\sc.exe'), 'query {#ServiceName}', '', SW_HIDE, ewWaitUntilTerminated, ResultCode) and (ResultCode = 0);
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
  Result := '';
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usPostUninstall then
    MsgBox('The data folder ' + ExpandConstant('{commonappdata}') + '\{#ServiceName} was NOT deleted.' + #13#10 + #13#10 +
      'It contains the invoice database, master.key, certificates and backups. Keep it (records must be retained for six years) ' +
      'or copy it to safe storage before removing it manually.', mbInformation, MB_OK);
end;
