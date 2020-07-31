#!/bin/bash

BEELOG_STATE=/tmp/beelog-*.log
APP_LOG=/tmp/log-file-*.log

echo "The following files will be permanently REMOVED:"
echo "$BEELOG_STATE"
echo "$APP_LOG"
echo ""
echo "deleting in 3s ..."
sleep 3s

rm $BEELOG_STATE
rm $APP_LOG
echo "finished!"
