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
#ifndef AppId
  #define AppId "{{9D6584CB-59EC-4EC2-80EC-E95B14BD1A5D}"
#endif
#ifndef UninstallKeyName
  #define UninstallKeyName "{9D6584CB-59EC-4EC2-80EC-E95B14BD1A5D}_is1"
#endif

[Setup]
AppId={#AppId}
AppName={#AppName}
AppVersion={#AppVersion}
AppVerName={#AppName} {#AppVersion}
AppPublisher=VRC++
AppSupportURL=mailto:2579362548@qq.com
DefaultDirName={code:GetInstallDirectory}
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
const
  VRCPlusPlusUninstallKey = 'Software\Microsoft\Windows\CurrentVersion\Uninstall\{#UninstallKeyName}';

var
  UpgradeDetected: Boolean;
  InstalledVersion: String;
  InstalledDirectory: String;

function ReadInstalledApplication(const RootKey: Integer): Boolean;
begin
  Result := RegQueryStringValue(RootKey, VRCPlusPlusUninstallKey,
    'DisplayVersion', InstalledVersion);
  if Result then
    RegQueryStringValue(RootKey, VRCPlusPlusUninstallKey,
      'InstallLocation', InstalledDirectory);
end;

function DetectInstalledApplication(): Boolean;
begin
  InstalledVersion := '';
  InstalledDirectory := '';
  Result := ReadInstalledApplication(HKCU);
  if not Result then
    Result := ReadInstalledApplication(HKLM);
end;

function InitializeSetup(): Boolean;
begin
  UpgradeDetected := DetectInstalledApplication();
  Result := True;
end;

function GetInstallDirectory(Param: String): String;
begin
  if UpgradeDetected and (InstalledDirectory <> '') then
    Result := RemoveBackslashUnlessRoot(InstalledDirectory)
  else
    Result := ExpandConstant('{localappdata}\Programs\VRC++');
end;

procedure InitializeWizard();
var
  UpgradePage: TOutputMsgWizardPage;
begin
  if UpgradeDetected then
  begin
    UpgradePage := CreateOutputMsgPage(wpWelcome,
      '检测到已安装的 VRC++',
      '安装程序将执行原位覆盖升级',
      '已安装版本：' + InstalledVersion + #13#10 +
      '升级版本：{#AppVersion}' + #13#10 +
      '原安装目录：' + InstalledDirectory + #13#10#13#10 +
      '程序文件会被新版本覆盖；好友记录、登录会话、特别关心、邮件配置和守护记录会继续保留。无需先卸载旧版本。');
  end;
end;

function ShouldSkipPage(PageID: Integer): Boolean;
begin
  Result := UpgradeDetected and (PageID = wpSelectDir);
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
