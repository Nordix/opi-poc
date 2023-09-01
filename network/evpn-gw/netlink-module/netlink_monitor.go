package main //netlink_monitor

import (
	//	"fmt"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"
	"strings"
	"regexp"
	"strconv"	
//	"reflect"
	//"net"
	netlink "github.com/vishvananda/netlink"
	//"github.com/vishvananda/netns"
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


func run(cmd1 string, cmd2 string, cmd3 string) string {
	//log.Println(cmd)
	//log.Println("Executing command {cmd}")
	out, err := exec.Command(cmd1,cmd2,cmd3).Output()
	if err != nil {
		log.Println("%s", err)
	}
	output := string(out[:])
	return output
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
type Neigh_key struct {
	dst string
	VRF_name string
	dev  int 
}


type Route_key struct {
	Table int
	Dst string
}

type Nexthop_key struct {
	dst string
	VRF_name string
	dev  int 
	local int
}

type FDB_key struct {
	vlan_id int
	mac string
}

type L2Nexthop_key struct {
	dev int
	vlan_id int
	dst string
}
var	Routes = make( map[Route_key]Route_value)
var	Nexthops = make(map[Nexthop_key]Nexthop)
var	Neighbors = make(map[Neigh_key]Neigh_value)
var	FDB  = make(map[FDB_key]FdbEntry)
var	L2Nexthops = make( map[L2Nexthop_key]L2Nexthop)


	// Shadow tables for building a new netlink DB snapshot
var	LatestRoutes =  make(map[Route_key]Route_value)
var	LatestNexthops = make(map[Nexthop_key]Nexthop)
var     LatestNeighbors = make(map[Neigh_key]Neigh_value)
var	LatestFDB  =  make(map[FDB_key]FdbEntry)
var	LatestL2Nexthops = make(map[L2Nexthop_key]L2Nexthop)

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

//type Commonfp interface {
//}

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
	//log.Println("In read_latest_netlink_state function \n")
	// LatestRoutes = nil
	// LatestNexthops = nil
	// LatestNeighbors = nil
	// LatestFDB = nil
	// LatestL2Nexthops = nil
	read_neighbors()
	fmt.Println("\n\n\n")
	read_routes()
}

func ensureIndex(link *netlink.LinkAttrs) {
	if link != nil && link.Index == 0 {
			newlink, _ := netlink.LinkByName(link.Name)
			if newlink != nil {
					link.Index = newlink.Attrs().Index
			}
	}
}


type dump_entries interface {
	 add_neigh(string)
	 add_route()
}

type Neigh_Struct struct {
	 Neigh0 []netlink.Neigh
	 Err error
}


type Route_Struct struct {
	Route0 []netlink.Route
	Err error 
}

type Neigh_value struct {
	Neigh_val netlink.Neigh
	Name       string 
}

type Route_value struct {
	Route_val netlink.Route
	Name       string 
}


func Check_Ndup(tmp_key Neigh_key) bool {
	var dup = false
//	fmt.Printf("dup %d", dup)
	for k,_ := range LatestNeighbors {
			if k == tmp_key{
		   	dup = true	
		   	break
			}
	}  
	return dup		
}

func Check_Rdup(tmp_key Route_key) bool {
	var dup = false
	for j,_ := range LatestRoutes {
			if j == tmp_key{
		   	dup = true	
		   	break
			}
	}
	return dup		
}

func (dump Neigh_Struct ) add_neigh (str string){
	 
	for _, n := range dump.Neigh0 {
		if (n.State !=  netlink.NUD_NONE && n.State !=  netlink.NUD_INCOMPLETE && n.State !=  netlink.NUD_FAILED && n.State !=  netlink.NUD_NOARP) { 
			temp_key := Neigh_key{dst :n.IP.String(),VRF_name: "",dev: n.LinkIndex}
			temp_val := Neigh_value{Neigh_val : n , Name : str}
			if(len(LatestNeighbors)==0) {
				LatestNeighbors[temp_key]= temp_val
			} else {
				if !Check_Ndup(temp_key){
				   LatestNeighbors[temp_key]= temp_val
				}   
			}	
		}
	   }
}


func dump_neighDB() {
	neigh_state := map[int]string{
		netlink.NUD_NONE      : "NONE" ,
		netlink.NUD_INCOMPLETE: "INCOMPLETE",
		netlink.NUD_REACHABLE : "REACHABLE",
		netlink.NUD_STALE     : "STALE",
		netlink.NUD_DELAY     : "DELAY",
		netlink.NUD_PROBE     : "PROBE",
		netlink.NUD_FAILED    : "FAILED",
		netlink.NUD_NOARP     : "NOARP",
		netlink.NUD_PERMANENT : "PERMANENT",
	}
	for _, n := range LatestNeighbors {
		str := fmt.Sprintf("Linkindex:%[2]d, Family:%[2]d Type:%[2]d Flags:%[2]d Vlan:%[2]d VNI:%[2]d MasterIndex:%[2]d State: "+neigh_state[n.Neigh_val.State]+"  Dev : "+n.Name+"  Mac Address: "+ n.Neigh_val.HardwareAddr.String()+"  LLIPAddr: "+n.Neigh_val.LLIPAddr.String()+" IP Address: "+n.Neigh_val.IP.String(),n.Neigh_val.LinkIndex,n.Neigh_val.Family,n.Neigh_val.Type,n.Neigh_val.Flags,n.Neigh_val.Vlan,n.Neigh_val.VNI,n.Neigh_val.MasterIndex,n.Name)
		fmt.Println(str)
			
    }
}


func dump_RouteDB() {
	for _, n := range LatestRoutes {
		str := fmt.Sprintf("Linkindex:%d, ILinkIndex:%d Scope:%d Dst: %s SCR :%s  GW: %s Protocol: %d Priority: %d  Table:%d Type: %d Tos: %d Flags: %d MTU: %d AdvMSS: %d HopLimit: %d\n\n",
				 	    n.Route_val.LinkIndex,n.Route_val.ILinkIndex,n.Route_val.Scope,n.Route_val.Dst.String(),n.Route_val.Src.String(),n.Route_val.Gw.String(), n.Route_val.Protocol,n.Route_val.Priority,n.Route_val.Table,n.Route_val.Type,n.Route_val.Tos,n.Route_val.Flags,n.Route_val.MTU,n.Route_val.AdvMSS,n.Route_val.Hoplimit)
		fmt.Println(str)
	}
		fmt.Printf("\n\n")	
}


func add_nexthop(n netlink.Route){
	if len(n.MultiPath) > 0 {
		//for  ( i = 0 ; nh:=n.MultiPath[i]; i++) {
				// Hear need to fill the next hop data base  once the multipath = nil issue is fixed 
		//}
	}
}

func (dump Route_Struct) add_route(str string ){
//	 var i int
	for _, n := range dump.Route0 {
		//str := fmt.Sprintf("Linkindex:%d, ILinkIndex:%d Scope:%d Dst: %s SCR :%s  GW: %s Protocol: %d Priority: %d  Table:%d Type: %d Tos: %d Flags: %d MTU: %d AdvMSS: %d HopLimit: %d\n\n",
	    //	    n.LinkIndex,n.ILinkIndex,n.Scope,n.Dst.String(),n.Src.String(),n.Gw.String(), n.Protocol,n.Priority,n.Table,n.Type,n.Tos,n.Flags,n.MTU,n.AdvMSS,n.Hoplimit)
			temp_key := Route_key{Table:n.Table, Dst : n.Dst.String()}
			temp_val := Route_value{Route_val: n, Name : str}
			if(len(LatestRoutes)==0) {
				LatestRoutes[temp_key]= temp_val
			} else {
				if !Check_Rdup(temp_key){
					LatestRoutes[temp_key]= temp_val
				}		
			}
			//add_nexthop(n)   
		}	
}

var vrf_table = make(map[string]int)

func get_vrf_table() {
	var j int
	re := regexp.MustCompile("[0-9]+")
	vrf_list := run("ip","vrf","show")
	res1:=strings.Split(vrf_list,"\n")
	
	for i:=0; i< len(res1) ; i++ {
		if (i>1){
			res2:=strings.Split(res1[i]," ")
 			str1 := re.FindAllString(res1[i], -1)
			if (str1 != nil){	
				j ,_ = strconv.Atoi(str1[0])
		 		vrf_table[res2[0]] = j
			}	
		}
	}
}	

func read_neighbors() {

	link_int := []string{"lo","enp0s1f0", "enp0s1f0d1","enp0s1f0d2","enp0s1f0d3","enp0s1f0d4","enp0s1f0d5","rep-GRD","vxlan-vtep","br-tenant", "red","br-red","vxlan-red","rep-red","red-30","vport-18","blue","br-blue","vxlan-blue","rep-blue","green","br-green","vxlan-green","rep-green","blue-10","green-20","green-21","green-22","vport-35"}
	for _,str := range link_int  {
	device := netlink.Device{netlink.LinkAttrs{Name: str}} //vxlan-vtep"}}
	ensureIndex(device.Attrs())
	var neigh  Neigh_Struct
 	neigh.Neigh0 , neigh.Err = netlink.NeighList(device.Index, netlink.FAMILY_V4)
	if neigh.Err != nil {
	    fmt.Print("Failed to NeighList: %v", neigh.Err)
	}
	neigh.add_neigh(device.Attrs().Name)
//	dump_neighDB()
	}
}

func read_routes() {

//	link_int := []string{"lo","enp0s1f0", "enp0s1f0d1","enp0s1f0d2","enp0s1f0d3","enp0s1f0d4","enp0s1f0d5","rep-GRD","vxlan-vtep","br-tenant", "red","br-red","vxlan-red","rep-red","red-30","vport-18","blue","br-blue","vxlan-blue","rep-blue","green","br-green","vxlan-green","rep-green","blue-10","green-20","green-21","green-22","vport-35"}
	get_vrf_table()
	for V,T := range vrf_table  {
//	for _,str := range link_int  {
	 link,err := netlink.LinkByName(V)
		if err != nil {
			fmt.Println(err)
			continue
		}
	//fmt.Printf("Ifname %s\n",str)		
	var routes Route_Struct
//	routes.Route0,routes.Err = netlink.RouteList(nil, netlink.FAMILY_V4)
	routes.Route0,routes.Err = netlink.RouteListFiltered(netlink.FAMILY_V4, &netlink.Route{
		LinkIndex: link.Attrs().Index,
		Table:    T,
		MultiPath: []*netlink.NexthopInfo{{LinkIndex: link.Attrs().Index}},
	}, netlink.RT_FILTER_TABLE | netlink.RT_FILTER_IIF)
	if routes.Err != nil {
		fmt.Println(routes.Err)
	}

	routes.add_route(V)
	dump_RouteDB()
	}
}

func notify_route_added(R map[Route_key]Route){
	log.Println("Notify: Adding {R.format()}.")

/*for subscriber in cls.Subscribers:
	try:
		subscriber.route_added(R)
	except:
		logger.exception(f"Netlink subscriber {subscriber} failed to add {R}")
*/
}
 
func notify_route_deleted(R map[Route_key]Route){
	log.Println("Notify: Deleting {R.format()}.")

/*for subscriber in cls.Subscribers:
	try:
		subscriber.route_deleted(R)
	except:
		logger.exception(f"Netlink subscriber {subscriber} failed to delete {R}")
*/
}
 
func notify_route_updated(new map[Route_key]Route, old map[Route_key]Route){
	log.Println("Notify: Replacing {old.format()}\n")

/*for subscriber in cls.Subscribers:
	try:
		subscriber.route_updated(new)
	except:
		logger.exception(f"Netlink subscriber {subscriber} failed to update {new}")
*/
}		

 
func notify_nexthop_added(NH map[Nexthop_key]Nexthop){
	log.Println("Notify: Adding {NH.format()}.")

/*for subscriber in cls.Subscribers:
	try:
		subscriber.nexthop_added(NH)
	except:
		logger.exception(f"Netlink subscriber {subscriber} failed to add {NH}")
*/
}
 
func notify_nexthop_deleted( NH map[Nexthop_key]Nexthop){
	log.Println("Notify: Deleting {NH.format()}.")

/*for subscriber in cls.Subscribers:
	try:
		subscriber.nexthop_deleted(NH)
	except:
		logger.exception(f"Netlink subscriber {subscriber} failed to delete {NH}")
*/
}
 
func notify_nexthop_updated(new map[Nexthop_key]Nexthop, old map[Nexthop_key]Nexthop){
	log.Println("Notify: Replacing {old.format()} with {new.format()}.")

/* for subscriber in cls.Subscribers:
	try:
		subscriber.nexthop_updated(new)
	except:
		logger.exception(f"Netlink subscriber {subscriber} failed to update {new}")
*/
}
 
func notify_fdb_entry_added(M map[FDB_key]FdbEntry){
	log.Println("Notify: Adding {M.format()}.")

/*for subscriber in cls.Subscribers:
	try:
		subscriber.fdb_entry_added(M)
	except:
		logger.exception(f"Netlink subscriber {subscriber} failed to add {M}")
*/
}

func notify_fdb_entry_deleted(M map[FDB_key]FdbEntry){
    log.Println("Notify: Deleting {M.format()}.")

/*for subscriber in cls.Subscribers:
	try:
		subscriber.fdb_entry_deleted(M)
	except:
		logger.exception(f"Netlink subscriber {subscriber} failed to delete {M}")
*/
}
 
func notify_fdb_entry_updated(new map[FDB_key]FdbEntry, old map[FDB_key]FdbEntry){
    log.Println("Notify: Replacing {old.format()} with {new.format()}.")

/*for subscriber in cls.Subscribers:
	try:
		subscriber.fdb_entry_updated(new)
	except:
		logger.exception(f"Netlink subscriber {subscriber} failed to update {new}")
*/
}
func notify_l2_nexthop_added(L2N map[L2Nexthop_key]L2Nexthop){
    log.Println("Notify: Adding {L2N.format()}.")

/*for subscriber in cls.Subscribers:
	try:
		subscriber.l2_nexthop_added(L2N)
	except:
		logger.exception(f"Netlink subscriber {subscriber} failed to add {L2N}")
*/
}
 
func notify_l2_nexthop_deleted(L2N map[L2Nexthop_key]L2Nexthop){
    log.Println("Notify: Deleting {L2N.format()}.")

/*for subscriber in cls.Subscribers:
	try:
		subscriber.l2_nexthop_deleted(L2N)
	except:
		logger.exception(f"Netlink subscriber {subscriber} failed to delete {L2N}")
*/
}
 
func notify_l2_nexthop_updated(new map[L2Nexthop_key]L2Nexthop, old map[L2Nexthop_key]L2Nexthop){
	log.Println("Notify: Replacing {old.format()} with {new.format()}.")

/*for subscriber in cls.Subscribers:
	try:
		subscriber.l2_nexthop_updated(new)
	except:
		logger.exception(f"Netlink subscriber {subscriber} failed to update {new}")
*/
}

func notify_route_changes(new map[Route_key]Route_value, old map[Route_key]Route_value,added_cb func (R map[Route_key]Route), updated_cb func (N map[Route_key]Route, O map[Route_key]Route), deleted_cb func (R map[Route_key]Route)) {
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

func notify_nexthop_changes(new map[Nexthop_key]Nexthop,old map[Nexthop_key]Nexthop,added_cb func (N map[Nexthop_key]Nexthop), updated_cb func (N map[Nexthop_key]Nexthop, O map[Nexthop_key]Nexthop), deleted_cb func (N map[Nexthop_key]Nexthop) ){
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


func notify_FDB_changes(new map[FDB_key]FdbEntry, old map[FDB_key]FdbEntry,added_cb func (F map[FDB_key]FdbEntry), updated_cb func (N map[FDB_key]FdbEntry, O map[FDB_key]FdbEntry), deleted_cb func (F map[FDB_key]FdbEntry) ){
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

func notify_L2nexthop_changes(new map[L2Nexthop_key]L2Nexthop, old map[L2Nexthop_key]L2Nexthop,added_cb func (L2N map[L2Nexthop_key]L2Nexthop), updated_cb func (N map[L2Nexthop_key]L2Nexthop, O map[L2Nexthop_key]L2Nexthop), deleted_cb func (L2N map[L2Nexthop_key]L2Nexthop) ){
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

func annotate_db_entries(LatestRoutes map[Route_key]Route_value, LatestNexthops map[Nexthop_key]Nexthop , LatestFDB map[FDB_key]FdbEntry ,LatestL2Nexthops map[L2Nexthop_key]L2Nexthop ){
	//log.Println("In annotate_db_entries function \n")
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

func apply_install_filters(LatestRoutes map[Route_key]Route_value, LatestNexthops map[Nexthop_key]Nexthop , LatestFDB map[FDB_key]FdbEntry ,LatestL2Nexthops map[L2Nexthop_key]L2Nexthop){
	//log.Println("In apply_install_filters function \n")
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
	// with db_lock:
	Routes = LatestRoutes
	Nexthops = LatestNexthops
	Neighbors = LatestNeighbors
	FDB = LatestFDB
	L2Nexthops = LatestL2Nexthops
	// LatestRoutes = nil
	// LatestNexthops = nil
	// LatestNeighbors = nil
	// LatestFDB = nil
	// LatestL2Nexthops = nil

}

/*func GetContainerNetDevFromPci(netNSPath, pciAddress string) ([]string, error) {
	PidSlice := strings.Split(netNSPath, "/")[1:3]
	PidPath := strings.Join(PidSlice, "/")
	containerPciNetPath := filepath.Join(PidPath, ContainerSysBusPci, pciAddress, "net")
	return getFileNamesFromPath(containerPciNetPath)
}
*/


func monitor_netlink(p4_enabled bool) {
	// Wait for the p4 module to subscribe TBD
	//while p4_enabled and not NetlinkDB.Subscribers:
	//       time.sleep(1)
	for stop_monitoring != true {
		//log.Print("netlink: Polling netlink databases.")
		resync_with_kernel()
		//log.Print("netlink: Polling netlink databases completed.")
		time.Sleep(time.Duration(poll_interval) * time.Second)
	}
	//log.Println("netlink: Stopped periodic polling. Waiting for Infra DB cleanup to finish.\n")
	time.Sleep(2 * time.Second)
	//log.Println("netlink: One final netlink poll to identify what's still left.")
	//resync_with_kernel()
	// Inform subscribers to delete configuration for any still remaining Netlink DB objects.
	//log.Println("netlink: Delete any residual objects in DB")
    //  GetContainerNetDevFromPci()
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
	//log.Println("netlink: DB cleanup completed.")
	//run("grep")
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
	//log.Println(poll_interval)
	br_tenant = config.Linux_frr.Br_tenant
	//log.Println(br_tenant)
	nl_enabled := config.Netlink.Enable
	if nl_enabled != true {
		log.Println("netlink_monitor disabled")
		return
	}
	go monitor_netlink(config.P4.Enable) //monitor Thread started
	//log.Println("Started netlink_monitor thread with {poll_interval} s poll interval.")
//	time.Sleep(1 * time.Second)
//	stop_monitoring = true
	wg.Wait()
}
