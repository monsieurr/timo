@echo off
setlocal
set "DEST=%LOCALAPPDATA%\Programs\Timo"
if exist "%DEST%\timo.exe" (
 "%DEST%\timo.exe" stop-desktop >nul 2>nul
 timeout /t 2 /nobreak >nul
)
if exist "%DEST%\shortcuts.vbs" (
 cscript.exe //nologo "%DEST%\shortcuts.vbs" remove "%DEST%"
)
if exist "%DEST%\timo.exe" del /F /Q "%DEST%\timo.exe"
if exist "%DEST%\timo-app.exe" del /F /Q "%DEST%\timo-app.exe"
echo Timo executables and Start Menu entries removed.
echo Your history and settings in AppData were not deleted.
echo If files were in use, close Timo and run the uninstaller again.
echo.
pause
