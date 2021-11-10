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
		Rule    Rule     `fig:"ports"`
		Servers []Server `fig:"server"`
	}
	var in Config
	if err := fig.Decode(strings.NewReader(demo), &in); err != nil {
		fmt.Printf("fail to decode input string: %s\n", err)
		return
	}
	fmt.Printf("%+v\n", in)
	// Output:
	// {Email:midbel@midbel.org Admin:true TTL:100 Rule:{TCP:[{List:[80 443 22] Action:allow Disable:false}] UDP:[{List:[80 443 22] Action:block Disable:false}]} Servers:[{Addr:192.168.67.181 Host:alpha Back:[10.100.0.1 10.100.0.2]} {Addr:192.168.67.236 Host:omega Back:[10.101.0.1 10.101.0.2 10.101.0.3]}]}
}
