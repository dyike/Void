#!/bin/bash

set -e

echo "🚀 开始执行 Release 准备脚本..."

# 1. 重新生成多平台应用图标
echo "🎨 [1/3] 正在生成多平台应用图标 (macOS .icns, Windows .ico)..."
wails3 generate icons -input build/appicon.png -macassetdir build/darwin -windowsfilename build/windows/icon.ico

# 2. 覆盖开发包图标（解决 macOS 开发过程中的图标缓存顽固问题）
echo "🔄 [2/3] 强制更新 macOS 开发包 (Void.dev.app) 的图标缓存..."
if [ -d "bin/Void.dev.app" ]; then
    cp build/darwin/icons.icns bin/Void.dev.app/Contents/Resources/icons.icns || true
    touch bin/Void.dev.app
    touch bin/Void.dev.app/Contents/Info.plist
    echo "✅ 开发包图标缓存刷新指令已发送"
else
    echo "⚠️ 未找到 bin/Void.dev.app，跳过开发包缓存刷新"
fi

# 3. 打包发布版本
echo "📦 [3/3] 正在构建生产环境应用包..."
wails3 package

echo "✨ Release 准备完成！"
