#!/bin/sh

IPROUTE=/sbin/ip

case "$reason" in
  pre-init)
    /usr/share/vpnc-scripts/vpnc-script pre-init
    ;;
  connect)
    INTERNAL_IP4_PREFIX=$(echo $INTERNAL_IP4_ADDRESS | sed -e's/\.[0-9]\+$//')
    $IPROUTE link set dev "$TUNDEV" up mtu 1024
    $IPROUTE addr add "$INTERNAL_IP4_ADDRESS/255.255.255.0" peer "$INTERNAL_IP4_ADDRESS" dev "$TUNDEV"
    $IPROUTE route replace "$INTERNAL_IP4_PREFIX.0/255.255.255.0" dev "$TUNDEV"
    $IPROUTE route flush cache

    while ! ip link show $TUNDEV >/dev/null 2>&1 ; do
	     sleep 0.1
    done

    $IPROUTE route add 192.168.1.0/24 via 192.168.178.1
    ;;
  disconnect)
    $IPROUTE link set dev "$TUNDEV" down
    ;;
  *)
    echo "unknown reason '$reason'. Maybe vpnc-script is out of date" 1>&2
    exit 1
    ;;
esac
exit 0
