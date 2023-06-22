# Gopherit Webcast Demo

## Install Go

```bash
sudo apt update
sudo apt install golang -y

# OR
wget https://go.dev/dl/go1.20.5.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.20.5.linux-amd64.tar.gz
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

## Lab 1

### Initialize the project

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

## Lab 2

### Install Viper

```bash
go get github.com/spf13/viper
go mod tidy
```

## Lab 3

### Lint and Test

```bash
make -f Makefile lint
make -f Makefile test
```

### Debugging

Step through the code with a debugger of your choice.

## Lab 4

### Install nmap library

```bash
go get -v github.com/Ullaakut/nmap/v2
go mod tidy
```

## Lab 5

### Install govalidator

```bash
go get -v github.com/asaskevich/govalidator
go mod tidy
```