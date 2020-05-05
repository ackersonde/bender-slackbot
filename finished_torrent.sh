#!/bin/bash

#  referenced by ~/.config/transmission-daemon/settings.json @
#  "script-torrent-done-filename": "/home/ubuntu/finished_torrent.sh"

TR_TORRENT_DIR=${TR_TORRENT_DIR:-$1}
TR_TORRENT_NAME=${TR_TORRENT_NAME:-$2}
TR_TORRENT_ID=${TR_TORRENT_ID:-$3}

PLEX_TORRENT_LIBRARY_SECTION=3
PLEX_TOKEN=$PLEX_TOKEN
# https://support.plex.tv/articles/201638786-plex-media-server-url-commands/

sourcePath="/home/ubuntu/torrents"
destinationPath="/mnt/usb4TB/DLNA/torrents"

transmission-remote localhost:9091 -t "${TR_TORRENT_ID}" --move "${sourcePath}"
transmission-remote localhost:9091 -t "${TR_TORRENT_ID}" --remove

if mv "${sourcePath}/$TR_TORRENT_NAME" "${destinationPath}"/ ; then
    detox -r "${destinationPath}/$TR_TORRENT_NAME"
    curl "http://vpnpi.fritz.box:32400/library/sections/3/refresh?X-Plex-Product=Plex%20Web&X-Plex-Version=4.22.3&X-Plex-Client-Identifier=$PLEX_TOKEN&X-Plex-Platform=Chrome&X-Plex-Platform-Version=80.0&X-Plex-Sync-Version=2&X-Plex-Features=external-media%2Cindirect-media&X-Plex-Model=bundled&X-Plex-Device=OSX&X-Plex-Device-Name=Chrome&X-Plex-Device-Screen-Resolution=1152x1926%2C1440x2560&X-Plex-Language=en"
fi
