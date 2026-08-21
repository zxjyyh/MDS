@echo off
chcp 65001 >nul
setlocal
REM 脚本位于 scripts\ 下，回退一级到仓库根目录
set "ROOT=%~dp0..\"

echo ============================================================
echo   内业管理系统 - Dev Mode Launcher
echo   (Backend: go run, Frontend: npm run dev)
echo ============================================================
echo.
echo   Backend  : http://localhost:3000  (go run)
echo   Frontend : http://localhost:8080  (Vite dev server)
echo.

start "Backend (3000)" cmd /k cd /d "%ROOT%backend" ^&^& go run ./cmd/server
start "Frontend (8080)" cmd /k cd /d "%ROOT%frontend" ^&^& npm run dev

echo.
echo   已打开两个窗口。稍候访问: http://localhost:8080
echo.
echo   首次启动的 admin 随机初始密码打印在 Backend 窗口，只显示一次。
echo   (关闭两个窗口即可停止全部服务)
timeout /t 3 >nul
