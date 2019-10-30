#!/bin/bash

set -e

# wait for MySQL DB to be ready
counter=1
while ! mysql --protocol TCP -h mysql_host -u root -ppassword -e "show databases;" > /dev/null 2>&1; do
    sleep 1
    ((counter++))
    if [ $counter -gt 10 ]; then
        >&2 echo "We have been waiting for MySQL too long already; failing."
        exit 1
    fi
done
