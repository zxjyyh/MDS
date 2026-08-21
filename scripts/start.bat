@echo off
chcp 65001 >nul
setlocal
REM 脚本位于 scripts\ 下，回退一级到仓库根目录
set "ROOT=%~dp0..\"

echo ============================================================
echo   内业管理系统 - Production Mode
echo ============================================================
echo.

REM 1. 构建前端
echo [1/3] 正在构建前端...
cd /d "%ROOT%frontend"
call npm run build
if %ERRORLEVEL% NEQ 0 (
    echo 前端构建失败!
    pause
    exit /b 1
)

REM 2. 复制到后端
echo [2/3] 正在复制前端产物...
if exist "%ROOT%backend\web\dist\assets" rd /s /q "%ROOT%backend\web\dist\assets" 2>nul
xcopy /s /y /q "%ROOT%frontend\dist\*" "%ROOT%backend\web\dist\"

REM 3. 启动后端
echo [3/3] 正在启动服务...
echo.
echo   访问地址: http://localhost:3000
echo   首次启动会创建 admin 账号，其随机初始密码只在本窗口打印一次，
echo   请立即复制保存；也可提前设置环境变量 ADMIN_INIT_PASSWORD 自定义。
echo   Ctrl+C 停止服务
echo ============================================================
echo.
cd /d "%ROOT%backend"
go run ./cmd/server
