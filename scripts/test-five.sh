#!/bin/sh

SCRIPTDIR="./scripts"
LOCALHOST="127.0.0.1"
BASEDIR="${HOME}/z/test-five-base"
TEMPDIR="${HOME}/z/test-five-tmp"
LOGFILE="${HOME}/z/test-five-log"
PIDFILE="${HOME}/z/test-five-pid"

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
    kill `cat ${PIDFILE}4`
    kill `cat ${PIDFILE}5`
    exit ${exitcode}
}

launchenv() {
    mkdir -p "${BASEDIR}${1}"
    mkdir -p "${TEMPDIR}${1}"
    touch "${LOGFILE}${1}"
    #./main --port 800${1} --basedir "${BASEDIR}${1}" --temp "${TEMPDIR}${1}" --log "${LOGFILE}${1}" --me "http://${LOCALHOST}:800${1}/" --peerlist extras/peerlist-five.txt &
    ./main --port 800${1} --basedir "${BASEDIR}${1}" --temp "${TEMPDIR}${1}" --log "${LOGFILE}${1}" --melist extras/melist-${1}.txt --peerlist extras/peerlist-five.txt &
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
launchenv 4
launchenv 5


curl -s -v -X PUT --data-binary @LICENSE http://${LOCALHOST}:8001/license.txt 2>&1 | grep ${LICENSEHASH} || exiterror 1 "couldn't PUT license.txt"

curl -s -v -X GET http://${LOCALHOST}:8002/${LICENSEHASH} 2>&1 | grep 'The MIT License' || exiterror 1 "couldn't GET license.txt"

curl -s -v -X GET http://${LOCALHOST}:8003/${LICENSEHASH} 2>&1 | grep 'The MIT License' || exiterror 1 "couldn't GET license.txt"

curl -s -v -X GET http://${LOCALHOST}:8004/${LICENSEHASH} 2>&1 | grep 'The MIT License' || exiterror 1 "couldn't GET license.txt"

curl -s -v -X GET http://${LOCALHOST}:8005/${LICENSEHASH} 2>&1 | grep 'The MIT License' || exiterror 1 "couldn't GET license.txt"

curl -s -v -X GET http://${LOCALHOST}:8005/404404404404404404404404404 2>&1 | grep '404 Not Found\.' || exiterror 1 "This is supposed to 404"

curl -s -v -X HEAD http://${LOCALHOST}:8005/${LICENSEHASH}

curl -s -v -X HEAD http://${LOCALHOST}:8005/404404404404404404404404404

exiterror 0
