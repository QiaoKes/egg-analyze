#define AppName "Egg Analyze V2"
#ifndef AppVersion
  #define AppVersion "0.1.0"
#endif
#ifndef AppSourceDir
  #error AppSourceDir must be defined
#endif
#ifndef OutputDir
  #define OutputDir "."
#endif
#ifndef OutputBaseFilename
  #define OutputBaseFilename "egg-analyze-v2-setup"
#endif

[Setup]
AppId={{A0CC79B8-CC5B-4435-9F6C-3D824AF62F84}
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher=top.aoe
DefaultDirName={autopf}\Egg Analyze V2
DefaultGroupName=Egg Analyze V2
DisableProgramGroupPage=yes
OutputDir={#OutputDir}
OutputBaseFilename={#OutputBaseFilename}
Compression=lzma
SolidCompression=yes
WizardStyle=modern
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
UninstallDisplayIcon={app}\egg_analyze_v2.exe

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"

[Files]
Source: "{#AppSourceDir}\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs createallsubdirs

[Icons]
Name: "{autoprograms}\Egg Analyze V2"; Filename: "{app}\egg_analyze_v2.exe"
Name: "{autodesktop}\Egg Analyze V2"; Filename: "{app}\egg_analyze_v2.exe"; Tasks: desktopicon

[Run]
Filename: "{app}\egg_analyze_v2.exe"; Description: "{cm:LaunchProgram,Egg Analyze V2}"; Flags: nowait postinstall skipifsilent
