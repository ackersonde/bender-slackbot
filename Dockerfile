FROM ubuntu:16.10
RUN apt-get update && apt-get install -y --no-install-recommends openssh-client inetutils-ping vim vpnc curl ca-certificates
# prepare build environment
ARG vpnc_gateway
ARG vpnc_id
ARG vpnc_secret
ARG vpnc_username
ARG vpnc_password

# setup vpnc post-hook scripts
WORKDIR /etc/vpnc
COPY fritzbox.conf .
COPY fritzbox-script.sh .
RUN sed -i -e "s/{{gateway}}/$vpnc_gateway/" -e "s/{{id}}/$vpnc_id/" -e "s/{{secret}}/$vpnc_secret/" -e "s/{{username}}/$vpnc_username/" -e "s/{{password}}/$vpnc_password/" fritzbox.conf

# get slackbot in place
RUN mkdir /app
COPY bender /app/
WORKDIR /app
ENTRYPOINT ["./bender"]
