# go playground for testing against mesos

1. vagrant up
2. install golang
```
wget https://storage.googleapis.com/golang/go1.5.1.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.5.1.linux-amd64.tar.gz
chmod +x go1.5.1.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```
3. install direnv
```
wget https://github.com/direnv/direnv/releases/download/v2.6.0/direnv.linux-amd64
chmod +x direnv.linux-amd64
sudo mv direnv.linux-amd64 /usr/local/bin/direnv
```
4. install criu & git:
```
sudo apt-get install software-properties-common git -y
sudo add-apt-repository ppa:ubuntu-lxc/stable
sudo apt-get update
sudo apt-get install -y libprotobuf-c0
sudo apt-get install -f #if error occurs
sudo apt-get install -y criu
```

5. install docker with checkpoint/restore:
```
curl -sSL https://experimental.docker.com/ | sh
```

6. install mesos
```
sudo apt-get update
sudo apt-get install -y git openjdk-7-jdk autoconf libtool build-essential python-dev python-boto libcurl4-nss-dev libsasl2-dev maven libapr1-dev libsvn-dev

git clone https://git-wip-us.apache.org/repos/asf/mesos.git
cd mesos
./bootstrap
mkdir build
cd build
../configure
make
make check
make install

# Start mesos master (Ensure work directory exists and has proper permissions).
mkdir -p /var/lib/mesos
sudo ./bin/mesos-master.sh --ip=127.0.0.1 --work_dir=/var/lib/mesos
# Start mesos slave.
sudo ./bin/mesos-slave.sh --master=127.0.0.1:5050
```

don't forget to ```go get ./...```

```cd $GOPATH/src/github.com/emc-cmd/test-framework && go build -o example_scheduler && cd executor/ && go build -o example_executor && cd $GOPATH/src/github.com/emc-cmd/test-framework && sudo ./example_scheduler --master=127.0.0.1:5050 --executor="$GOPATH/src/github.com/emc-cmd/test-framework/executor/example_executor" --logtostderr=true```
