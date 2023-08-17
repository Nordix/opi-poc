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
	"gopkg.in/yaml.v3"
	"io/ioutil"
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

const ( //Nexthop TYPE & L2NEXTHOP TYPE & FDBentry
	PHY = iota
	SVI
	ACC
	VXLAN
	BRIDGE_PORT
	OTHER = ACC
)

//Subscribers    list[Subscriber] = []

 
var	Routes  map[string]Route
var	Nexthops map[string]Nexthop
var	Neighbors map[string]Neighbor
var	FDB  map[string]FdbEntry
var	L2Nexthops map[string]L2Nexthop

	// Shadow tables for building a new netlink DB snapshot
var	LatestRoutes  map[string]Route 
var	LatestNexthops map[string]Nexthop
var	LatestNeighbors map[string]Neighbor
var	LatestFDB  map[string]FdbEntry
var	LatestL2Nexthops map[string]L2Nexthop

type NetlinkDB struct {
	 


}

type Ri func(int /* V: ipu_db*/, map[string]string)
type Genfunc func()

type Commonfp struct {
	annotate       *Genfunc
	format         *Genfunc
	__str__        *Genfunc
	__eq__         *Genfunc
	__lt__         *Genfunc
	assign_id      *Genfunc
	install_filter *Genfunc
}

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
	Route_init   *Ri
	Route_common Commonfp
}

/*--------------------------------------------------------------------------
###  Nexthop Database Entries
--------------------------------------------------------------------------*/

type Ni func(int /*VRF  from ipu_db*/, dst int, dev int, local bool, weight int, flags []string)

//type Assign_id func()
type Try_resolve func(map[string]string)

type Nexthop struct {
	Nethop_init  *Ni
	try_resolve  *Try_resolve
	Route_common Commonfp
}

/*--------------------------------------------------------------------------
###  Bridge MAC Address Database
###
###  We split the Linux FDB entries into DMAC and L2 Nexthop tables similar
###  to routes and L3 nexthops, Thus, all remote EVPN DMAC entries share a
###  single VXLAN L2 nexthop table entry.
###
###  TODO: Support for dynamically learned MAC addresses on BridgePorts
###  (e.g. for pod interfaces operating in promiscuous mode).
--------------------------------------------------------------------------*/

type Fi func(map[string]string) //FdbEntry_init

type FdbEntry struct {
	FdbEntry_init   *Fi
	Fdbentry_common Commonfp
}

/* L2Nexthop */

type L2NI func(dev int, vlan_id int, dst int, lb int, bp int)

type L2Nexthop struct {
	l2Nexthop_init   *L2NI
	L2Nexthop_common Commonfp
}

/*--------------------------------------------------------------------------
###  Neighbor Database Entries
--------------------------------------------------------------------------*/
type Neigh_init func(int, map[string]string)

type Neighbor struct {
	neigh_init      *Neigh_init
	Neighbor_common Commonfp
}

var wg sync.WaitGroup

func read_latest_netlink_state() {

}

func notify_route_added(R map[string]Route){
	log.Println("Notify: Adding {R.format()}.")

/*for subscriber in cls.Subscribers:
	try:
		subscriber.route_added(R)
	except:
		logger.exception(f"Netlink subscriber {subscriber} failed to add {R}")
*/
}
 
func notify_route_deleted(R map[string]Route){
	log.Println("Notify: Deleting {R.format()}.")

/*for subscriber in cls.Subscribers:
	try:
		subscriber.route_deleted(R)
	except:
		logger.exception(f"Netlink subscriber {subscriber} failed to delete {R}")
*/
}
 
func notify_route_updated(new map[string]Route, old map[string]Route){
	log.Println("Notify: Replacing {old.format()}\n")

/*for subscriber in cls.Subscribers:
	try:
		subscriber.route_updated(new)
	except:
		logger.exception(f"Netlink subscriber {subscriber} failed to update {new}")
*/
}		

 
func notify_nexthop_added(NH map[string]Nexthop){
	log.Println("Notify: Adding {NH.format()}.")

/*for subscriber in cls.Subscribers:
	try:
		subscriber.nexthop_added(NH)
	except:
		logger.exception(f"Netlink subscriber {subscriber} failed to add {NH}")
*/
}
 
func notify_nexthop_deleted( NH map[string]Nexthop){
	log.Println("Notify: Deleting {NH.format()}.")

/*for subscriber in cls.Subscribers:
	try:
		subscriber.nexthop_deleted(NH)
	except:
		logger.exception(f"Netlink subscriber {subscriber} failed to delete {NH}")
*/
}
 
