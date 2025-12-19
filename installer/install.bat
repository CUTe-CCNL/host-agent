@echo off
setlocal enabledelayedexpansion

echo ========================================
echo   Host Agent 安裝程式 (Windows)
echo ========================================
echo.

:: 檢查管理員權限
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo 錯誤: 請以管理員身份執行此腳本
    pause
    exit /b 1
)

:: 配置
set SERVICE_NAME=HostAgent
set INSTALL_DIR=C:\Program Files\HostAgent
set CONFIG_FILE=%INSTALL_DIR%\config.yaml
set BINARY_NAME=host-agent-windows-amd64.exe
set DEST_BINARY=%INSTALL_DIR%\host-agent.exe

:: 檢查執行檔是否存在
if not exist "%BINARY_NAME%" (
    echo 錯誤: 找不到執行檔 %BINARY_NAME%
    echo 請先執行編譯
    pause
    exit /b 1
)

:: 停止舊服務
echo 停止舊服務...
sc stop %SERVICE_NAME% >nul 2>&1
sc delete %SERVICE_NAME% >nul 2>&1
timeout /t 2 /nobreak >nul

:: 建立安裝目錄
echo 建立安裝目錄...
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"

:: 複製檔案
echo 安裝執行檔...
copy /Y "%BINARY_NAME%" "%DEST_BINARY%" >nul

:: 複製配置檔（如果不存在）
if not exist "%CONFIG_FILE%" (
    echo 安裝配置檔...
    copy /Y config.yaml "%CONFIG_FILE%" >nul
) else (
    echo 配置檔已存在，跳過
)

:: 安裝服務
echo 安裝 Windows 服務...
"%DEST_BINARY%" -install -config "%CONFIG_FILE%"

if %errorLevel% neq 0 (
    echo 錯誤: 服務安裝失敗
    pause
    exit /b 1
)

:: 啟動服務
echo 啟動服務...
sc start %SERVICE_NAME%

if %errorLevel% neq 0 (
    echo 警告: 服務啟動失敗，請檢查配置
    sc query %SERVICE_NAME%
    pause
    exit /b 1
)

:: 等待服務啟動
timeout /t 3 /nobreak >nul

:: 檢查服務狀態
sc query %SERVICE_NAME% | find "RUNNING" >nul
if %errorLevel% equ 0 (
    echo.
    echo ========================================
    echo   安裝成功！
    echo ========================================
    echo.
    echo 服務名稱: %SERVICE_NAME%
    echo 安裝路徑: %INSTALL_DIR%
    echo 配置檔案: %CONFIG_FILE%
    echo.
    echo 常用命令:
    echo   查看狀態: sc query %SERVICE_NAME%
    echo   啟動服務: net start %SERVICE_NAME%
    echo   停止服務: net stop %SERVICE_NAME%
    echo   重啟服務: net stop %SERVICE_NAME% ^&^& net start %SERVICE_NAME%
    echo   卸載服務: uninstall.bat
    echo.
    echo 測試 API:
    echo   curl http://localhost:9100/health
    echo.
) else (
    echo 錯誤: 服務未正常啟動
    sc query %SERVICE_NAME%
)

pause
