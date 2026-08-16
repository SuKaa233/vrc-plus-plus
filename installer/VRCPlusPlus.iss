#ifndef AppVersion
  #error AppVersion must be provided by scripts/package.ps1
#endif
#ifndef SourceExe
  #error SourceExe must be provided by scripts/package.ps1
#endif
#ifndef AppNumericVersion
  #error AppNumericVersion must be provided by scripts/package.ps1
#endif
#ifndef OutputDirectory
  #error OutputDirectory must be provided by scripts/package.ps1
#endif

#define AppName "VRC++"
#define AppExeName "vrc-plus-plus.exe"
#define AppId "{{9D6584CB-59EC-4EC2-80EC-E95B14BD1A5D}"

[Setup]
AppId={#AppId}
AppName={#AppName}
AppVersion={#AppVersion}
AppVerName={#AppName} {#AppVersion}
AppPublisher=VRC++
AppSupportURL=mailto:2579362548@qq.com
DefaultDirName={localappdata}\Programs\VRC++
DefaultGroupName=VRC++
DisableDirPage=no
UsePreviousAppDir=yes
AlwaysShowDirOnReadyPage=yes
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
OutputDir={#OutputDirectory}
OutputBaseFilename=VRC++-Setup-{#AppVersion}
SetupIconFile=..\apps\gateway\internal\tray\icon.ico
UninstallDisplayIcon={app}\vrc-plus-plus.ico
Compression=lzma2/ultra64
SolidCompression=yes
WizardStyle=modern
WizardSizePercent=110
CloseApplications=yes
CloseApplicationsFilter={#AppExeName}
RestartApplications=no
VersionInfoVersion={#AppNumericVersion}
VersionInfoProductName={#AppName}
VersionInfoDescription=VRC++ 安装程序
VersionInfoCompany=VRC++
VersionInfoCopyright=VRC++ contributors
MinVersion=10.0.17763
ChangesAssociations=no
ChangesEnvironment=no

[Languages]
Name: "chinesesimplified"; MessagesFile: "compiler:Languages\ChineseSimplified.isl"
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "startup"; Description: "登录 Windows 后自动启动 VRC++"; GroupDescription: "后台运行："; Flags: unchecked

[Files]
Source: "{#SourceExe}"; DestDir: "{app}"; DestName: "{#AppExeName}"; Flags: ignoreversion
Source: "..\apps\gateway\internal\tray\icon.ico"; DestDir: "{app}"; DestName: "vrc-plus-plus.ico"; Flags: ignoreversion

[Icons]
Name: "{group}\VRC++"; Filename: "{app}\{#AppExeName}"; WorkingDir: "{app}"; IconFilename: "{app}\vrc-plus-plus.ico"
Name: "{group}\卸载 VRC++"; Filename: "{uninstallexe}"
Name: "{autodesktop}\VRC++"; Filename: "{app}\{#AppExeName}"; WorkingDir: "{app}"; IconFilename: "{app}\vrc-plus-plus.ico"
Name: "{userstartup}\VRC++"; Filename: "{app}\{#AppExeName}"; WorkingDir: "{app}"; IconFilename: "{app}\vrc-plus-plus.ico"; Tasks: startup

[Run]
Filename: "{app}\{#AppExeName}"; Description: "启动 VRC++"; Flags: nowait postinstall; Check: ShouldLaunchApplication

[UninstallRun]
Filename: "{cmd}"; Parameters: "/C taskkill /IM {#AppExeName} /F"; Flags: runhidden; RunOnceId: "StopVRCPlusPlus"

[Code]
function InitializeSetup(): Boolean;
begin
  Result := True;
end;

function HasCommandLineParameter(const Value: String): Boolean;
var
  Index: Integer;
begin
  Result := False;
  for Index := 1 to ParamCount do
  begin
    if CompareText(ParamStr(Index), Value) = 0 then
    begin
      Result := True;
      Exit;
    end;
  end;
end;

function ShouldLaunchApplication(): Boolean;
begin
  Result := (not WizardSilent) or HasCommandLineParameter('/UPDATE=1');
end;
