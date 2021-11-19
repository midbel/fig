package fig_test

import (
	"fmt"
	"strings"

	"github.com/midbel/fig"
)

func ExampleDecode() {
	const demo = `
# comment are skipped by the parser
contact = "midbel@midbel.org"
admin   = true
TTL     = 100

metadata {
	version = "1.0.1"
	tracker = "redmine"
	vcs     = "git"
}

ports tcp {
  list    = [80, 443, 22]
  action  = allow
  disable = false # set it to true to enable the rule
}

ports udp {
  list    = [80, 443, 22]
  action  = block
  disable = false # set it to true to enable the rule
}

server {
  addr     = "192.168.67.181"
  backup   = "10.100.0.1"
  backup   = "10.100.0.2"
  hostname = "alpha"
}
server {
  addr     = "192.168.67.236"
  backup   = ["10.101.0.1", "10.101.0.2"]
  backup   = "10.101.0.3"
  hostname = "omega"
}
  `
	type Server struct {
		Addr string
		Host string   `fig:"hostname"`
		Back []string `fig:"backup"`
	}
	type Port struct {
		List    []uint16
		Action  string
		Disable bool
	}
	type Rule struct {
		TCP []Port
		UDP []Port
	}
	type Config struct {
		Email   string `fig:"contact"`
		Admin   bool
		TTL     int
		Meta    map[string]interface{} `fig:"metadata"`
		Rule    Rule                   `fig:"ports"`
		Servers []Server               `fig:"server"`
	}
	var in Config
	if err := fig.NewDecoder(strings.NewReader(demo)).Decode(&in); err != nil {
		fmt.Printf("fail to decode input string: %s\n", err)
		return
	}
	fmt.Printf("%+v\n", in)
	// Output:
	// {Email:midbel@midbel.org Admin:true TTL:100 Meta:map[tracker:redmine vcs:git version:1.0.1] Rule:{TCP:[{List:[80 443 22] Action:allow Disable:false}] UDP:[{List:[80 443 22] Action:block Disable:false}]} Servers:[{Addr:192.168.67.181 Host:alpha Back:[10.100.0.1 10.100.0.2]} {Addr:192.168.67.236 Host:omega Back:[10.101.0.1 10.101.0.2 10.101.0.3]}]}
}

func ExampleDecode_Generic() {
	const demo = `
name = demo
server {
	addr = "192.168.67.181"
	name = alpha
	ttl  = 100
	enable = false
}
server {
	addr = "192.168.67.236"
	name = alpha
	ttl  = 100
	enable = true
}
	`

	var empty interface{}
	if err := fig.NewDecoder(strings.NewReader(demo)).Decode(&empty); err != nil {
		fmt.Printf("fail to decode input string into interface{}: %s\n", err)
		return
	}

	data := make(map[string]interface{})
	if err := fig.NewDecoder(strings.NewReader(demo)).Decode(&data); err != nil {
		fmt.Printf("fail to decode input string into map[string]interface{}: %s\n", err)
		return
	}

	fmt.Printf("%+v\n", empty)
	fmt.Printf("%+v\n", data)
	// Output:
	// map[name:demo server:[map[addr:192.168.67.181 enable:false name:alpha ttl:100] map[addr:192.168.67.236 enable:true name:alpha ttl:100]]]
	// map[name:demo server:[map[addr:192.168.67.181 enable:false name:alpha ttl:100] map[addr:192.168.67.236 enable:true name:alpha ttl:100]]]
}

func ExampleDecode_Variables() {
	const demo = `
name = demo
ttl  = 30m
addr = "192.168.67.181"
server {
	addr = $addr
	name = alpha
	ttl  = $ttl
	enable = false
}
server {
	addr = $addr
	name = alpha
	ttl  = $ttl
	enable = true
}
	`

	data := make(map[string]interface{})
	if err := fig.NewDecoder(strings.NewReader(demo)).Decode(&data); err != nil {
		fmt.Printf("fail to decode input string into map[string]interface{}: %s\n", err)
		return
	}

	fmt.Printf("%+v\n", data)
	// Output:
	// map[addr:192.168.67.181 name:demo server:[map[addr:192.168.67.181 enable:false name:alpha ttl:1800] map[addr:192.168.67.181 enable:true name:alpha ttl:1800]] ttl:1800]
}
