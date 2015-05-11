/*
The MIT License (MIT)

Copyright (c) 2015 John Ko

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func fileNotExists(str string) bool {
	var err error
	if _, err = os.Lstat(config.DENY + str); err != nil {
		// denyput not found, so allowed = true
		return true
	}
	return false
}

func allowedPut() bool {
	return fileNotExists("put")
}

func allowedGet() bool {
	return fileNotExists("get")
}

func allowedHead() bool {
	return fileNotExists("head")
}

func allowedDelete() bool {
	return fileNotExists("delete")
}

func refreshPeerList() error {
	newhash, err := Sha512(config.PEERLIST, "")
	if err != nil {
		log.Printf("Error while hashing peerlist. %s", err.Error())
	} else {
		if config.PEERLISTHASH != newhash {
			config.PEERS, err = readLines(config.PEERLIST)
			if err != nil {
				log.Printf("Error while reading peerlist. %s", err.Error())
			} else {
				log.Printf("config.PEERS: %s", config.PEERS)
				config.PEERLISTHASH = newhash
			}
		}
	}
	return err
}

func foundHardLinkSha512Path(oldhash string, oldfile string) (found bool, filename string, reader io.ReadSeeker, modTime time.Time, err error) {
	found = false
	var hash string
	if hash, err = Sha512(oldfile, ""); err != nil {
		log.Printf("%s", err.Error())
		return
	} else {
		// compare oldhash to newhash so we are returning the right data and peer is not corrupt
		if oldhash == hash {
			_, _, err = storage.HardLinkSha512Path(oldfile, filename)
			if err != nil {
				log.Printf("%s", err.Error())
				return
			}
			filename, reader, _, modTime, err = storage.Seeker(hash)
			if err == nil {
				found = true
			}
		}
	}
	return
}

func getFromPeers(oldhash string) (found bool, filename string, reader io.ReadSeeker, modTime time.Time, err error) {
	var file *os.File
	var req *http.Request
	var resp *http.Response
	var currentpeer string
	fnre := regexp.MustCompile("filename=\".*\"")
	found = false
	const rootPEM = `
-----BEGIN CERTIFICATE-----
MIIFBjCCAu4CCQC9q4DpHHr4xzANBgkqhkiG9w0BAQsFADBFMQswCQYDVQQGEwJB
VTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50ZXJuZXQgV2lkZ2l0
cyBQdHkgTHRkMB4XDTE1MDUxMTIwMTYwMFoXDTE2MDUxMDIwMTYwMFowRTELMAkG
A1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoMGEludGVybmV0
IFdpZGdpdHMgUHR5IEx0ZDCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIB
AKFqAQncIZkYA5qshWFeE5bGSJTZkJ5cIItMhg2SYguwTBgto6eMzv3rW/OfvHmZ
IikEVopacYdQRh6uaysl57zJ3hb5WR+nzcKQvAcAvtIH36KUkNEQFxeE2LKUp2oZ
ff3KjWYa9OpwY2fIiOgx07Mm/jLox637rIKGA41DERSZt8SZePn49iTEu15BEDas
fPBi670uYsM+r5WNOnnwV3zPdy4Cxmb+n2MW/N5Ayhrz2+vAi3+Hxclx/6nicUx1
dDsR8krkQONS8LV/NyhFgdtG+jJfMS0aNUG9PklVsYFAeWUimB1wVF8P4Dy8ZRXC
qc0an6Vip+XWMFqXn3hAc1WlW4As5LP39kpCmvEDD7hHBY3WOuuD7wrlKPD99myG
CgR5cxedrauXVp+0DOVwiK8ZGgUzjqLbRhNNsG92DvAwt7z3NWzehIY+lFH8rfEZ
lhjvQWYiVurS8zD0dHmCn484aKEpGfV+hsefG924nld96OZAenjOUszzhznF0SfG
Nkv3ka+hW1+2B6VKLJ4RYyE75z/+gjiQfYQa1J0mwfH4Z6jIVt+oyGhEyzag6ilt
dMFMJZ9EN+NFZLVX6j8yIWeHOP6/NLY/D35myh2g352DetRsI5TP1lPFVPpF2hsX
c09IF8rX67A9BXsgXFyKyOXE1SYB7tbRC/kJETWG4yj1AgMBAAEwDQYJKoZIhvcN
AQELBQADggIBAERdD3O/BZkPvdQFWpPPrDFrC9z/4n/CvW0xSiM56BkeB3tsKrDW
IC675mRv+A9f+Aqk9MgMxw8o7MsFkYbqipmRp8d3uPMLVA3Tw6PxU9SAvd0Qp4s1
Vb1IJs1o+EQQDpiMGr8PM4W6wCwYOOhchkO8EEh1Wahm/c1JOqlzqN0+nZraWcTU
GzzC4+1gd0eHeevExbF1YxHs/O42ugk0r7rdYpYyfszwSG7VEg/bawbyvREAL4Of
Iz7MwfR1KzKqWVf4B2rYykPURUJcrnGaZaRzzKxDKCGZNtHhqbHMu539PhRSdVAp
i6pOa2oGeLHTxvqCtlM1llbHh3dGemTTln421I4yoxjvg67BrXMxXzulj0CD9oSv
NiXOrmARJMh5XzxuHoTpe5FvvAkC2RpQGNHA0Nj3igGg0Yi1QoGYlalkpRWKYmPm
S826JNUZ05HryMpJbHRYf64bDQBIbloYcR5FO3S/oYTPRL5LWHsOZtCtJ8AMC8mF
bqrMAIv4/+dAymz5FF5Az3j9fAaA9bftilJwTJXXllC8fk7Jg+aZ68wMV7/ohh3L
ILvvdsHzr3ZEkATUzAT7WufDproLztag0Zdo5fLp4o4CM9zIypjpjPT+n6TSgwtA
vA4glIN+41258NZRB9Q0YGWZSoGoKwioosdk3IV0buOsmTJ+/lW/BUL0
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIFBjCCAu4CCQDhut8z2k+prjANBgkqhkiG9w0BAQsFADBFMQswCQYDVQQGEwJB
VTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50ZXJuZXQgV2lkZ2l0
cyBQdHkgTHRkMB4XDTE1MDUxMTIwNDAwMloXDTE2MDUxMDIwNDAwMlowRTELMAkG
A1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoMGEludGVybmV0
IFdpZGdpdHMgUHR5IEx0ZDCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIB
AKzFrY7ucHGIsip2lG/hxp65zyUt9mKZ3bCh3a5+0kk77gNgMvTUMkaYn17oxngO
1YrTaHe1QdlEI/VCTeeRXJNzoLlmEDJNnsq3ehPC2bWT/fJif3B6+FJbb4zwfK8l
zidSGQyXoAjq572v8S8Yol9vrcUFeT/jna4vRzZD9Kn0Qc84rlaMAyrvwUpT1tpq
T3I0Yn4fQVxfNIDWPFXhdoqWefJqEEFMz18wL+Jlwr5o9zEk/nJQX/HcXbGZE+SZ
jmUDnEo5VMceCdqexsTCtnr5pVN9CT6znxqVfx5FzmdDjp96tiu2rUteGtftM6fj
1kv8m55ZYv/aQdvrlVNlZw5Cj0+2fbjeBCd3Jyr3NLiVb3RAhRH154ECePC0PIde
KMO+VEHQkpHwNEA/R5kUGlqsZOw1VfVHqXfGZvAcQ7Of4G/jbIfoGUw86fk86RVI
qBACeFXVwvtAEi3ol5LdCLxyilVkEvR+zxl8Dlvc9WCmvhgdUiftS5x3zo4pFc+R
/4qE0EfcWXenw968AODp8Ied46lB0rTCxJ8xYhodX3cwmM1S++1Hk1u8g/VVzv8L
THvMtxkOVWdHlGnx33kr5tlr8hFtgfVA5ppCuE5/PwsVjLV7uAt2bgxz4rATTPlD
Y4GMWpnDdVbCDcM8rqmqDqMWWibYAp2ntNO3CX27cLT9AgMBAAEwDQYJKoZIhvcN
AQELBQADggIBAEZ8ynvB3Lmiv/MwxK9NOEv0awFBmziJNVuYOs25YqtpV9eq3Xms
hzujziHxkzvpmqRTh15d3oplpSkVqjCiAPrr54eg2f2IuZ331sSGu0aeEQmLwVEK
FDatVaaiK6su4Tc2siKtMnRiO0dbCKJ4bn4rkJuQ55QGKL22EwTE9HpCnwRpc79v
fU69wrnjWIVbSzkSOZG5osvUGu/fpp5lrvfcNN9IzkvcgblarUX0+9o/CFiKZsnG
RIKKH7ygU5acdsb3q+pjNYze7ZNl2AxJIzp/nqnKJEURa08QyNJS5gi8cAFxBsar
e89Ne3AkS4zKDt/uw/QN+1kYwfguG0ZYKsnYzAVxHho3pSU9GNMCDMluBwAAxGUP
8JvcGnC/DN+rPlLEP8bUxyxDSAOi5DUZKkwjOUxPqaHR0uaWEZeix6YscgeRFnAO
gDEBlYg65no29O4WaSQQDetEQwhiGJArjVD5UMtDIwk7ntWzfN3Gxd2UcDlaknfa
t53fbR9tx794gqsQePBeT64PDHSHVkJyoW6abq+zKPz664oF9fciF+7KGesKF4i+
9jLKxa8A9dWOj+eO5coWp1Y2XxUgO/H/l1BRd3aUm0bhm6vMb4l8/ySq9wmcVthV
Di/uC5dEGUDExO9DBYGxtwRVr+peNi4h/DHhY996tQoow7GoVK5wpjvO
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIFBjCCAu4CCQC/8GuxE+BeZTANBgkqhkiG9w0BAQsFADBFMQswCQYDVQQGEwJB
VTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50ZXJuZXQgV2lkZ2l0
cyBQdHkgTHRkMB4XDTE1MDUxMTIwNDMyNFoXDTE2MDUxMDIwNDMyNFowRTELMAkG
A1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoMGEludGVybmV0
IFdpZGdpdHMgUHR5IEx0ZDCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIB
AOT8vvGHqfrN3Q9gXoJzBcQUbfcj5xR+sU2+3TyxdN/kNVaW4L6sJAVhMJpUmbCB
d4P6a07vTsqw4XfMP0k4mYLgDPYO3j+BgAWEKy+KehoGLKquMya7ZTQCrMGktot2
/3d4VS/g3eiT4339Rv5VNcRrY5uFiE78CqlS0O+5gm3ae3iLm7XgCgL/XC/xXKkp
gCw3Iw964hRjqH95h/neJq3sL7apRgpqUuRFURqj4NT0kz4R4WA63RuKnKGFm088
W1J3wmTsmpyiuOlLyALMcPtVpkxFsRd6Nb8hEl9YBlUO2kE7mbX5ATg9UegXm979
SoPZZOaYw5f95IuOk88dsG1X5WYZ1vbObAFlVKpHMW6RiDmoiz4r64DqKo5vDR2e
+jGaRuNVy8qTSVFLk55PMXzMO28iTWG2hj+EnNOL7dNLwhHkcNPGasmTGUNTnbX3
DltyHgwBU5u9V9lgPMO8KGFmAIZI2Lwkrp7APR2PWLp4sLUz0DzwAJRe4/kwBlCQ
xAhZSDEitdO8jhEXRtVgz7Q52U7O/EIYPkbSyOrYF0vkDzSR3KpbBXka/JGpT+3t
XPx70lcob3c1GrF0vAyvDFXBdnFxe9zLm1JCoodmU+MU46YYwXJMSDnu+ko2WkmL
xGyeJgLB39B3294bmzhOmA1NYiY/zJrAxLUXlNfgIRVrAgMBAAEwDQYJKoZIhvcN
AQELBQADggIBALeDB9w8on/o8AG2GCQEp9BlOPCla2ZiT06mPz590UdruaHxoZBL
nOQrxnPBvk6UjyLhqCp9Iz6WlW/+v4jHPJVGP964wj6npzlXwFVzEjj1J5a3U77C
l1xuWtjQFw4T6d5qwH+iObZIZse4AQN9jZmNbZtZRtsMrjWKWjBm41x8zGXnLg7h
CObuG8AMP5CKPHGcru+WQdn5sflzDFV82QjAANtXR0C1/zXd6wIDS3uLQoLfeKs6
UxHOKxUcg2p2Gi3UjpSTjJlAsftNyurT612EfjlshiXN7GK2jG2FJ8Qfo9ua6RMx
cQXCPb88muozufO3CzdIgMJZ0LPIUptyr0JE5GTR2gPEMYdKaQY+v6q/WR1DpHXW
NECmNKhIMf7BaiuYlOCsIxuB1Bg5oK5U4hURs9p0P5a4KL8aGSnqMoQvtCRsxFcY
2jve3HGkpbr/yg+0I04f6SMdn8cwFvRw31nes9zvFz1AqV+j9SZzqoXdh6/b+qAa
sa8k0Rtn/GWKM6Uz8Mc7kuFzavMKoFgrfM6qkdtosa3LiEnJMR6KdR0cqAvFR/SD
swhZXn2dmPGpdVz4kjLs82eytDzkWefo093bOlrROYYqCYMZO3MA9jzg2TtRQCv5
ZraXXJTXbIBuWOeSFOuIU42qt0KFF94/s2vuneX2PBqMhpazFe/kGwub
-----END CERTIFICATE-----`
	// from golang example
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(rootPEM))
	if !ok {
		panic("failed to parse root certificate")
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: roots},
	}
	// end golang example
	client := &http.Client{Transport: tr}
	tmphash := filepath.Join(config.Temp, oldhash)
	for i := range config.PEERS {
		currentpeer = strings.Trim(config.PEERS[i], "")
		if (currentpeer != config.ME) && (currentpeer != "") && (found == false) {
			var url = currentpeer + oldhash
			log.Printf("trying to get from peer %s", url)
			// if tmp file exists, means last download was incomplete
			if _, err = os.Lstat(tmphash); err == nil {
				// file found, continue download with curl
				cmd := exec.Command(cmdCURL, "--continue-at", "-", "--output", tmphash, url)
				err = cmd.Run()
				if err == nil {
					found, filename, reader, modTime, err = foundHardLinkSha512Path(oldhash, tmphash)
				}
			} else {
				file, err = os.OpenFile(tmphash, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
				if err != nil {
					log.Printf("%s", err.Error())
				} else {
					defer file.Close()
					req, err = http.NewRequest("GET", url, nil)
					if err != nil {
						log.Printf("%s", err.Error())
					} else {
						// set user agent
						req.Header.Set("User-Agent", SERVER_INFO+"/"+SERVER_VERSION)
						resp, err = client.Do(req)
						if err == nil {
							if resp.StatusCode == 200 {
								// get filename
								if fnre.MatchString(resp.Header.Get("Content-Disposition")) {
									filename = strings.Replace(
										strings.Replace(
											fnre.FindString(
												resp.Header.Get("Content-Disposition")),
											"filename=",
											"",
											-1),
										"\"",
										"",
										-1)
									// ssave filename early
									storage.saveFilename(oldhash, filename)
								}
								defer resp.Body.Close()
								// save file
								_, err = io.Copy(file, resp.Body)
								if err != nil {
									// download interrupted
									log.Printf("%s", err.Error())
								} else {
									// go through hash and hardlink process
									found, filename, reader, modTime, err = foundHardLinkSha512Path(oldhash, file.Name())
								}
							}
						}
					}
				}
			}
		}
	}
	return
}
