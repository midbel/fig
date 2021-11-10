package fig_test

import (
	"fmt"
	"strings"

	"github.com/midbel/fig"
)

func ExampleDecode() {
	const demo = `
contact = "midbel@midbel.org"
admin   = true

ports tcp {
  list    = [80, 443, 22]
  action  = allow
  disable = false
}

ports udp {
  list    = [80, 443, 22]
  action  = block
  disable = false
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
		List    []int
		Action  string
		Disable bool
	}
	type Config struct {
		Email string `fig:"contact"`
		Admin bool
		Ports struct {
			TCP Port
			UDP Port
		}
		Servers []Server `fig:"server"`
	}
	var in Config
	if err := fig.Decode(strings.NewReader(demo), &in); err != nil {
		fmt.Printf("fail to decode input string: %s\n", err)
		return
	}
	fmt.Printf("%#v\n", in)
	// Output:
	// fig_test.Config{Email:"midbel@midbel.org", Admin:true, Servers:[]fig_test.Server{fig_test.Server{Addr:"192.168.67.181", Host:"alpha", Back:[]string{"10.100.0.1", "10.100.0.2"}}, fig_test.Server{Addr:"192.168.67.236", Host:"omega", Back:[]string{"10.101.0.1", "10.101.0.2", "10.101.0.3"}}}}
}