func notify_nexthop_updated(new map[string]Nexthop, old map[string]Nexthop){
	log.Println("Notify: Replacing {old.format()} with {new.format()}.")

/* for subscriber in cls.Subscribers:
	try:
		subscriber.nexthop_updated(new)
	except:
		logger.exception(f"Netlink subscriber {subscriber} failed to update {new}")
*/
}
 
func notify_fdb_entry_added(M map[string]FdbEntry){
	log.Println("Notify: Adding {M.format()}.")

/*for subscriber in cls.Subscribers:
	try:
		subscriber.fdb_entry_added(M)
	except:
		logger.exception(f"Netlink subscriber {subscriber} failed to add {M}")
*/
}

func notify_fdb_entry_deleted(M map[string]FdbEntry){
    log.Println("Notify: Deleting {M.format()}.")

/*for subscriber in cls.Subscribers:
	try:
		subscriber.fdb_entry_deleted(M)
	except:
		logger.exception(f"Netlink subscriber {subscriber} failed to delete {M}")
*/
}
 
func notify_fdb_entry_updated(new map[string]FdbEntry, old map[string]FdbEntry){
    log.Println("Notify: Replacing {old.format()} with {new.format()}.")

/*for subscriber in cls.Subscribers:
	try:
		subscriber.fdb_entry_updated(new)
	except:
		logger.exception(f"Netlink subscriber {subscriber} failed to update {new}")
*/
}
 
func notify_l2_nexthop_added(L2N map[string]L2Nexthop){
    log.Println("Notify: Adding {L2N.format()}.")

/*for subscriber in cls.Subscribers:
	try:
		subscriber.l2_nexthop_added(L2N)
	except:
		logger.exception(f"Netlink subscriber {subscriber} failed to add {L2N}")
*/
}
 
func notify_l2_nexthop_deleted(L2N map[string]L2Nexthop){
    log.Println("Notify: Deleting {L2N.format()}.")

/*for subscriber in cls.Subscribers:
	try:
		subscriber.l2_nexthop_deleted(L2N)
	except:
		logger.exception(f"Netlink subscriber {subscriber} failed to delete {L2N}")
*/
}
 
func notify_l2_nexthop_updated(new map[string]L2Nexthop, old map[string]L2Nexthop){
	log.Println("Notify: Replacing {old.format()} with {new.format()}.")

/*for subscriber in cls.Subscribers:
	try:
		subscriber.l2_nexthop_updated(new)
	except:
		logger.exception(f"Netlink subscriber {subscriber} failed to update {new}")
*/
}

func notify_route_changes(new map[string]Route, old map[string]Route,added_cb func (R map[string]Route), updated_cb func (N map[string]Route, O map[string]Route), deleted_cb func (R map[string]Route)) {
	/*old_keys = Old.keys()
	new_keys = New.keys()
	added = [New[k] for k in (new_keys - old_keys)]
	deleted = [Old[k] for k in (old_keys - new_keys)]
	modified = [(New[k], Old[k]) for k in (new_keys & old_keys) if New[k] != Old[k]]
	for E in added: added_cb(E)
	for E_new, E_old in modified: updated_cb(E_new, E_old)
	for E in deleted: deleted_cb(E)
	*/
}

func notify_nexthop_changes(new map[string]Nexthop,old map[string]Nexthop,added_cb func (N map[string]Nexthop), updated_cb func (N map[string]Nexthop, O map[string]Nexthop), deleted_cb func (N map[string]Nexthop) ){
	/*old_keys = Old.keys()
	new_keys = New.keys()
	added = [New[k] for k in (new_keys - old_keys)]
	deleted = [Old[k] for k in (old_keys - new_keys)]
	modified = [(New[k], Old[k]) for k in (new_keys & old_keys) if New[k] != Old[k]]
	for E in added: added_cb(E)
	for E_new, E_old in modified: updated_cb(E_new, E_old)
	for E in deleted: deleted_cb(E)
	*/
}


