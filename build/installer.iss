#define AppExeName "clip-compress.exe"

#ifndef AppVersion
  #define AppVersion "0.0.0-dev"
#endif

#ifdef Dev
  #define AppName "ClipCompress (Dev)"
  #define AppId "{{89044BE7-7091-487D-8FA1-20C83D11A643}"
  #define DirName "ClipCompress Dev"
  #define DataName "ClipCompress (Dev)"
  #define MutexName "Global\ClipCompress-Dev"
  #define RunName "ClipCompress (Dev)"
  #define OutBase "ClipCompressSetup-Dev"
#else
  #define AppName "ClipCompress"
  #define AppId "{{21301B41-E199-4652-818B-6C54717A49BA}"
  #define DirName "ClipCompress"
  #define DataName "ClipCompress"
  #define MutexName "Global\ClipCompress"
  #define RunName "ClipCompress"
  #define OutBase "ClipCompressSetup"
#endif

[Setup]
AppId={#AppId}
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher=xsaveopt
SourceDir=..
DefaultDirName={autopf}\{#DirName}
DefaultGroupName={#AppName}
DisableProgramGroupPage=yes
OutputDir=dist
OutputBaseFilename={#OutBase}
SetupIconFile=assets\icon.ico
UninstallDisplayIcon={app}\{#AppExeName}
AppMutex={#MutexName}
Compression=lzma2
SolidCompression=yes
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible

[Files]
Source: "{#AppExeName}"; DestDir: "{app}"; Flags: ignoreversion
Source: "ffmpeg.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "ffmpeg-COPYING.LGPLv2.1.txt"; DestDir: "{app}"; Flags: ignoreversion
Source: "ffmpeg-README.txt"; DestDir: "{app}"; Flags: ignoreversion
Source: "assets\icon.ico"; DestDir: "{app}"; Flags: ignoreversion

[Tasks]
Name: "startup"; Description: "{cm:AutoStartProgram,{#AppName}}"; GroupDescription: "{cm:AutoStartProgramGroupDescription}"; Flags: unchecked

[Icons]
Name: "{group}\{#AppName}"; Filename: "{app}\{#AppExeName}"; IconFilename: "{app}\icon.ico"
Name: "{userstartmenu}\{#AppName}"; Filename: "{app}\{#AppExeName}"; IconFilename: "{app}\icon.ico"

[Registry]
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; ValueName: "{#RunName}"; ValueData: """{app}\{#AppExeName}"""; Flags: uninsdeletevalue; Tasks: startup

[UninstallDelete]
Type: filesandordirs; Name: "{userappdata}\{#DataName}"

[Run]
Filename: "{app}\{#AppExeName}"; Description: "Launch {#AppName}"; Flags: nowait postinstall skipifsilent

[Code]
procedure StopRunningApp;
var
  ResultCode: Integer;
begin
  Exec(ExpandConstant('{sys}\taskkill.exe'), '/f /t /im {#AppExeName}', '',
    SW_HIDE, ewWaitUntilTerminated, ResultCode);
end;

function InitializeSetup(): Boolean;
begin
  StopRunningApp;
  Result := True;
end;

function InitializeUninstall(): Boolean;
begin
  StopRunningApp;
  Result := True;
end;
