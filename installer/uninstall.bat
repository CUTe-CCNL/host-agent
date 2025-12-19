@echo off

echo ========================================
echo   Host Agent 卸載程式 (Windows)
echo ========================================
echo.

:: 檢查管理員權限
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo 錯誤: 請以管理員身份執行此腳本
    pause
    exit /b 1
)

set SERVICE_NAME=HostAgent
set INSTALL_DIR=C:\Program Files\HostAgent

:: 停止服務
echo 停止服務...
sc stop %SERVICE_NAME%
timeout /t 2 /nobreak >nul

:: 刪除服務
echo 刪除服務...
sc delete %SERVICE_NAME%

if %errorLevel% neq 0 (
    echo 警告: 服務刪除失敗或服務不存在
)

:: 詢問是否刪除檔案
set /p CONFIRM="是否刪除安裝目錄和配置檔? (Y/N): "
if /i "%CONFIRM%"=="Y" (
    echo 刪除檔案...
    rd /s /q "%INSTALL_DIR%" 2>nul
    echo 卸載完成
) else (
    echo 保留安裝目錄: %INSTALL_DIR%
)

echo.
echo 卸載完成
pause
