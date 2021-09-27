#!/bin/bash

cmd=$1
max_attempts=10
interval=2
emulator_port=6969
running=0


if [ $(gcloud components list 2> /dev/null | grep -E "│ Installed.*(cloud-firestore-emulator|beta)" -c) -ne 2 ]; then # expects two lines of matches: beta and firestore
  echo "installing required gcloud components"
  gcloud components install beta cloud-firestore-emulator
fi

gcloud beta emulators firestore start --host-port=localhost:${emulator_port} &> /dev/null &
echo "waiting for firestore emulator to start..."

for attempt in $(seq 1 $max_attempts); do
  echo "attempt ${attempt}/${max_attempts}"
  if netstat -an | grep LISTEN | grep ${emulator_port} &>/dev/null; then
    running=1
    echo "firestore is running"
    break
  fi
  sleep ${interval}
done

if [ "$running" -eq 1 ]; then
  FIRESTORE_EMULATOR_HOST=localhost:${emulator_port} $cmd
  pkill -f firestore
else
  echo "could not start firestore :("
  pkill -f firestore # this will hopefully kill a potentially initializing firestore anyway
fi
