#!/bin/bash

# Go安装脚本 for macOS

set -e

GO_VERSION="1.21.5"
GO_OS="darwin"
GO_ARCH="amd64"

# 检测系统架构
if [[ $(uname -m) == "arm64" ]]; then
    GO_ARCH="arm64"
fi

GO_TARBALL="go${GO_VERSION}.${GO_OS}-${GO_ARCH}.tar.gz"
GO_URL="https://golang.org/dl/${GO_TARBALL}"

echo "正在下载 Go ${GO_VERSION} for ${GO_OS}-${GO_ARCH}..."

# 下载Go
curl -L "${GO_URL}" -o "/tmp/${GO_TARBALL}"

# 删除旧的Go安装（如果存在）
sudo rm -rf /usr/local/go

# 解压到/usr/local
sudo tar -C /usr/local -xzf "/tmp/${GO_TARBALL}"

# 清理下载文件
rm "/tmp/${GO_TARBALL}"

# 添加到PATH
echo "正在配置环境变量..."

# 检查shell类型并添加到相应的配置文件
if [[ $SHELL == *"zsh"* ]]; then
    SHELL_RC="$HOME/.zshrc"
elif [[ $SHELL == *"bash"* ]]; then
    SHELL_RC="$HOME/.bash_profile"
else
    SHELL_RC="$HOME/.profile"
fi

# 检查是否已经添加了Go路径
if ! grep -q "/usr/local/go/bin" "$SHELL_RC" 2>/dev/null; then
    echo "" >> "$SHELL_RC"
    echo "# Go" >> "$SHELL_RC"
    echo "export PATH=\$PATH:/usr/local/go/bin" >> "$SHELL_RC"
    echo "export GOPATH=\$HOME/go" >> "$SHELL_RC"
    echo "export PATH=\$PATH:\$GOPATH/bin" >> "$SHELL_RC"
fi

# 临时设置环境变量
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin

echo "Go ${GO_VERSION} 安装完成！"
echo "请运行以下命令重新加载环境变量："
echo "source ${SHELL_RC}"
echo ""
echo "或者重新打开终端。"
echo ""
echo "验证安装："
echo "go version"