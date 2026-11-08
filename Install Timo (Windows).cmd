@echo off
setlocal
set "HERE=%~dp0"
set "DEST=%LOCALAPPDATA%\Programs\Timo"
if not exist "%HERE%dist\timo-windows-amd64.exe" (
 echo Missing timo Windows executable. Extract the complete ZIP first.
 pause
 exit /b 1
)
if not exist "%HERE%dist\timo-app-windows-amd64.exe" (
 echo Missing timo desktop launcher. Extract the complete ZIP first.
 pause
 exit /b 1
)
if exist "%DEST%\timo.exe" (
 "%DEST%\timo.exe" stop-desktop >nul 2>nul
 timeout /t 2 /nobreak >nul
)
if not exist "%DEST%" mkdir "%DEST%"
if errorlevel 1 goto error
copy /Y "%HERE%dist\timo-windows-amd64.exe" "%DEST%\timo.exe" >nul
if errorlevel 1 goto error
copy /Y "%HERE%dist\timo-app-windows-amd64.exe" "%DEST%\timo-app.exe" >nul
if errorlevel 1 goto error
copy /Y "%HERE%packaging\windows\shortcuts.vbs" "%DEST%\shortcuts.vbs" >nul
if errorlevel 1 goto error
copy /Y "%HERE%Uninstall Timo (Windows).cmd" "%DEST%\Uninstall Timo.cmd" >nul
if errorlevel 1 goto error
cscript.exe //nologo "%DEST%\shortcuts.vbs" install "%DEST%"
if errorlevel 1 goto error
echo Installed Timo to "%DEST%".
echo Find Timo, Timo terminal and Quit Timo in the Start Menu.
echo Your existing settings and sessions are preserved.
start "" "%DEST%\timo-app.exe"
echo.
pause
exit /b 0
:error
echo Installation failed. Close any running Timo process and try again.
pause
exit /b 1
