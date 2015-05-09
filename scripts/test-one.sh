#!/bin/sh

SCRIPTDIR="./scripts"
LOCALHOST="127.0.0.1"
LOCALPORT="8000"
BASEDIR="${HOME}/z/test-one-base"
TEMPDIR="${HOME}/z/test-one-tmp"
LOGFILE="${HOME}/z/test-one.log"

LICENSEHASH=`openssl dgst -sha512 LICENSE | awk '{print $NF}'`

exiterror() {
    exitcode=$1
    shift
    if [ "0" != "${exitcode}" ]; then
        echo "ERROR: ${@}" >&2
    fi
    exit ${exitcode}
}

sh ${SCRIPTDIR}/build-server.sh

if [ ! -x ./main ]; then
    exiterror 1 "build-server.sh did not create a main executable"
fi

mkdir -p "${BASEDIR}"
mkdir -p "${TEMPDIR}"
touch "${LOGFILE}"

./main --port ${LOCALPORT} --basedir "${BASEDIR}" --temp "${TEMPDIR}" --log "${LOGFILE}" --me "http://${LOCALHOST}:8001/" --peerlist extras/peerlist-one.txt &
TESTPID=$!
echo ${TESTPID}

sleep 1
curl -v -X GET http://${LOCALHOST}:${LOCALPORT}/health.html || exiterror 1 "couldn't GET health.html"

sleep 1
curl -v -X PUT --data-binary @LICENSE http://${LOCALHOST}:${LOCALPORT}/license.txt || exiterror 1 "couldn't PUT license.txt"

echo "---------------------------"

sleep 1
curl -v -X GET http://${LOCALHOST}:${LOCALPORT}/${LICENSEHASH} || exiterror 1 "couldn't GET license.txt"

kill ${TESTPID}
