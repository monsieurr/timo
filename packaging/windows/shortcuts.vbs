Option Explicit
Dim shell, fso, targetDir, menu, shortcut, action
Set shell = CreateObject("WScript.Shell")
Set fso = CreateObject("Scripting.FileSystemObject")
action = WScript.Arguments(0)
targetDir = WScript.Arguments(1)
menu = shell.SpecialFolders("Programs")
If action = "remove" Then
  If fso.FileExists(menu & "\Timo.lnk") Then fso.DeleteFile menu & "\Timo.lnk", True
  If fso.FileExists(menu & "\Timo terminal.lnk") Then fso.DeleteFile menu & "\Timo terminal.lnk", True
  If fso.FileExists(menu & "\Quit Timo.lnk") Then fso.DeleteFile menu & "\Quit Timo.lnk", True
  WScript.Quit 0
End If
If action <> "install" Then WScript.Quit 1
Set shortcut = shell.CreateShortcut(menu & "\Timo.lnk")
shortcut.TargetPath = targetDir & "\timo-app.exe"
shortcut.WorkingDirectory = targetDir
shortcut.Description = "Open the Timo timer"
shortcut.Save
Set shortcut = shell.CreateShortcut(menu & "\Timo terminal.lnk")
shortcut.TargetPath = targetDir & "\timo.exe"
shortcut.WorkingDirectory = targetDir
shortcut.Description = "Open the Timo terminal"
shortcut.Save

Set shortcut = shell.CreateShortcut(menu & "\Quit Timo.lnk")
shortcut.TargetPath = targetDir & "\timo-app.exe"
shortcut.Arguments = "stop-desktop"
shortcut.WorkingDirectory = targetDir
shortcut.Description = "Stop the Timo dashboard server"
shortcut.Save
