FROM ubuntu
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends tzdata openssh-client vpnc curl ca-certificates
# prepare build environment
ARG vpnc_gateway
ARG vpnc_id
ARG vpnc_secret
ARG vpnc_username
ARG vpnc_password

# setup vpnc config & scripts
WORKDIR /etc/vpnc
COPY fritzbox.conf .
COPY fritzbox-script.sh .
RUN sed -i -e "s/{{gateway}}/$vpnc_gateway/" -e "s/{{id}}/$vpnc_id/" -e "s/{{secret}}/$vpnc_secret/" -e "s/{{username}}/$vpnc_username/" -e "s/{{password}}/$vpnc_password/" fritzbox.conf

# get localtime setup
RUN ln -fs /usr/share/zoneinfo/Europe/Berlin /etc/localtime && dpkg-reconfigure -f noninteractive tzdata

# get slackbot in place
RUN mkdir /app
COPY bender /app/
WORKDIR /app
ENTRYPOINT ["/bin/sh", "./bender"]
