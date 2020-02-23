#!/bin/bash

#  referenced by ~/.config/transmission-daemon/settings.json @
#  "script-torrent-done-filename": "/home/pi/torrent_finished.sh"

TR_TORRENT_DIR=${TR_TORRENT_DIR:-$1}
TR_TORRENT_NAME=${TR_TORRENT_NAME:-$2}
TR_TORRENT_ID=${TR_TORRENT_ID:-$3}

PLEX_TORRENT_LIBRARY_SECTION=3
PLEX_TOKEN=$PLEX_TOKEN
# https://support.plex.tv/articles/201638786-plex-media-server-url-commands/

sourcePath="/home/pi/torrents"
destinationPath="/mnt/usb4TB/DLNA/torrents"

transmission-remote localhost:9091 -t "${TR_TORRENT_ID}" --move "${sourcePath}"
transmission-remote localhost:9091 -t "${TR_TORRENT_ID}" --remove

if mv "${sourcePath}/$TR_TORRENT_NAME" "${destinationPath}"/ ; then
    detox -r "${destinationPath}/$TR_TORRENT_NAME"
    curl http://192.168.179.59:32400/library/sections/$PLEX_TORRENT_LIBRARY_SECTION/refresh?X-Plex-Token=$PLEX_TOKEN
fi
