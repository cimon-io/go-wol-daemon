package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
)

type macStruct struct {
	MACAddr string `json:"mac"`
}

var (
	flagPort  = flag.String("port", "3000", "Port to listen on")
	flagToken = flag.String("token", "02ec48d46e0a7ae83ed4", "Token")
)

func mainHandler(w http.ResponseWriter, r *http.Request) {
	var data macStruct

	if r.Method != "POST" {
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return
	}

	if len(data.MACAddr) != 17 {
		log.Println("MAC format: xx:xx:xx:xx:xx:xx")
		return
	}

	macBytes, err := hex.DecodeString(strings.Join(strings.Split(data.MACAddr, ":"), ""))
	if err != nil {
		return
	}

	b := []uint8{255, 255, 255, 255, 255, 255}
	for i := 0; i < 16; i++ {
		b = append(b, macBytes...)
	}

	a, err := net.ResolveUDPAddr("udp", net.JoinHostPort("255.255.255.255", "9"))
	if err != nil {
		return
	}

	c, err := net.DialUDP("udp", nil, a)
	if err != nil {
		return
	}

	written, err := c.Write(b)
	c.Close()

	if written != 102 {
		return
	}
}

func main() {
	flag.Parse()

	http.HandleFunc("/"+*flagToken, mainHandler)
	log.Panic(http.ListenAndServe(":"+*flagPort, nil))
}
