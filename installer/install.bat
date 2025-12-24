@echo off
setlocal enabledelayedexpansion

cd /d "%~dp0"
echo Current directory: %CD%

echo ========================================
echo   Host Agent Install (Windows)
echo ========================================
echo.

:: Check admin
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo Error: Please run as administrator
    pause
    exit /b 1
)

:: Config
set SERVICE_NAME=HostAgent
set INSTALL_DIR=C:\Program Files\HostAgent
set CONFIG_FILE=%INSTALL_DIR%\config.yaml
set BINARY_NAME=host-agent.exe
set DEST_BINARY=%INSTALL_DIR%\host-agent.exe

:: Check binary
if not exist "%BINARY_NAME%" (
    echo Error: Binary not found: %BINARY_NAME%
    pause
    exit /b 1
)

:: Stop old service
echo Stopping old service...
sc query %SERVICE_NAME% >nul 2>&1
if %errorLevel% equ 0 (
    sc stop %SERVICE_NAME% >nul 2>&1
    timeout /t 2 /nobreak >nul
    sc delete %SERVICE_NAME% >nul 2>&1
    timeout /t 2 /nobreak >nul
)

:: Create directory
echo Creating directory...
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"

:: Copy files
echo Installing binary...
copy /Y "%BINARY_NAME%" "%DEST_BINARY%" >nul

if not exist "%CONFIG_FILE%" (
    if exist "config.yaml" (
        echo Installing config...
        copy /Y config.yaml "%CONFIG_FILE%" >nul
    )
)

:: Install service
echo Installing service...
"%DEST_BINARY%" -install -config "%CONFIG_FILE%"
if %errorLevel% neq 0 (
    echo Error: Service installation failed
    pause
    exit /b 1
)

:: Start service
echo Starting service...
sc start %SERVICE_NAME% >nul 2>&1

:: Wait for service
echo Waiting for service to start...
set /a count=0
:check_loop
timeout /t 1 /nobreak >nul
sc query %SERVICE_NAME% | find "RUNNING" >nul
if %errorLevel% equ 0 goto success
set /a count+=1
if %count% lss 10 goto check_loop

echo.
echo Warning: Service did not start in time
sc query %SERVICE_NAME%
pause
exit /b 1

:success
echo.
echo ========================================
echo   Installation Success!
echo ========================================
echo.
echo Service: %SERVICE_NAME%
echo Path: %INSTALL_DIR%
echo Config: %CONFIG_FILE%
echo.
echo Commands:
echo   sc query %SERVICE_NAME%
echo   net start %SERVICE_NAME%
echo   net stop %SERVICE_NAME%
echo.
echo Test:
echo   curl http://localhost:9100/health
echo.

pause