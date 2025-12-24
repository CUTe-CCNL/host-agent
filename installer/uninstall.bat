@echo off
setlocal enabledelayedexpansion

:: Change to script directory
cd /d "%~dp0"

echo ========================================
echo   Host Agent Uninstall (Windows)
echo ========================================
echo.

:: Check admin privileges
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo Error: Please run as administrator
    pause
    exit /b 1
)

set SERVICE_NAME=HostAgent
set INSTALL_DIR=C:\Program Files\HostAgent

:: Check if service exists
echo Checking service status...
sc query %SERVICE_NAME% >nul 2>&1
if %errorLevel% equ 0 (
    echo Service found: %SERVICE_NAME%
    
    :: Stop service
    echo Stopping service...
    sc stop %SERVICE_NAME% >nul 2>&1
    timeout /t 2 /nobreak >nul
    
    :: Delete service
    echo Deleting service...
    sc delete %SERVICE_NAME% >nul 2>&1
    if %errorLevel% equ 0 (
        echo Service deleted successfully
    ) else (
        echo Warning: Failed to delete service
    )
) else (
    echo Service not found or already deleted
)

echo.

:: Check if install directory exists
if exist "%INSTALL_DIR%" (
    echo Install directory found: %INSTALL_DIR%
    echo.
    
    :: Ask to delete files
    set /p CONFIRM="Delete installation directory and config files? (Y/N): "
    if /i "!CONFIRM!"=="Y" (
        echo Deleting files...
        rd /s /q "%INSTALL_DIR%" 2>nul
        if exist "%INSTALL_DIR%" (
            echo Warning: Some files could not be deleted
        ) else (
            echo Installation directory deleted successfully
        )
    ) else (
        echo Keeping installation directory: %INSTALL_DIR%
    )
) else (
    echo Installation directory not found: %INSTALL_DIR%
)

echo.
echo ========================================
echo   Uninstall Complete
echo ========================================
echo.
pause