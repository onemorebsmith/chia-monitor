#!/bin/bash
echo "Building Monitor..."
go build -o chia_monitor
echo "Killing previous..."
pkill chia_monitor
echo "Launching..."
./chia_monitor &
echo "Finished."