func notify_FDB_changes(new map[string]FdbEntry, old map[string]FdbEntry,added_cb func (F map[string]FdbEntry), updated_cb func (N map[string]FdbEntry, O map[string]FdbEntry), deleted_cb func (F map[string]FdbEntry) ){
/*old_keys = Old.keys()
	new_keys = New.keys()
	added = [New[k] for k in (new_keys - old_keys)]
	deleted = [Old[k] for k in (old_keys - new_keys)]
	modified = [(New[k], Old[k]) for k in (new_keys & old_keys) if New[k] != Old[k]]
	for E in added: added_cb(E)
	for E_new, E_old in modified: updated_cb(E_new, E_old)
	for E in deleted: deleted_cb(E)
	*/
}

func notify_L2nexthop_changes(new map[string]L2Nexthop, old map[string]L2Nexthop,added_cb func (L2N map[string]L2Nexthop), updated_cb func (N map[string]L2Nexthop, O map[string]L2Nexthop), deleted_cb func (L2N map[string]L2Nexthop) ){
/*old_keys = Old.keys()
	new_keys = New.keys()
	added = [New[k] for k in (new_keys - old_keys)]
	deleted = [Old[k] for k in (old_keys - new_keys)]
	modified = [(New[k], Old[k]) for k in (new_keys & old_keys) if New[k] != Old[k]]
	for E in added: added_cb(E)
	for E_new, E_old in modified: updated_cb(E_new, E_old)
	for E in deleted: deleted_cb(E)
	*/
}

func annotate_db_entries(LatestRoutes map[string]Route, LatestNexthops map[string]Nexthop , LatestFDB map[string]FdbEntry ,LatestL2Nexthops map[string]L2Nexthop ){
/*	for NH in LatestNexthops.values():
		NH.annotate()
	for R in LatestRoutes.values():
		R.annotate()
	for M in LatestFDB.values():
		M.annotate()
	for L2N in LatestL2Nexthops.values():
		L2N.annotate()
*/
}

func apply_install_filters(LatestRoutes map[string]Route, LatestNexthops map[string]Nexthop , LatestFDB map[string]FdbEntry ,LatestL2Nexthops map[string]L2Nexthop){
/*	for k, R in list(LatestRoutes.items()):
		if not R.install_filter():
			//Remove route from its nexthop(s)
			for NH in R.nexthops:
				NH.route_refs.remove(R)
			LatestRoutes.pop(k)
	for k, NH in list(LatestNexthops.items()):
		if not NH.install_filter():
			LatestNexthops.pop(k)
	for k, M in list(LatestFDB.items()):
		if not M.install_filter():
			LatestFDB.pop(k)
	for k, L2N in list(LatestL2Nexthops.items()):
		if not L2N.install_filter():
			LatestL2Nexthops.pop(k)
*/
}

func resync_with_kernel() {
	// Build a new DB snapshot from netlink and other sources
	read_latest_netlink_state()
	// Annotate the latest DB entries
	annotate_db_entries(LatestRoutes,LatestNexthops,LatestFDB,LatestL2Nexthops)
	//Filter the latest DB to retain only entries to be installed
	apply_install_filters(LatestRoutes,LatestNexthops,LatestFDB,LatestL2Nexthops)
	// Compute changes between current and latest DB versions and
    // inform subscribers about the changes
        notify_route_changes(LatestRoutes, Routes,
                notify_route_added,
                notify_route_updated,
                notify_route_deleted)
        notify_nexthop_changes(LatestNexthops, Nexthops,
                notify_nexthop_added,
                notify_nexthop_updated,
                notify_nexthop_deleted)
        notify_FDB_changes(LatestFDB, FDB,
                notify_fdb_entry_added,
                notify_fdb_entry_updated,
                notify_fdb_entry_deleted)
        notify_L2nexthop_changes(LatestL2Nexthops, L2Nexthops,
                notify_l2_nexthop_added,
                notify_l2_nexthop_updated,
                notify_l2_nexthop_deleted)

}

func monitor_netlink(p4_enabled bool) {
	// Wait for the p4 module to subscribe TBD
	//while p4_enabled and not NetlinkDB.Subscribers:
	//       time.sleep(1)
	for stop_monitoring != true {
		log.Print("netlink: Polling netlink databases.")
		resync_with_kernel()
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
	      try: NetlinkDB.notify_route_deleted(R)
	      except:
	          logger.exception(f"Failed to clean up {str(R)}")
	  for NH in NetlinkDB.Nexthops.values():
	      try: NetlinkDB.notify_nexthop_deleted(NH)
	      except:
	          logger.exception(f"Failed to clean up {str(NH)}")
	  for M in NetlinkDB.FDB.values():
	      try: NetlinkDB.notify_fdb_entry_deleted(M)
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
