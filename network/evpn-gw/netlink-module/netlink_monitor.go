package main //netlink_monitor

import (
	//	"fmt"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"

	//	"net"
	//	"github.com/vishvananda/netlink"
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

var db_lock int
var GRD int
var poll_interval int
var phy_ports int
var br_tenant int
var stop_monitoring bool
var logger int

type Config_t struct {
	P4 struct {
		Enable bool `yaml:"enabled"`
	} `yaml: "p4"`
	Linux_frr struct {
		Enable       bool   `yaml:"enabled"`
		Default_vtep string `yaml:"default_vtep"`
		Port_mux     string `yaml:"port_mux"`
		Vrf_mux      string `yaml:"vrf_mux"`
		Br_tenant    int    `yaml:"br_tenant"`
	} `yaml:"linux_frr"`
	Netlink struct {
		Enable        bool `yaml:"enabled"`
		Poll_interval int  `yaml:"poll_interval"`
		Phy_ports     []struct {
			Name string `yaml:"name"`
			Vsi  int    `yaml:"vsi"`
		} `yaml:"phy_ports"`
	} `yaml:"netlink"`
}

type Direction int

func run(cmd string) {
	log.Println(cmd)
	//log.Println("Executing command {cmd}")
	out, err := exec.Command(cmd, "-rin", " monitor_netlink").Output()
	if err != nil {
		log.Println("%s", err)
	}
	output := string(out[:])
	fmt.Println(output)
}

const ( // Route direction
	None_ Direction = iota
	RX
	TX
	RX_TX
)

/*--------------------------------------------------------------------------
###  Route Database Entries
###
###  In the internal Route table, there is one entry per VRF and IP prefix
###  to be installed in the routing table of the P4 pipeline. If there are
###  multiple routes in the Linux  route database for the same VRF and
###  prefix, we pick the one with the lowest metric (as does the Linux
###  forwarding plane).
###  The key of the internal Route table consists of (vrf, dst prefix) and
###  corresponds to the match fields in the P4 routing table. The rx/tx
###  direction match field of the MEV P4 pipeline and the necessary
###  duplication of some route entries is a technicality the MEV P4 pipeline
###  and must be handled by the p4ctrl module.
--------------------------------------------------------------------------*/

type Route struct {
	//RX

}

var wg sync.WaitGroup

func read_latest_netlink_state() {

}

func resync_with_kernel() {
	// Build a new DB snapshot from netlink and other sources
	read_latest_netlink_state()
	// Annotate the latest DB entries
	annotate_db_entries()
	//Filter the latest DB to retain only entries to be installed
	apply_install_filters()

}

func monitor_netlink(p4_enabled bool) {
	// Wait for the p4 module to subscribe TBD
	//while p4_enabled and not NetlinkDB.Subscribers:
	//       time.sleep(1)
	for stop_monitoring != true {
		log.Print("netlink: Polling netlink databases.")
		//   NetlinkDB.resync_with_kernel()
		log.Print("netlink: Polling netlink databases completed.")
		time.Sleep(time.Duration(poll_interval) * time.Second)
	}
	log.Println("netlink: Stopped periodic polling. Waiting for Infra DB cleanup to finish.")
	time.Sleep(2 * time.Second)
	log.Println("netlink: One final netlink poll to identify what's still left.")
	resync_with_kernel()
	// Inform subscribers to delete configuration for any still remaining Netlink DB objects.
	log.Println("netlink: Delete any residual objects in DB")
	/*for R in NetlinkDB.Routes.values():
	      try: NetlinkDB._notify_route_deleted(R)
	      except:
	          logger.exception(f"Failed to clean up {str(R)}")
	  for NH in NetlinkDB.Nexthops.values():
	      try: NetlinkDB._notify_nexthop_deleted(NH)
	      except:
	          logger.exception(f"Failed to clean up {str(NH)}")
	  for M in NetlinkDB.FDB.values():
	      try: NetlinkDB._notify_fdb_entry_deleted(M)
	      except:
	          logger.exception(f"Failed to clean up {str(M)}")
	*/
	log.Println("netlink: DB cleanup completed.")
	run("grep")
	wg.Done()
}

func main() {

	var config Config_t
	wg.Add(1)
	yfile, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Fatal(err)
	}
	err2 := yaml.Unmarshal(yfile, &config)
	if err2 != nil {
		log.Fatal(err2)
	}
	poll_interval = config.Netlink.Poll_interval
	log.Println(poll_interval)
	br_tenant = config.Linux_frr.Br_tenant
	log.Println(br_tenant)
	nl_enabled := config.Netlink.Enable
	if nl_enabled != true {
		log.Println("netlink_monitor disabled")
		return
	} else {
		log.Println("netlink_monitor Enabled")
	}
	go monitor_netlink(config.P4.Enable) //monitor Thread started
	log.Println("Started netlink_monitor thread with {poll_interval} s poll interval.")
	time.Sleep(5 * time.Second)
	stop_monitoring = true
	wg.Wait()
}
