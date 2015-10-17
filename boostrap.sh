#!bin/sh
# This script is meant for quick & easy install via:
#   'curl -sSL https://github.com/emc-cmd/test-framework/bootstrap.sh | sh'
set -e
wget https://storage.googleapis.com/golang/go1.5.1.linux-amd64.tar.gz && sudo tar -C /usr/local -xzf go1.5.1.linux-amd64.tar.gz && chmod +x go1.5.1.linux-amd64.tar.gz && export GOPATH=/vagrant/my_go && export PATH=$PATH:/usr/local/go/bin && rm go1.5.1.linux-amd64.tar.gz
sudo apt-get update && sudo apt-get install software-properties-common git -y && sudo apt-get update && sudo add-apt-repository ppa:ubuntu-lxc/stable && sudo apt-get update && sudo apt-get install -y libprotobuf-c0 && sudo apt-get install -y criu
curl -sSL -O https://github.com/boucher/docker/releases/download/v1.9.0-experimental-cr.1/docker-1.9.0-dev && chmod +x docker-1.9.0-dev && sudo mv docker-1.9.0-dev /usr/local/bin/docker
sudo docker daemon &
mkdir -p $GOPATH/src/github.com/emc-cmd/
cd $GOPATH/src/github.com/emc-cmd/
git clone https://github.com/emc-cmd/test-framework
cd test-framework
cd $GOPATH/src/github.com/emc-cmd/test-framework && go build -o example_scheduler && cd executor/ && go build -o example_executor && cd $GOPATH/src/github.com/emc-cmd/test-framework && sudo ./example_scheduler --master=127.0.0.1:5050 --executor="$GOPATH/src/github.com/emc-cmd/test-framework/executor/example_executor" --logtostderr=true