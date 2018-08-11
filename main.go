package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type macStruct struct {
	MACAddr string `json:"mac"`
}

type state struct {
	System    string
	Power     bool
	Users     uint64
	Uptime    string
	LA        float64
	LA5       float64
	LA15      float64
	Available bool
}

const tpl = `
<!DOCTYPE html>
<html><head><meta charset="UTF-8"><title>Status</title></head>
<style>
.table{margin:auto;display:table;width:500px;padding:70px 0}
.row{display:table-row;}
.cell{display:table-cell;padding:3px 10px;font-size:13px;color:#727272;font-weight:bold}
.cell_header{color:#333}
.cell_power{color:#{{if .Power}}396{{else}}c30{{end}}}
.cell_state{color:#{{if .Available}}396{{else}}c30{{end}}}
</style>
<body><div class="table">
<div class="row"><div class="cell cell_header">Power</div><div class="cell cell_power">{{if .Power}}on{{else}}off{{end}}</div></div>
<div class="row"><div class="cell cell_header">OS</div><div class="cell">{{.System}}</div></div>
{{if eq .System "Linux"}}
<div class="row"><div class="cell cell_header">Uptime</div><div class="cell">{{.Uptime}}</div></div>
<div class="row"><div class="cell cell_header">LoadAvg</div><div class="cell">{{.LA}}<br> {{.LA5}}<br> {{.LA15}}</div></div>
<div class="row"><div class="cell cell_header">Active users</div><div class="cell">{{.Users}}</div></div>
<br>
<div class="row"><div class="cell cell_header">State</div><div class="cell cell_state">{{if .Available}}Available{{else}}Busy{{end}}</div></div>
{{end}}
</div>
</body></html>`

var (
	flagPort  = flag.String("port", "3000", "Port to listen on")
	flagToken = flag.String("token", "02ec48d46e0a7ae83ed4", "Token")
	flagUser  = flag.String("user", "root", "Remote SSH user")
)

func mainHandler(w http.ResponseWriter, r *http.Request) {
	var data macStruct

	if r.Method == "POST" {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return
		}

		err = json.Unmarshal(body, &data)
		if err != nil {
			return
		}

		sendWakeOnLAN(data.MACAddr)
		return
	}

	if r.Method == "GET" {
		keys, ok := r.URL.Query()["ipaddr"]
		if !ok || len(keys[0]) < 1 {
			return
		}
		ipaddr := keys[0]

		st := getStatus(string(ipaddr))

		t, err := template.New("webpage").Parse(tpl)
		if err != nil {
			log.Fatal(err)
		}

		err = t.Execute(w, st)
		if err != nil {
			log.Fatal(err)
		}

		return
	}

}

func sendWakeOnLAN(mac string) {
	if len(mac) != 17 {
		log.Println("MAC format: xx:xx:xx:xx:xx:xx")
		return
	}

	macBytes, err := hex.DecodeString(strings.Join(strings.Split(mac, ":"), ""))
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

func getStatus(ipaddr string) state {
	st := state{Power: false, System: "", Users: 0, Uptime: "0", LA: 0, LA5: 0, LA15: 0, Available: false}

	cmd := exec.Command("ping", ipaddr, "-c 2")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return st
	}

	st.Power = true

	if strings.Contains(out.String(), "ttl=1") || strings.Contains(out.String(), "ttl=2") {
		st.System = "Windows"
		return st
	}

	st.System = "Linux"

	cmd = exec.Command("ssh", *flagUser+"@"+ipaddr, "cat /proc/uptime; cat /proc/loadavg; who -q")
	out.Reset()
	cmd.Stdout = &out
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	timer := time.AfterFunc(20*time.Second, func() {
		cmd.Process.Kill()
	})
	if err := cmd.Wait(); err != nil {
		return st
	}
	timer.Stop()

	data := strings.Fields(out.String())
	uptime, _ := time.ParseDuration(data[0] + "s")
	st.Uptime = uptime.Round(time.Second).String()
	st.LA, _ = strconv.ParseFloat(data[2], 64)
	st.LA5, _ = strconv.ParseFloat(data[3], 64)
	st.LA15, _ = strconv.ParseFloat(data[4], 64)
	st.Users, _ = strconv.ParseUint(strings.Split(data[len(data)-1], "users=")[1], 10, 64)

	if st.Users > 1 {
		st.Available = false
		return st
	}

	if uptime < 1500 {
		st.Available = false
	}

	if (st.LA5 > 1) || (st.LA15 > 1) {
		st.Available = false
	}

	st.Available = true

	return st
}

func main() {
	flag.Parse()

	http.HandleFunc("/"+*flagToken, mainHandler)
	log.Panic(http.ListenAndServe(":"+*flagPort, nil))
}
