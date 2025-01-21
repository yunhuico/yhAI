#/bin/sh
# setenforce is in this path
PATH=$PATH:/sbin

dist=$(cat /etc/os-release | sed -n 's@^ID="\(.*\)"$@\1@p')

if ([ x$dist == 'xcoreos' ]); then
  echo "Detected CoreOS. All prerequisites already installed" >&2
  exit 0
fi

if ([ x$dist != 'xrhel' ] && [ x$dist != 'xcentos' ]); then
  echo "$dist is not supported. Only RHEL and CentOS are supported" >&2
  exit 0
fi

version=$(cat /etc/*-release | sed -n 's@^VERSION_ID="\(.*\)"$@\1@p')
if [ $version -lt 7 ]; then
  echo "$version is not supported. Only >= 7 version is supported" >&2
  exit 0
fi

if [ -f /opt/dcos-prereqs.installed ]; then
  echo "install_prereqs has been already executed on this host, exiting..."
  exit 0
fi

sudo setenforce 0 && sudo sed -i 's/^SELINUX=.*/SELINUX=disabled/g' /etc/sysconfig/selinux

sudo tee /etc/yum.repos.d/docker.repo <<-'EOF'
[dockerrepo]
name=Docker Repository
baseurl=https://yum.dockerproject.org/repo/main/centos/7
enabled=1
gpgcheck=1
gpgkey=https://yum.dockerproject.org/gpg
EOF

sudo yum -y update --exclude="docker-engine*"

sudo mkdir -p /etc/systemd/system/docker.service.d
sudo tee /etc/systemd/system/docker.service.d/override.conf <<- EOF
[Service]
Restart=always
StartLimitInterval=0
RestartSec=15
ExecStartPre=-/sbin/ip link del docker0
ExecStart=
ExecStart=/usr/bin/docker daemon --storage-driver=overlay -H fd://
EOF

sudo yum install -y docker-engine-1.11.2
sudo systemctl start docker
sudo systemctl enable docker

sudo yum install -y wget
sudo yum install -y git
sudo yum install -y unzip
sudo yum install -y curl
sudo yum install -y xz
sudo yum install -y ipset

sudo getent group nogroup || sudo groupadd nogroup
sudo touch /opt/dcos-prereqs.installed

