@echo off
REM SparkEnhance — install shortcut and auto-start
REM Run as Administrator if you want machine-wide (all users) install.

setlocal

set "EXE=%~dp0build\bin\sparkenhance.exe"
set "ICO=%~dp0build\windows\icon.ico"
set "LNK=%APPDATA%\Microsoft\Windows\Start Menu\Programs\SparkEnhance.lnk"

if not exist "%EXE%" (
    echo ERROR: %EXE% not found. Run 'wails build' first.
    exit /b 1
)

if not exist "%ICO%" (
    echo WARNING: icon file not found, shortcut will use default icon.
    set "ICO=%EXE%"
)

echo Creating Start Menu shortcut...
powershell -NoProfile -ExecutionPolicy Bypass -Command ^
    "$wsh = New-Object -ComObject WScript.Shell; " ^
    "$lnk = $wsh.CreateShortcut('%LNK%'); " ^
    "$lnk.TargetPath = '%EXE%'; " ^
    "$lnk.WorkingDirectory = '%~dp0build\bin'; " ^
    "$lnk.IconLocation = '%ICO%,0'; " ^
    "$lnk.Description = 'SparkEnhance - AI prompt enhancement'; " ^
    "$lnk.Save(); " ^
    "Write-Host 'Shortcut created: %LNK%'"

echo.
echo Registering auto-start at login...
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" /v SparkEnhance /t REG_SZ /d "\"%EXE%\"" /f >nul
if errorlevel 1 (
    echo WARNING: failed to register Run key.
) else (
    echo Auto-start registered.
)

echo.
echo Done! Launch SparkEnhance from the Start Menu.
endlocal
