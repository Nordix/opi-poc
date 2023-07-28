// Om sai ram

package main  //netlink_monitor

import (
	"fmt"
	"sync"
	"time"
	"log"
//	"net"
//	"github.com/vishvananda/netlink"
        "gopkg.in/yaml.v3"
        "io/ioutil"
)

var  db_lock int
var  GRD int 
var  poll_interval int
var  phy_ports int
var  br_tenant int
var  stop_monitoring int
var  logger int
	

type Config_t struct {
     P4 struct {
       Enable bool `yaml:"enabled"`
     } `yaml: "p4"`
     Linux_frr struct {
  	Enable bool `yaml:"enabled"`
	Default_vtep string `yaml:"default_vtep"`
	Port_mux string `yaml:"port_mux"`
	Vrf_mux string `yaml:"vrf_mux"`
	Br_tenant int `yaml:"br_tenant"`
     } `yaml:"linux_frr"`
     Netlink struct {
	 Enable bool `yaml:"enabled"`	
	 Poll_interval int `yaml:"poll_interval"`
	 Phy_ports []struct {
		Name string `yaml:"name"`
		Vsi  int    `yaml:"vsi"`
	} `yaml:"phy_ports"`
     } `yaml:"netlink"`		 

}

var wg sync.WaitGroup		

func monitor_netlink(p4_enabled bool) {
	fmt.Println(p4_enabled)
	for ;; {
	     time.Sleep(1 * time.Second)
	}       
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
     poll_interval := config.Netlink.Poll_interval
     log.Println(poll_interval)
     br_tenant := config.Linux_frr.Br_tenant
     log.Println(br_tenant)
     nl_enabled := config.Netlink.Enable
     if nl_enabled != true {
        log.Println("netlink_monitor disabled")
        return
     } else {
        log.Println("netlink_monitor Enabled")
     }		
     go monitor_netlink(config.P4.Enable)   //monitor Thread started 
     log.Println("Started netlink_monitor thread with {poll_interval} s poll interval.")
     wg.Wait()	
}

