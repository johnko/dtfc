#!/bin/sh

SCRIPTDIR="./localscripts"
LOCALHOST="127.0.0.1"
BASEDIR="${HOME}/z/test-five-base"
TEMPDIR="${HOME}/z/test-five-tmp"
LOGFILE="${HOME}/z/test-five-log"
PIDFILE="${HOME}/z/test-five-pid"

exiterror() {
    exitcode=$1
    shift
    if [ "0" != "${exitcode}" ]; then
        echo "ERROR: ${@}" >&2
    fi
    exit ${exitcode}
}

launchenv() {
    mkdir -p "${BASEDIR}${1}"
    mkdir -p "${TEMPDIR}${1}"
    touch "${LOGFILE}${1}"
    ./main --port 800${1} --basedir "${BASEDIR}${1}" --temp "${TEMPDIR}${1}" --log "${LOGFILE}${1}" --me "http://${LOCALHOST}:800${1}/" --peerlist extras/peerlist-five.txt &
    echo $! >"${PIDFILE}${1}"
    sleep 1
    curl -X GET http://${LOCALHOST}:800${1}/health.html || exiterror 1 "couldn't GET health.html"
}

sh ${SCRIPTDIR}/build-server.sh

if [ ! -x ./main ]; then
    exiterror 1 "build-server.sh did not create a main executable"
fi

launchenv 1
launchenv 2
launchenv 3
launchenv 4
launchenv 5

sleep 1
curl -v -X PUT --data-binary @LICENSE http://${LOCALHOST}:8001/license.txt || exiterror 1 "couldn't PUT license.txt"

echo "---------------------------"

sleep 1
curl -v -X GET http://${LOCALHOST}:8002/13377b3886e4f6fa1db0610fe4983f3bfa8fa0e7ab3b7179687a7d3ad1f60317a5951f4c4accf6596244531b8f7c4967480b04366925a0eac915697c3daecaf8 || exiterror 1 "couldn't GET license.txt"

kill `cat ${PIDFILE}1`

echo "---------------------------"

sleep 1
curl -v -X GET http://${LOCALHOST}:8003/13377b3886e4f6fa1db0610fe4983f3bfa8fa0e7ab3b7179687a7d3ad1f60317a5951f4c4accf6596244531b8f7c4967480b04366925a0eac915697c3daecaf8 || exiterror 1 "couldn't GET license.txt"

kill `cat ${PIDFILE}2`

echo "---------------------------"

sleep 1
curl -v -X GET http://${LOCALHOST}:8004/13377b3886e4f6fa1db0610fe4983f3bfa8fa0e7ab3b7179687a7d3ad1f60317a5951f4c4accf6596244531b8f7c4967480b04366925a0eac915697c3daecaf8 || exiterror 1 "couldn't GET license.txt"

kill `cat ${PIDFILE}3`

echo "---------------------------"

sleep 1
curl -v -X GET http://${LOCALHOST}:8005/13377b3886e4f6fa1db0610fe4983f3bfa8fa0e7ab3b7179687a7d3ad1f60317a5951f4c4accf6596244531b8f7c4967480b04366925a0eac915697c3daecaf8 || exiterror 1 "couldn't GET license.txt"

kill `cat ${PIDFILE}4`

echo "---------------------------"

sleep 1
curl -v -X GET http://${LOCALHOST}:8005/404

kill `cat ${PIDFILE}5`
