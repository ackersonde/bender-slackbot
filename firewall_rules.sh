#!/bin/bash
SERVERS="ubuntu@$MASTER_HOSTNAME ubuntu@$SLAVE_HOSTNAME ackersond@$BUILD_HOSTNAME"

for i in $SERVERS
do
   ssh -o StrictHostKeyChecking=no $i \
   "echo \"would be ufw allowing $NEW_SERVER_IPV6 and ufw removing $OLD_SERVER_IPV6 for port 22\" > /tmp/touch"
#      "sudo ufw allow from $NEW_SERVER_IPV6 to any port 22 && \
#      sudo ufw --force delete \`sudo ufw status numbered | grep $OLD_SERVER_IPV6 | grep -o -E '[0-9]+' | head -1\`"
done
