#!/bin/sh

SCRIPTDIR="./localscripts"
LOCALHOST="127.0.0.1"
BASEDIR="${HOME}/z/test-three-base"
TEMPDIR="${HOME}/z/test-three-tmp"
LOGFILE="${HOME}/z/test-three-log"
PIDFILE="${HOME}/z/test-three-pid"

LICENSEHASH=`openssl dgst -sha512 LICENSE | awk '{print $NF}'`

exiterror() {
    exitcode=$1
    shift
    if [ "0" != "${exitcode}" ]; then
        echo "ERROR: ${@}" >&2
    fi
    kill `cat ${PIDFILE}1`
    kill `cat ${PIDFILE}2`
    kill `cat ${PIDFILE}3`
    exit ${exitcode}
}

launchenv() {
    mkdir -p "${BASEDIR}${1}"
    mkdir -p "${TEMPDIR}${1}"
    touch "${LOGFILE}${1}"
    ./main --port 800${1} --basedir "${BASEDIR}${1}" --temp "${TEMPDIR}${1}" --log "${LOGFILE}${1}" --me "http://${LOCALHOST}:800${1}/" --peerlist extras/peerlist-three.txt &
    echo $! >"${PIDFILE}${1}"
    sleep 1
    curl -s -X GET http://${LOCALHOST}:800${1}/health.html | grep 'All systems go.' || exiterror 1 "couldn't GET health.html"
}

sh ${SCRIPTDIR}/build-server.sh

if [ ! -x ./main ]; then
    exiterror 1 "build-server.sh did not create a main executable"
fi

launchenv 1
launchenv 2
launchenv 3


curl -s -v -X PUT --data-binary @LICENSE http://${LOCALHOST}:8001/license.txt 2>&1 | grep ${LICENSEHASH} || exiterror 1 "couldn't PUT license.txt"

curl -s -v -X GET http://${LOCALHOST}:8002/${LICENSEHASH} 2>&1 | grep 'The MIT License' || exiterror 1 "couldn't GET license.txt"

curl -s -v -X GET http://${LOCALHOST}:8003/${LICENSEHASH} 2>&1 | grep 'The MIT License' || exiterror 1 "couldn't GET license.txt"

curl -s -v -X GET http://${LOCALHOST}:8003/404 2>&1 | grep '404 Not Found\.' || exiterror 1 "This is supposed to 404"

exiterror 0
