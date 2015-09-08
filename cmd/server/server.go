package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net"
	"net/http"

	"github.com/zond/hackyhack/server"
	"github.com/zond/hackyhack/server/persist"
)

const (
	certPem = `-----BEGIN CERTIFICATE-----
MIIC+zCCAeOgAwIBAgIJAJFppcTeNOxCMA0GCSqGSIb3DQEBBQUAMBQxEjAQBgNV
BAMMCWhhY2t5aGFjazAeFw0xNTA5MDUxOTE2MDJaFw0yNTA5MDIxOTE2MDJaMBQx
EjAQBgNVBAMMCWhhY2t5aGFjazCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoC
ggEBALmTt737gE0RlDfrdlquYmXdY5FkOwLsz/znx1ag8G0SHkuUqnEDoTZeQdZW
ISxsLTr2BT+333ZVJ5fdbjhxeyI5dB2dSElAGxI4LVikFtlYXlHlDerITN5iSWa9
GVTEWC0wOgg9MiLf3Be7tXcbEtcxMT1suzF48C2LuHi4JHQHpU6qpEmhKbovrXma
4CETCIzCHq/ZCfgz+uLisBx6/JiJieJFVg45mvcBnoVL8g5a0Y5IaWXhK3ZE/1iB
DGch4Yh0CBtoKVZCZlZzVNdTg1nWGlk3ZCUvfyRE83mS7q9q8n2FNBy1d2eFRN6T
ZLNg3XqZ01R2qJWuDAgkBF6DkIECAwEAAaNQME4wHQYDVR0OBBYEFFRWfVG5uMRC
6QfjpPJwAVtKLwZrMB8GA1UdIwQYMBaAFFRWfVG5uMRC6QfjpPJwAVtKLwZrMAwG
A1UdEwQFMAMBAf8wDQYJKoZIhvcNAQEFBQADggEBACETV5L8nGNi58axvbGOZ+Qg
78pQcdoAtwZfCQv5rEfwGV/pZm69dwpg+aEh2051FX/IcTfPZJZUyYCF6xOY22G3
sLzn0xAu1RZzTvXFRhPf/nLcZaNwyP6/W102W8bz66TI8y9DuvhJGJ0A6i7fI4lU
DM3D1uwNjAx+AZr2dyiFB+stYUeeE5S6HL1Zh5sL4N4DUe3jB2I6lJleSz1Ig9Rg
Jj899EFWN/lIP6wIU1beHERaW1GZtHOy88+aNde9cBtEjw674iUrNFn5KPMUdlnK
tNSoF6zm/WFN5E3V5Oo6eP+ye5U6BRP2ltRbyuEMBlCIYCkV1dYEs68iViLrgQA=
-----END CERTIFICATE-----`
	keyPem = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAuZO3vfuATRGUN+t2Wq5iZd1jkWQ7AuzP/OfHVqDwbRIeS5Sq
cQOhNl5B1lYhLGwtOvYFP7ffdlUnl91uOHF7Ijl0HZ1ISUAbEjgtWKQW2VheUeUN
6shM3mJJZr0ZVMRYLTA6CD0yIt/cF7u1dxsS1zExPWy7MXjwLYu4eLgkdAelTqqk
SaEpui+teZrgIRMIjMIer9kJ+DP64uKwHHr8mImJ4kVWDjma9wGehUvyDlrRjkhp
ZeErdkT/WIEMZyHhiHQIG2gpVkJmVnNU11ODWdYaWTdkJS9/JETzeZLur2ryfYU0
HLV3Z4VE3pNks2DdepnTVHaola4MCCQEXoOQgQIDAQABAoIBAB2QrCBHVjxxBYUX
LUbrK2ABMmCycDhaFBS9tGNXxpYJ4eu2pqTUqDVqNOD53dUe8uHG2jU5jQ9kJ6ep
LmstoSllr9sb+K062lU/v/G0SrObwYMYk+wItz5iuED29XcsxMOGQGiZn0gxE/Zw
AEwWcxz3iFm53eTW2KTY8q3A4IXfgEtiolmLcXGOPb8V0o9nS+PvNarPTHyLnYOU
oto6cbmBGxjrTjVVJ293zvilqr2NQT4ZfAXOwr+wecERGMvYjEPrFRyPRAVe5MF4
UFD5bxnqK35+X0i7Z8HDHXh1MK1XSV/PaovD73z4llwvQIdnc2Nic02g+/cGFx7D
Kw9llLECgYEA68fCadhLn3ysIzoYwCPXR4yv8IMly/sIlvHq/u07y55naF67wqFf
kU5nIQY+PszjVqGwKNzLRfNK/9UQ0AkOLPS50UCmD3RBxKAQkiTKUe8TK25vmhaj
09omKvaA+HWO+K7b/GOEQta5iNjua4ORo2hOBL92VMO5YHR/XB/zOkUCgYEAyX3P
/Fte1Tf0r8D0BeuVXNXrvqm40zz9PqS8MvcqTXHQOT9RAhAD0sU8QQWfA2Cnl0ie
xWUyB8cbzG15KfrXXLDGCBTiAp+bnIezT04jSpbW5/ABldqZF+dDjj3QJndBS04d
ThrD0jLjJEgAaSXObD2mMM1WmCENQkwEVWgCXw0CgYEAtCy9iybHe0PJQ04lFccN
vtZqqG9/1aWqxbZuboqZNBuDSAWEk9G/dwmj01+y90iYvV3ngQJgr76gZGnMZD1X
QNFuodI2U/7yNzBeGV/V39DDJGBLFkQQw1aj7hbbLYKgU7dD0lW1/2GY/FNRtoUf
KPEPFZ+97D55DZVYseyUcMUCgYArRSN3NEAHVf7sB2ngI5lt2FrKFTSl2IEiBMqN
v1qMSxbGVHyXDs1jZAvugsCFPyp+aJAAIB1AYlfr7M6KX14Ef8nnTmTC33fRg6rU
KxmVGROJt5b/kXQzF+0ADPI4cH/LJjlQ3pqS926kCfpcmkvcHtkjvdUM0nxAcoaz
uKRZuQKBgAOpkGj5iFAz7vHBq8quqCkn0Ejuneh1LzceRxmYsC4oQJ4TY/Q8VeWh
gCxqcD0Nv9qCbbeW/r9oCMSV/s24UNje+WphPf/VPSD75+te1QFW6rq4woXXtNDC
fauKTZKIO5xJUx0HYLL1diaGtTT08pvnNW6lnZvaowluNJ2ztNhd
-----END RSA PRIVATE KEY-----`
)

func main() {
	socketAddr := flag.String("loginAddr", ":6000", "Where to listen for sockets")
	httpAddr := flag.String("httpAddr", ":8080", "Where to listen for http")

	flag.Parse()

	cert, err := tls.X509KeyPair([]byte(certPem), []byte(keyPem))
	if err != nil {
		panic(err)
	}

	s, err := server.New(&persist.Persister{
		Backend: persist.NewMem(),
	})

	httpServer := &http.Server{
		Addr:    *httpAddr,
		Handler: s,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{
				cert,
			},
		},
	}

	socketListener, err := net.Listen("tcp", *socketAddr)
	if err != nil {
		panic(err)
	}

	if err != nil {
		log.Fatal(err)
	}
	go func() {
		log.Fatal(s.ServeLogin(socketListener))
	}()

	log.Fatal(httpServer.ListenAndServeTLS("", ""))
}
