# Commands

## Install Go

```bash
sudo apt update
sudo apt install golang -y

# OR
wget https://go.dev/dl/go1.20.5.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.20.5.linux-amd64.tar.gz
```

Update your PATH

```bash
cat << 'EOF' >> "${HOME}/.zshrc"

# Add ~/go/bin to path
[[ ":$PATH:" != *":${HOME}/go/bin:"* ]] && export PATH="${PATH}:${HOME}/go/bin"
# Set GOPATH
if [[ -z "${GOPATH}" ]]; then export GOPATH="${HOME}/go"; fi
EOF
fi

# now source .zshrc to initialize the changes
source ~/.zshrc
```

# Lab 1

## Initialize the project

```bash
mkdir –p ~/projects/goforit && cd ~/projects/goforit
git init
go mod init github.com/USERNAME/PROJECTNAME
```

- Initialize Cobra and add a subcommand

```bash
go install github.com/spf13/cobra-cli@latest
cobra-cli init

cobra-cli add scan
go mod tidy
```
