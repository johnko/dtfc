#!/bin/sh
# Copyright (c) 2015, John Ko

MYUSER="dtfc"
name="dtfc"

SCRIPTDIR="./localscripts"

exiterror() {
    exitcode=$1
    shift
    echo "${@}" >&2
    exit ${exitcode}
}

if [ "FreeBSD" = "`uname -s`" ]; then
    # test variables in this function before using them
    if     [ "x" = "x${MYUSER}" ]; then
    	echo "Variable MYUSER undefined in $0" >&2
    	exit 1
    fi
    if     [ "x" = "x${name}" ]; then
    	echo "Variable name undefined in $0" >&2
    	exit 1
    fi
    pkg install -y nginx sudo go || exit 1
    if [ ! -d /usr/home/${MYUSER} ]; then
    	echo "${MYUSER}:1004::::::/usr/home/${MYUSER}:/bin/sh:" | adduser -w no -f -
    fi
    install -d -m 755 -o ${MYUSER} /opt
    install -m 644 ${SCRIPTDIR}/load-dependencies.sh /opt/load-dependencies.sh
    sudo -u ${MYUSER} -i -- sh /opt/load-dependencies.sh
    install -d -m 755 /usr/local/etc/rc.d
    install -m 755 ./extras/dtfc.rc /usr/local/etc/rc.d/${name}
    chmod a+x /usr/local/etc/rc.d/${name}
    if [ ! -e main ]; then
    	sudo -u ${MYUSER} -- sh ${SCRIPTDIR}/build-server.sh
    fi
    if [ -e main ]; then
    	mv -f main /usr/home/${MYUSER}/${name}-server
    fi
fi

##### For a reverse tls proxy:
# nginx-config-sites-enabled
# install -m 644 /usr/local/etc/nginx/sites-available/example-api /usr/local/etc/nginx/sites-enabled/dtfc
# sysrc nginx_enable="YES"
