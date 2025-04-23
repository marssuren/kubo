#!/bin/bash

echo "===== 开始编译IPFS Kubo项目 ====="

# 确保输出目录存在
mkdir -p kubo/windows kubo/linux

echo "[1/2] 编译Windows版本..."
GOOS=windows GOARCH=amd64 go build -o kubo/windows/ipfs.exe ./cmd/ipfs
if [ $? -ne 0 ]; then
    echo "Windows版本编译失败!"
    exit 1
fi
echo "Windows版本编译成功: kubo/windows/ipfs.exe"

echo "[2/2] 编译Linux版本..."
GOOS=linux GOARCH=amd64 go build -o kubo/linux/ipfs ./cmd/ipfs
if [ $? -ne 0 ]; then
    echo "Linux版本编译失败!"
    exit 1
fi
echo "Linux版本编译成功: kubo/linux/ipfs"

# 设置Linux版本执行权限
chmod +x kubo/linux/ipfs

echo "===== 编译完成 ====="
echo "Windows可执行文件: kubo/windows/ipfs.exe"
echo "Linux可执行文件: kubo/linux/ipfs"
echo ""
echo "提示: "
echo "- Windows版本需要在Windows环境下运行"
echo "- Linux版本已添加执行权限"
echo ""
echo "启动示例:"
echo "Windows: ipfs.exe daemon --gateway=/ip4/127.0.0.1/tcp/8081"
echo "Linux: ./ipfs daemon --gateway=/ip4/127.0.0.1/tcp/8081"