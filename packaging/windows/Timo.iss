; Compile with Inno Setup ISCC.exe from the repository root.
; This per-user installer never requires administrator rights.
#define AppVersion "1.0.4"
[Setup]
AppId={{B550E144-EF6D-4E19-9C89-7B4B1626E154}
AppName=Timo
AppVersion={#AppVersion}
AppPublisher=Timo
DefaultDirName={localappdata}\Programs\Timo
DefaultGroupName=Timo
PrivilegesRequired=lowest
OutputDir=..\..\release
OutputBaseFilename=timo-setup-windows-amd64
Compression=lzma
SolidCompression=yes
UninstallDisplayIcon={app}\timo-app.exe
CloseApplications=yes
WizardStyle=modern

[Files]
Source: "..\..\dist\timo-windows-amd64.exe"; DestDir: "{app}"; DestName: "timo.exe"; Flags: ignoreversion
Source: "..\..\dist\timo-app-windows-amd64.exe"; DestDir: "{app}"; DestName: "timo-app.exe"; Flags: ignoreversion

[Tasks]
Name: "desktopicon"; Description: "Create a desktop shortcut"; GroupDescription: "Shortcuts:"; Flags: unchecked

[Icons]
Name: "{autoprograms}\Timo"; Filename: "{app}\timo-app.exe"; Comment: "Open the Timo timer"
Name: "{autoprograms}\Timo terminal"; Filename: "{app}\timo.exe"; Comment: "Start the Timo terminal interface"
Name: "{autoprograms}\Quit Timo"; Filename: "{app}\timo-app.exe"; Parameters: "stop-desktop"; Comment: "Stop the Timo dashboard server"
Name: "{autodesktop}\Timo"; Filename: "{app}\timo-app.exe"; Tasks: desktopicon

[Run]
Filename: "{app}\timo-app.exe"; Description: "Launch Timo"; Flags: postinstall nowait skipifsilent
