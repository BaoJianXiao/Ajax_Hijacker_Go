package main

import (
	"bufio"
	"crypto/tls"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

type Cfg struct {
	Port   *string
	Raddr  *string
	Static *string
}

type HandlerWrapper struct {
	MyConfig        *Cfg
	wrapped         http.Handler
	pkPem           []byte
	issuingCertPem  []byte
	serverTLSConfig *tls.Config
	certMutex       sync.Mutex
	https           bool
}

func InitConfig(conf *Cfg) *HandlerWrapper {
	hw := &HandlerWrapper{MyConfig: conf}
	return hw
}

func (hw *HandlerWrapper) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	raddr := *hw.MyConfig.Raddr
	path := req.URL.Path
	if strings.LastIndex(path, ".html") > 0 ||
		strings.LastIndex(path, ".css") > 0 ||
		strings.LastIndex(path, ".js") > 0 ||
		strings.LastIndex(path, ".jpg") > 0 ||
		strings.LastIndex(path, ".png") > 0 ||
		strings.LastIndex(path, ".gif") > 0 ||
		strings.LastIndex(path, ".map") > 0 {
		fin, err := os.Open(*hw.MyConfig.Static + path)
		defer fin.Close()
		if err != nil {
			hw.Forward(resp, req, raddr)
		}
		fd, _ := ioutil.ReadAll(fin)
		resp.Write(fd)
	} else {
		hw.Forward(resp, req, raddr)
	}
}

func (hw *HandlerWrapper) Forward(resp http.ResponseWriter, req *http.Request, raddr string) {
	connIn, _, err := resp.(http.Hijacker).Hijack()
	connOut, err := net.Dial("tcp", raddr)
	if err != nil {
		logger.Println("dial tcp error", err)
	}

	err = connectProxyServer(connOut, raddr)
	if err != nil {
		logger.Println("connectProxyServer error:", err)
	}

	if req.Method == "CONNECT" {
		b := []byte("HTTP/1.1 200 Connection Established\r\n" +
			"Proxy-Agent: html5_Dev_KevinBao/" + Version + "\r\n\r\n")
		_, err := connIn.Write(b)
		if err != nil {
			logger.Println("Write Connect err:", err)
			return
		}
	} else {
		req.Header.Del("Proxy-Connection")
		req.Header.Set("Connection", "Keep-Alive")
		req.Header.Set("Origin", "http://"+raddr)
		if err = req.Write(connOut); err != nil {
			logger.Println("send to server err", err)
			return
		}
	}
	err = Transport(connIn, connOut)
	if err != nil {
		log.Println("trans error ", err)
	}
}
func connectProxyServer(conn net.Conn, addr string) error {

	req := &http.Request{
		Method:     "CONNECT",
		URL:        &url.URL{Host: addr},
		Host:       addr,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
	}
	req.Header.Set("Proxy-Connection", "keep-alive")
	req.Header.Set("Origin", "http://"+addr)
	if err := req.Write(conn); err != nil {
		return err
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}
	return nil
}
func Transport(conn1, conn2 net.Conn) (err error) {
	rChan := make(chan error, 1)
	wChan := make(chan error, 1)

	go MyCopy(conn1, conn2, wChan)
	go MyCopy(conn2, conn1, rChan)

	select {
	case err = <-wChan:
	case err = <-rChan:
	}

	return
}

func MyCopy(src io.Reader, dst io.Writer, ch chan<- error) {
	_, err := io.Copy(dst, src)
	ch <- err
}
