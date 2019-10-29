#!/bin/bash
set -ex

# set configuration
rm /etc/scion/gen -rf
tar -C /etc/scion/ -xf /root/host_config.tar
for srv in `cat /etc/scion/scionlab-services.txt`; do
    systemctl enable $srv
done
systemctl start scionlab.target
