package netlinkmodule

import (
	//	"fmt"
	"fmt"
	"log"
	"os/exec"
	"os"
	"sync"
	"time"
	"strings"
	"regexp"
	"strconv"	
//	"unicode"
	"reflect"
//	"sort"
//        "unsafe"
	"net"
	"golang.org/x/sys/unix"
	"encoding/binary"
	"encoding/json"
	//ipu_db "xpu/ipu_db"
	netlink "github.com/vishvananda/netlink"
	//"github.com/vishvananda/netns"
	eb "opi-poc/network/evpn-gw/netlink-module/vendor-plugin/event_bus"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

var db_lock int
var GRD int
var poll_interval int
var phy_ports  = make(map[string]int)
var br_tenant string
var stop_monitoring bool
var logger int
var EventBus = eb.NewEventBus()
var LOG_FILE string

type Vrf struct{
	Name string
	Vni uint32
	Routing_tables []uint32
	Rmac net.HardwareAddr
	//Routing_tables uint32
}

type Config_t struct {
	P4 struct {
		Enable bool `yaml:"enabled"`
	} `yaml: "p4"`
	Linux_frr struct {
		Enable       bool   `yaml:"enabled"`
		Default_vtep string `yaml:"default_vtep"`
		Port_mux     string `yaml:"port_mux"`
		Vrf_mux      string `yaml:"vrf_mux"`
		Br_tenant    string    `yaml:"br_tenant"`
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


func run(cmd []string) (string, error) {
        var out []byte
        var err error
        out, err = exec.Command("sudo",cmd...).Output()
        if err != nil {
                log.Println(cmd)
                return "",err
        }
        output := string(out[:])
        return output,err
}


const ( // Route direction
	None_ Direction = iota
	RX
	TX
	RX_TX
)

const ( //Nexthop_struct TYPE & L2NEXTHOP TYPE & FDBentry
	PHY = iota
	SVI
	ACC
	VXLAN
	BRIDGE_PORT
	OTHER 
)

const ( 
	RTN_Neighbor = 1111
)
//Subscribers    list[Subscriber] = []
type Neigh_key struct {
	Dst string
	VRF_name string
	Dev  int 
}


type Route_key struct {
	Table int
	Dst string
}

type Nexthop_key struct {
        VRF_name string
        Dst  string
        Dev int
        Local bool
}

type  Neigh_IP_Struct struct {
	Dst string
	Dev string
	Lladdr string
	Extern_learn string
	State []string
	Protocol string 
}

type FDB_key struct {
	Vlan_id int
	Mac string
}

type L2Nexthop_key struct {
        Dev string
        Vlan_id int
        Dst string
}

type Fdb_IP_Struct struct{
        Mac string
        Ifname string
        Vlan int
        Flags []string
        Master string
        State string
        Dst string
}


var	Routes = make( map[Route_key]Route_struct)
var	Nexthops = make(map[Nexthop_key]Nexthop_struct)
var	Neighbors = make(map[Neigh_key]Neigh_Struct)
var     FDB  = make(map[FDB_key]FdbEntry_struct)
var	L2Nexthops = make(map[L2Nexthop_key]L2Nexthop_struct)


	// Shadow tables for building a new netlink DB snapshot
var	LatestRoutes =  make(map[Route_key]Route_struct)
var	LatestNexthop = make(map[Nexthop_key]Nexthop_struct)
var	LatestNeighbors = make(map[Neigh_key]Neigh_Struct)
var     LatestFDB  =  make(map[FDB_key]FdbEntry_struct)
var	LatestL2Nexthop = make(map[L2Nexthop_key]L2Nexthop_struct)

type NetlinkDB struct {
	 


}

type Ri func(int /* V: ipu_db*/, map[string]string)
type Genfunc func()


/*
type Route struct {
	Route_init   *Ri
	Route_common Commonfp
}
*/

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

type Route interface {
	Route_store(Vrf,map[string]string)
}

type Route_struct struct {
	Route0 netlink.Route
	Vrf Vrf
	Nexthops []Nexthop_struct
	Metadata map[interface{}]interface{}
	Nl_type  string
	Key Route_key
	Err error
}

type Route_list struct {
	RS []Route_struct
}
/*
type NexthopInfo struct {
	LinkIndex int
	Hops      int
	Gw        net.IP
	Flags     int
	NewDst    Destination
	Encap     Encap
}
*/
type Nexthop_struct struct {
	NH netlink.NexthopInfo  
	Vrf Vrf
	Local bool
	Weight int
	//dst net.IP
	Metric int
	Id int
	Scope int 
	Protocol int
	Route_refs []Route_struct
	Key Nexthop_key
	Resolved bool
	Neighbor *Neigh_Struct //???
	Nh_type int
	Metadata map[interface{}]interface{}
}

func NetMaskToInt(mask int) (netmaskint [4]int64) {
	var binarystring string
	
	for ii := 1; ii <= mask; ii++ {
		binarystring = binarystring + "1"
	}
	for ii := 1; ii <= (32 - mask); ii++ {
		binarystring = binarystring + "0"
	}
	oct1 := binarystring[0:8]
	oct2 := binarystring[8:16]
	oct3 := binarystring[16:24]
	oct4 := binarystring[24:]
	//var netmaskint [4]int
	netmaskint[0], _ = strconv.ParseInt(oct1, 2, 64)
	netmaskint[1], _ = strconv.ParseInt(oct2, 2, 64)
	netmaskint[2], _ = strconv.ParseInt(oct3, 2, 64)
	netmaskint[3], _ = strconv.ParseInt(oct4, 2, 64)
	
	//netmaskstring = strconv.Itoa(int(ii1)) + "." + strconv.Itoa(int(ii2)) + "." + strconv.Itoa(int(ii3)) + "." + strconv.Itoa(int(ii4))
	return netmaskint
}

var Rtn_type = map[string]int {
	"unspec": unix.RTN_UNSPEC,
	"unicast": unix.RTN_UNICAST,
	"local": unix.RTN_LOCAL,
	"broadcast": unix.RTN_BROADCAST,
	"anycast": unix.RTN_ANYCAST,
	"multicast": unix.RTN_MULTICAST,
	"blackhole":unix.RTN_BLACKHOLE,
	"unreachable": unix.RTN_UNREACHABLE,
	"prohibit": unix.RTN_PROHIBIT,
	"throw": unix.RTN_THROW,
	"nat": unix.RTN_NAT,
	"xresolve": unix.RTN_XRESOLVE,
	"neighbor": RTN_Neighbor,
}	

var Rtn_proto = map[string]int {
	"unspec": unix.RTPROT_UNSPEC, 
	"redirect": unix.RTPROT_REDIRECT,
	"kernel": unix.RTPROT_KERNEL,  
	"boot": unix.RTPROT_BOOT, 
	"static": unix.RTPROT_STATIC, 
	"bgp": int('B'),
	"ipu_infra_mgr" : int('I'),
	"196": 196,
}

var Rtn_scope = map[string]int {
	"global": unix.RT_SCOPE_UNIVERSE,   
	"site": unix.RT_SCOPE_SITE,       
	"link": unix.RT_SCOPE_LINK,       
	"local": unix.RT_SCOPE_HOST,       
	"nowhere": unix.RT_SCOPE_NOWHERE,    
}
type flagstring struct {
        f int
        s string
}


var testFlag = []flagstring{
        {f:  unix.RTNH_F_ONLINK, s: "onlink"},
        {f: unix.RTNH_F_PERVASIVE, s: "pervasive"},
}


func get_flags(s string)int{
	f := 0
	for _,F:= range testFlag {
		if  s == F.s {
		     f |=  F.f
	        } 
	}
	return f
}


func get_flag_string(flag int) string{
	f := ""
	for _,F:= range testFlag {
		if  F.f == flag {
		     str:= F.s	
		     return str
	        } 
	}
	return f
}


var Nh_id_cache = make(map[Nexthop_key]int)
var Nh_next_id = 16
func NH_assign_id(key Nexthop_key) int {
	id := Nh_id_cache[key]
        if id == 0 {
            // Assigne a free id and insert it into the cache
            id = Nh_next_id
            Nh_id_cache[key] = id
            Nh_next_id += 1
	}
	return id
}

func NH_parse(V Vrf ,Nh Route_cmd_info)  Nexthop_struct {
	var nh Nexthop_struct
	nh.Weight=1
	nh.Vrf = V
		if !reflect.ValueOf(Nh.Dev).IsZero() {
			vrf, _ := netlink.LinkByName(Nh.Dev)
                        nh.NH.LinkIndex = vrf.Attrs().Index
		 }
		if  len(Nh.Flags) !=0{
			nh.NH.Flags = get_flags(Nh.Flags[0])
		}
		if !reflect.ValueOf(Nh.Gateway).IsZero() {
			nIP := &net.IPNet{
                                IP: net.ParseIP(Nh.Gateway),
                        }
                        nh.NH.Gw = nIP.IP
		}	
		if !reflect.ValueOf(Nh.Protocol).IsZero() {
			nh.Protocol = Rtn_proto[Nh.Protocol]
		}	
		if !reflect.ValueOf(Nh.Scope).IsZero() {
			nh.Scope = Rtn_scope[Nh.Scope]
		}	
		if !reflect.ValueOf(Nh.Type).IsZero() {
			nh.Nh_type =Rtn_type[Nh.Type]
			if nh.Nh_type == unix.RTN_LOCAL{
				nh.Local= true
			} else {
				nh.Local = false
			}
		}
		if !reflect.ValueOf(Nh.Weight).IsZero() {
			nh.Weight= Nh.Weight
		}
//	}
	nh.Key = Nexthop_key{nh.Vrf.Name, nh.NH.Gw.String(), nh.NH.LinkIndex, nh.Local}
	return nh
}


func check_Rtype(Type string) bool {
	var Types=[6]string{"connected","evpn-vxlan","static","bgp","local","neighbor"}
	for _,v:=range Types{
		if v == Type {
			return true		
		}	
	}
	return false
}

func pre_filter_route(R Route_struct) bool{
	if check_Rtype(R.Nl_type) && R.Route0.Dst.IP.IsLoopback()!=true && strings.Compare(R.Route0.Dst.IP.String(),"0.0.0.0")!=0 {
		return true		
	}else {
	       return false
	}

}

func check_proto(proto int) bool {
	var protos=[3]int{unix.RTPROT_BOOT , unix.RTPROT_STATIC , 196}
	for _,v:=range protos{
		if proto == v {
			return true		
		}	
	}
	return false	
}

func (route Route_struct) annotate(){
	route.Metadata = make(map[interface{}]interface{})
	for i := 0; i < len(route.Nexthops); i++ {
		NH := route.Nexthops[i]
		//route.Metadata["nh_ids"] = append(route.Metadata["nh_ids"], string(NH.id))
		route.Metadata["nh_ids"] = NH.Id
	}
	if route.Vrf.Vni != 0 {
		route.Metadata["vrf_id"] = string(route.Vrf.Vni)
	} else {
		route.Metadata["vrf_id"] = ""
	}
	if len(route.Nexthops) != 0{
		NH := route.Nexthops[0]
		if route.Vrf.Vni != 0{ // GRD
			if NH.Nh_type == PHY{
				route.Metadata["direction"] = string(TX)
			} else if NH.Nh_type == ACC{
				route.Metadata["direction"] = string(RX)
			} else { 
				route.Metadata["direction"] = "NONE"
			}
		} else {
			if NH.Nh_type == VXLAN{
				route.Metadata["direction"] = string(TX)
			} else if (NH.Nh_type == SVI || NH.Nh_type == ACC){
				route.Metadata["direction"] = string(RX_TX)
			} else {
				route.Metadata["direction"] = "NONE"
			}
		}
	} else {
		route.Metadata["direction"] = "NONE"
	}
}



func set_route_type(rs Route_struct, V Vrf) string {
	if (rs.Route0.Type == unix.RTN_UNICAST  &&  rs.Route0.Protocol == unix.RTPROT_KERNEL && rs.Route0.Scope == unix.RT_SCOPE_LINK && len(rs.Nexthops) == 1) {
		// Connected routes are proto=kernel and scope=link with a netdev as single nexthop
		return "connected"
	} else if (rs.Route0.Type == unix.RTN_UNICAST && rs.Route0.Protocol == int('B') && rs.Route0.Scope == unix.RT_SCOPE_UNIVERSE) {
		// EVPN routes to remote destinations are proto=bgp, scope global withipu_infra_mgr_db
		// all Nexthops residing on the br-<VRF name> bridge interface of the VRF.
		var devs []string
		if (len(rs.Nexthops)!=0) {
			for _,d := range rs.Nexthops{
				devs = append(devs,Name_index[d.NH.LinkIndex])
			}
			if (len(devs) == 1 && devs[0] == "br-"+V.Name) {
				return "evpn-vxlan"
			}else {
				return "bgp"
			}
		}
	} else if rs.Route0.Type == unix.RTN_UNICAST && check_proto(rs.Route0.Protocol) && rs.Route0.Scope == unix.RT_SCOPE_UNIVERSE {
		return "static"
	} else if (rs.Route0.Type == unix.RTN_LOCAL){
		return "local"
	} else if (rs.Route0.Type == RTN_Neighbor){
		// Special /32 or /128 routes for Resolved neighbors on connected subnets
		return "neighbor"
	}
	return "unknown"
}

var  Route_slice []Route_struct



func Parse_Route(V Vrf, Rm []Route_cmd_info,T int) Route_list{
	var route Route_list
	for _,Ro := range Rm {
		var rs Route_struct
		rs.Vrf =V
		if !reflect.ValueOf(Ro.Nhid).IsZero()||!reflect.ValueOf(Ro.Gateway).IsZero() || !reflect.ValueOf(Ro.Dev).IsZero(){
			rs.Nexthops = append(rs.Nexthops,NH_parse(V,Ro))
		}
		rs.Nl_type = "unknown"
		rs.Route0.Table=T
		rs.Route0.Priority = 1
		if !reflect.ValueOf(Ro.Dev).IsZero() {
			dev, _ := netlink.LinkByName(Ro.Dev)
                        rs.Route0.LinkIndex = dev.Attrs().Index
		}
		if !reflect.ValueOf(Ro.Dst).IsZero() {
			var Mask int
			split := Ro.Dst
			if (strings.Contains(Ro.Dst, "/")){
				split4:=strings.Split(Ro.Dst,"/")
				Mask,_=strconv.Atoi(split4[1])
				split = split4[0]
			} else {
				Mask=0
			}
			 var nIP *net.IPNet
                                if Ro.Dst == "default" {
                                        nIP = &net.IPNet{
                                                IP: net.ParseIP("0.0.0.0"),
                                                Mask: net.IPv4Mask(0,0,0,0),
                                        }
                                }else {
                                        mtoip := NetMaskToInt(Mask)
                                        b3 := make([]byte,8)  // Converting int64 to byte
                                        binary.LittleEndian.PutUint64(b3, uint64(mtoip[3]))
                                        b2 := make([]byte,8)
                                        binary.LittleEndian.PutUint64(b2, uint64(mtoip[2]))
                                        b1 := make([]byte,8)
                                        binary.LittleEndian.PutUint64(b1, uint64(mtoip[1]))
                                        b0 := make([]byte,8)
                                        binary.LittleEndian.PutUint64(b0, uint64(mtoip[0]))
                                        nIP = &net.IPNet{
                                                IP: net.ParseIP(split),
                                                Mask: net.IPv4Mask(b0[0],b1[0],b2[0],b3[0]),
                                        }
                                }
                                rs.Route0.Dst = nIP
		}
		if !reflect.ValueOf(Ro.Metric).IsZero() {
			rs.Route0.Priority = Ro.Metric
		}
		if !reflect.ValueOf(Ro.Protocol).IsZero() {
                                if (Rtn_proto[Ro.Protocol]!=0) {
                                        rs.Route0.Protocol = Rtn_proto[Ro.Protocol]
                                } else {
                                                rs.Route0.Protocol = 0
                                }
		}
		if !reflect.ValueOf(Ro.Type).IsZero() {
			rs.Route0.Type = Rtn_type[Ro.Type]
		}
		if len(Ro.Flags) != 0 {
			rs.Route0.Flags = get_flags(Ro.Flags[0])
		}
		if !reflect.ValueOf(Ro.Scope).IsZero() {
			rs.Route0.Scope = netlink.Scope(Rtn_scope[Ro.Scope])
		}
		if !reflect.ValueOf(Ro.Prefsrc).IsZero() {
			nIP := &net.IPNet{
                                        IP: net.ParseIP(Ro.Prefsrc),
                                }
                                rs.Route0.Src= nIP.IP
		}
		if !reflect.ValueOf(Ro.Gateway).IsZero() {
			  nIP := &net.IPNet{
                                        IP: net.ParseIP(Ro.Gateway),
                                }
                                rs.Route0.Gw= nIP.IP
		}
		if !reflect.ValueOf(Ro.VRF).IsZero() {
                                rs.Vrf = get_vrf_name(Ro.VRF.Name)
		}
		if !reflect.ValueOf(Ro.Table).IsZero() {
                               rs.Route0.Table=Ro.Table
		}
		rs.Nl_type = set_route_type(rs,V)
		rs.Key = Route_key{Table : rs.Route0.Table , Dst:rs.Route0.Dst.String() }
		if ( pre_filter_route(rs)==true){
			route.RS = append(route.RS,rs)
		}
	}
//	Route_slice = route.RS
//	sort.Slice(Route_slice, comparekey)
//	route.RS = Route_slice
	//	log.Printf("%+v",route)
	return route
}

func comparekey(i, j int) bool {
   return Route_slice[i].Key.Table > Route_slice[j].Key.Table &&  Route_slice[i].Key.Dst > Route_slice[j].Key.Dst
}

/*--------------------------------------------------------------------------
###  Nexthop_struct Database Entries
--------------------------------------------------------------------------*/
/*
func Nexthop_init(vrf Vrf, dst string, dev string, Local bool, weight int, flags string) Nexthop_struct{
	var NH Nexthop_struct
	NH.vrf = vrf
        NH.dst = dst
        NH.dev = dev
        NH.Local = local
	NH.NH_key.VRF_name=vrf.Name
	NH.NH_key.dst= dst
	NH.NH_key.dev=dev
	NH.NH_key.Local=local
        NH.weight = weight
        NH.flags = flags
        NH.Id = 0                            // Defined if and only if Nexthop_struct in DB
        NH.route_refs = nil
        NH.Resolved = (!reflect.ValueOf(dst).IsZero())       // Nexthop_structs with a dst IP need resolution
     //   NH.neighbor = 0
        NH.Nh_type = "other"
        NH.Metadata =  nil
	return NH
}
*/
//type Assign_id func()
type Try_resolve func(map[string]string)

/*type Nexthop_struct struct {
	Nethop_init  *Ni
	try_resolve  *Try_resolve
	Route_common Commonfp
}*/

/*--------------------------------------------------------------------------
###  Bridge MAC Address Database
###
###  We split the Linux FDB entries into DMAC and L2 Nexthop_struct tables similar
###  to routes and L3 nexthops, Thus, all remote EVPN DMAC entries share a
###  single VXLAN L2 nexthop table entry.
###
###  TODO: Support for dynamically learned MAC addresses on BridgePorts
###  (e.g. for pod interfaces operating in promiscuous mode).
--------------------------------------------------------------------------*/

type L2Nexthop_struct struct{
        Dev string
        Vlan_id int
        Dst net.IP
        Key L2Nexthop_key
        //lb
        //bp
        Id int
        Fdb_refs []FdbEntry_struct
        Resolved bool
        //id_cache map[L2Nexthop_key]int
        Type int
	Metadata map[interface{}]interface{}
}

type FdbEntry_struct struct {
        //Route0 netlink.Route
        Vlan_id int
        Mac string
        Key FDB_key
        State string
        //lb
        //bp
        Nexthop L2Nexthop_struct
        Type int
        Metadata map[interface{}]interface{}
        Err error
}

type FDBEntry_list struct {
        FS []FdbEntry_struct
}

func Parse_Fdb(fdb_ip Fdb_IP_Struct, fdbentry FdbEntry_struct ) FdbEntry_struct{
        fdbentry.Vlan_id = fdb_ip.Vlan
        fdbentry.Mac = fdb_ip.Mac
        fdbentry.Key = FDB_key{fdb_ip.Vlan, fdb_ip.Mac}
        fdbentry.State = fdb_ip.State
	/*   //Need to complete InfraDB
	fdbentry.lb = InfraDB.get_LB(fdbentry.Vlan_id)
        // TODO: This only handles the case of the VF Mac address itself,
        // not any Mac addresses used over the VF (in promiscuous mode)
        if !(reflect.ValueOf(fdbentry.lb).IsZero()){
		bp = fdbentry.lb.lookup_Mac(fdbentry.Mac)
	}
	*/
        Dev := fdb_ip.Ifname
        dst := fdb_ip.Dst
        fdbentry.Nexthop = fdbentry.Nexthop.Parse_L2NH(fdbentry.Vlan_id, Dev, dst/*, lb, bp*/)
        fdbentry.Type = fdbentry.Nexthop.Type
        return fdbentry
}

func (L2NH L2Nexthop_struct)Parse_L2NH(Vlan_id int, Dev string, dst string /*, LB, BP */) L2Nexthop_struct{
        L2NH.Dev = Dev
        L2NH.Vlan_id = Vlan_id
        L2NH.Dst = net.IP(dst)
        L2NH.Key = L2Nexthop_key{L2NH.Dev, L2NH.Vlan_id, string(L2NH.Dst)}
	//L2NH.lb: ipu_db.LogicalBridge = LB
        //L2NH.bp: ipu_db.BridgePort = BP
        if L2NH.Dev == fmt.Sprintf("svi-",L2NH.Vlan_id){
                L2NH.Type = SVI
        } else if L2NH.Dev == fmt.Sprintf("vxlan-",L2NH.Vlan_id){
                L2NH.Type = VXLAN
        } //else if L2NH.bp {
        //TODO
        /*L2NH.Type = BRIDGE_PORT
	} else {
            L2NH.Type = None
    	}    
        */
        return L2NH
}



var l2nexthop_id = 16


var l2Nh_id_cache = make(map[L2Nexthop_key]int)
func L2NH_assign_id(key L2Nexthop_key) int {
        id := l2Nh_id_cache[key]
        if id == 0 {
            // Assigne a free id and insert it into the cache
            id = l2nexthop_id
            l2Nh_id_cache[key] = id
            l2nexthop_id += 1
        }
        return id
}


func add_fdb_entry(M FdbEntry_struct){
       M = add_l2_nexthop(M)
	//TODO
        //logger.debug(f"Adding {M.format()}.")
        LatestFDB[M.Key] = M
}

func add_l2_nexthop(M FdbEntry_struct) FdbEntry_struct{
	if reflect.ValueOf(LatestL2Nexthop).IsZero() {
		log.Fatal("L2Nexthop DB empty\n")
		return  FdbEntry_struct{}
	}
        L2N := LatestL2Nexthop[M.Nexthop.Key]
        if !(reflect.ValueOf(L2N).IsZero()){
                L2N.Fdb_refs = append(L2N.Fdb_refs, M) //L2N.fdb_refs.append(R) --- what is R here??????
                M.Nexthop = L2N

        } else{
                L2N = M.Nexthop
                L2N.Fdb_refs = append(L2N.Fdb_refs, M)
                L2N.Id= L2NH_assign_id(L2N.Key)
                //L2N.assign_id()
//		log.Printf("VV %d\n",L2N.Id)
                LatestL2Nexthop[L2N.Key] = L2N
		M.Nexthop = L2N
		//log.Printf("in add function %+v\n",M)
	}
	return M
}

/*--------------------------------------------------------------------------
###  Neighbor Database Entries
--------------------------------------------------------------------------*/
type Neigh_init func(int, map[string]string)


var wg sync.WaitGroup
var link_table []netlink.Link
var vrf_list []netlink.Link
var device_list []netlink.Link
var vlan_list []netlink.Link
var bridge_list []netlink.Link
var vxlan_list []netlink.Link
var link_list []netlink.Link
var  Name_index= make(map[int]string)

func getlink() {
	links, err := netlink.LinkList()
	if err != nil {
               log.Fatal(err)
        }
	for i := 0; i < len(links); i++ {
		link_table=append(link_table,links[i])
		Name_index[links[i].Attrs().Index]=links[i].Attrs().Name
		if reflect.DeepEqual(links[i].Type(), "vrf") {
			vrf_list = append(vrf_list,links[i])
		} else if reflect.DeepEqual(links[i].Type(), "device") {
			device_list= append(device_list,links[i])
		} else if reflect.DeepEqual(links[i].Type(), "vlan") {
			vlan_list = append(vlan_list,links[i])
		} else if reflect.DeepEqual(links[i].Type(), "bridge") {
			bridge_list =append(bridge_list,links[i])
		} else if reflect.DeepEqual(links[i].Type(), "vxlan") {
			vxlan_list = append(vxlan_list,links[i])
		}
		link_list = append(link_list,links[i])
	}
}



func read_latest_netlink_state() {
	for  _,V := range vrf_table {
		read_neighbors(V) // viswanantha library
	        read_routes(V) //Viswantha library
	}
        M := read_FDB()
        for i := 0; i < len(M); i++{
                add_fdb_entry(M[i])
        }
	dump_DBs()	
}

func dump_DBs(){
	dump_RouteDB()
	log.Printf("\n")
	dump_NexthDB()
	log.Printf("\n")
	dump_neighDB()
	log.Printf("\n")
	dump_FDB()
	log.Printf("\n")
	dump_L2NexthDB()
	
}

func ensureIndex(link *netlink.LinkAttrs) {
	if link != nil && link.Index == 0 {
			newlink, _ := netlink.LinkByName(link.Name)
			if newlink != nil {
					link.Index = newlink.Attrs().Index
			}
	}
}



type Neigh_Struct struct {
	 Neigh0 netlink.Neigh
	 Protocol string
	 Vrf_name  string
	 Type  int
	 Dev string
	 Err error
	 Key Neigh_key
	 Metadata map[string]string
}

type  Neigh_list struct {
	NS []Neigh_Struct	
}

func  neighbor_annotate(neighbor Neigh_Struct)Neigh_Struct{
        if strings.HasPrefix(neighbor.Dev, neighbor.Vrf_name) &&  neighbor.Protocol != "zebra"{
                pattern := fmt.Sprintf(`%s-\d+$`,neighbor.Vrf_name)
                mustcompile := regexp.MustCompile(pattern)
                s := mustcompile.FindStringSubmatch(neighbor.Dev)
                vlan_id := strings.Split(s[0],"-")[1]
                //TODO
                //LB = InfraDB.get_LB(vlan_id)
                //BP: ipu_db.BridgePort = LB.lookup_mac(self.lladdr)
                //if BP{
                        neighbor.Type = SVI
                        neighbor.Metadata["vport_id"] = "0xa"//BP.vport_id
                        neighbor.Metadata["vlan_id"] = vlan_id
                        neighbor.Metadata["port_type"] = "host"//BP.type
                /*else{
                        neighbor.Type = None;
                }
                logger.exception(f"Failed to lookup egress vport for SVI neighbor {self}")*/
        } else if neighbor.Vrf_name == "GRD" && neighbor.Protocol != "zebra"{
                for d, _ := range phy_ports{
                        if neighbor.Dev == d{
				//fmt.Printf("%+v\n",neighbor)
                                neighbor.Type = PHY
                                neighbor.Metadata["vport_id"] = string(phy_ports[d]) //neighbor.Dev]
                        }
                }
        //logger.debug(f"Annotated {self}: type={self.type} extra={self.metadata}")
        }
	return neighbor
}



func Check_Ndup(tmp_key Neigh_key) bool {
	var dup = false
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

func Check_Vrfdup(Vrf_tmp Vrf) bool {
	var dup = false
	for _,tmp := range vrf_table {
			if tmp.Name == Vrf_tmp.Name {
		   		dup = true	
		   		break
				}
	}
	return dup		
}

func  add_neigh (dump Neigh_list){
	for _, n := range dump.NS {
			n=neighbor_annotate(n)
			if(len(LatestNeighbors)==0) {
				LatestNeighbors[n.Key]= n
			} else {
				if !Check_Ndup(n.Key){
			   	     LatestNeighbors[n.Key]= n
				}
			}
		}
	}

func get_state_str(s int) string {
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
       return neigh_state[s]

}

func  print_Neigh(Ng *Neigh_Struct)string{
		var Proto string
		//N :=Neigh_Struct{}
		if Ng == nil {
			return "None"
		}
                if Ng.Protocol == "" {
                        Proto = "None"
                } else {
                        Proto = Ng.Protocol
                }
		str := fmt.Sprintf("Neighbor(vrf=%s dst=%s lladdr=%s dev=%s proto=%s state=%s) ",Ng.Vrf_name,Ng.Neigh0.IP.String(),Ng.Neigh0.HardwareAddr.String(),Name_index[Ng.Neigh0.LinkIndex],Proto,get_state_str(Ng.Neigh0.State))
                //log.Println(str)
		return str
}

func dump_RouteDB() {
		log.Printf("len %d\n",len(LatestRoutes))
		log.Printf("Route table:\n")
		for _, n := range LatestRoutes {
			var via string
			if  n.Route0.Gw.String()=="<nil>" {
				via = "None"
			} else {
				via = n.Route0.Gw.String()
			}
			str := fmt.Sprintf("Route(vrf=%s dst=%s type=%s proto=%s metric=%d  via=%s dev=%s nhid= %d Table= %d)",n.Vrf.Name, n.Route0.Dst.String(),n.Nl_type, get_proto(n),n.Route0.Priority,via,Name_index[n.Route0.LinkIndex],n.Nexthops[0].Id,n.Route0.Table)
			log.Println(str)
		}
		log.Printf("\n\n\n")
	}
	

func dump_L2NexthDB(){
	log.Printf("L2 Nexthop table:\n")
	log.Printf("len %d\n",len(LatestL2Nexthop))
	var ip string
	for _, n := range LatestL2Nexthop {
		 if  n.Dst.String()=="<nil>" {
                        ip= "None"
                } else {
                        ip = n.Dst.String()
                }
		 str := fmt.Sprintf("L2Nexthop(id=%d dev=%s vlan=%d dst=%s type=%d #FDB entries=%d Resolved=%t) ",n.Id,n.Dev,n.Vlan_id,ip,n.Type,len(n.Fdb_refs),n.Resolved)
                log.Println(str)
	}
	log.Printf("\n\n\n")
}

func dump_FDB(){
	log.Printf("FDB table:\n")
	log.Printf("len %d\n",len(LatestFDB))
	for _, n := range LatestFDB {
		 str := fmt.Sprintf("MacAddr(vlan=%d mac=%s state=%s type=%d l2nh_id=%d) ",n.Vlan_id,n.Mac,n.State,n.Type,n.Nexthop.Id)
                log.Println(str)
	}
	log.Printf("\n\n\n")
}
func dump_NexthDB(){
	log.Printf("Nexthop table:\n")
	log.Printf("len %d\n",len(LatestNexthop))
	for _, n := range LatestNexthop {
		 str := fmt.Sprintf("Nexthop(id=%d vrf=%s dst=%s dev=%s Local=%t weight=%d flags=[%s] #routes=%d Resolved=%t neighbor=%s) ",n.Id,n.Vrf.Name,n.NH.Gw.String(),Name_index[n.NH.LinkIndex],n.Local,n.Weight,get_flag_string(n.NH.Flags),len(n.Route_refs),n.Resolved,print_Neigh(n.Neighbor))
                log.Println(str)
	}
	log.Printf("\n\n\n")
}


func dump_neighDB() {
	log.Printf("Neighbor table:\n")
	log.Printf("len %d\n",len(LatestNeighbors))
	for _, n := range LatestNeighbors {
		var Proto string
		if n.Protocol == "" {
			Proto = "None"
		} else {
			Proto = n.Protocol	
		}
		str := fmt.Sprintf("Neighbor(vrf=%s dst=%s lladdr=%s dev=%s proto=%s state=%s Type : %d) ",n.Vrf_name,n.Neigh0.IP.String(),n.Neigh0.HardwareAddr.String(),Name_index[n.Neigh0.LinkIndex],Proto,get_state_str(n.Neigh0.State),n.Type)
		log.Println(str)
    }
}

func get_proto(n Route_struct) string{
	for p,i:=range Rtn_proto {
		if (i== n.Route0.Protocol){
			return p
		}
	}
	return string(0)
}


func get_type(n Route_struct) string{
	for t,i:=range Rtn_type {
		if (i== n.Route0.Type){
			return t
		}
	}
	return string(0)
}


func check_neigh(Nk Neigh_key) bool {
	for k,_ := range LatestNeighbors {
		if k == Nk{
		    return true
		}
	}
	return false
	
}

func try_resolve(Nh Nexthop_struct) Nexthop_struct {
	if len(Nh.NH.Gw)!=0 {
		// Nexthops with a gateway IP need resolution of that IP
		neighbor_key:= Neigh_key{Dst: Nh.NH.Gw.String(),VRF_name: Nh.Vrf.Name, Dev: Nh.NH.LinkIndex}
		ch:=check_neigh(neighbor_key)
		if ch == true  && LatestNeighbors[neighbor_key].Neigh0.Type !=0{
			Nh.Resolved = true
			nh:= LatestNeighbors[neighbor_key]
			Nh.Neighbor = &nh
			//fmt.Println(Nh.neighbor)
		} else {
			Nh.Resolved = false
			//Nh.Neighbor = Neigh_Struct{}
		}
	}else {
            Nh.Resolved = true
	}
	return Nh
}


func check_NhDB(Nhk Nexthop_key) bool {
        for k,_ := range LatestNexthop {
                if k == Nhk{
                    return true
                }
        }
        return false
}


func add_nexthop(NH Nexthop_struct,R Route_struct) Route_struct{
	ch := check_NhDB(NH.Key)
//	 fmt.Printf("CH %d\n",ch)
	if ch== true {
		NH0 := LatestNexthop[NH.Key]
		// Links route with existing nexthop
		NH0.Route_refs =append(NH0.Route_refs,R)
		R.Nexthops= append(R.Nexthops,NH0)
		//fmt.Printf("Adding route to %v\n",nh.Key)
	} else {
		// Create a new nexthop entry
		NH.Route_refs = append(NH.Route_refs,R)
		NH.Id = NH_assign_id(NH.Key)
		NH = try_resolve(NH)
		LatestNexthop[NH.Key] = NH
		R.Nexthops = append(R.Nexthops,NH)
	}
	return R
}


func check_route(R Route_struct) bool {
	Rk := R.Key
        for k,_ := range LatestRoutes {
                if k == Rk{
                    return true
                }
        }
	return false
}

func delete_NH(NH []Nexthop_struct)[]Nexthop_struct {
    index:=len(NH)
    if (index == 1){
        NH=append(NH[:0], NH[1:]...)
    } else {
    for i:=0; i<index-1 ;i++ {
        NH=append(NH[:0], NH[1:]...)
    }
    }
    return NH
}


func add_route(R Route_struct){
		ch:= check_route(R)
		if ch == true {
			R0:=LatestRoutes[R.Key]
			if R.Route0.Priority >= R0.Route0.Priority{
				// Route with lower metric exists and takes precedence
				log.Printf("Ignoring %+v  with higher metric than %+v\n",R,R0)
			}else{
				log.Printf("conflicts %+v with higher metric %+v. Will ignore it",R,R0)
			}
		} else {
			Nexthops := R.Nexthops
			R.Nexthops = delete_NH(R.Nexthops)
			for _,NH := range Nexthops{
				R= add_nexthop(NH, R)
			}
			LatestRoutes[R.Key] = R
		}
}


var vrf_table []Vrf
func get_vrf_info() {

	var vrf_tmp Vrf
	if(len(vrf_table)==0){
		vrf_tmp.Name="GRD"
		vrf_tmp.Routing_tables=append(vrf_tmp.Routing_tables,254,255)
		vrf_tmp.Vni=0
		vrf_table= append(vrf_table,vrf_tmp)
	}	
	for _,v := range vrf_list {
		var vrf_tmp1 Vrf
		Vtable := v.(*netlink.Vrf)		
		vrf_tmp1.Name= v.Attrs().Name
		vrf_tmp1.Vni=Vtable.Table 
		vrf_tmp1.Routing_tables=append(vrf_tmp1.Routing_tables,Vtable.Table)
		if !Check_Vrfdup(vrf_tmp1){
			vrf_table= append(vrf_table,vrf_tmp1)
		}	
	}	
}	


func cmd_process_Nb(nb string,v string) Neigh_list {
		var nbs []Neigh_IP_Struct
		CPs := strings.Split(nb[2:len(nb)-3], "},{")
		for i := 0; i<len(CPs); i++{
			var ni Neigh_IP_Struct
			log.Println(CPs[i])
			err := json.Unmarshal([]byte(fmt.Sprintf("{%v}",CPs[i])), &ni)
			if err != nil{
				log.Println("error-",err)
			}
			nbs = append(nbs, ni)
		}
		Neigh := Parse_neigh(nbs,v)
		return  Neigh
}


func get_state(s string) int{
	neigh_state := map[string]int{
                "NONE" : netlink.NUD_NONE,
                "INCOMPLETE" : netlink.NUD_INCOMPLETE,
                "REACHABLE" : netlink.NUD_REACHABLE,
                "STALE" : netlink.NUD_STALE,
                "DELAY": netlink.NUD_DELAY,
                "PROBE": netlink.NUD_PROBE,
                "FAILED": netlink.NUD_FAILED,
                "NOARP": netlink.NUD_NOARP,
                "PERMANENT": netlink.NUD_PERMANENT,
        	}
	return neigh_state[s]
}


func pre_filter_neighbor(n Neigh_Struct) bool {
	if (n.Neigh0.State !=  netlink.NUD_NONE && n.Neigh0.State !=  netlink.NUD_INCOMPLETE && n.Neigh0.State !=  netlink.NUD_FAILED && Name_index[n.Neigh0.LinkIndex] != "lo"){
		return true
	} else {
	       return false
	}
}
//func Parse_neigh(NM []map[string]string,v string) Neigh_list {
func Parse_neigh(NM []Neigh_IP_Struct,v string) Neigh_list {
	var NL Neigh_list  
	for  _,ND := range NM {
		var ns Neigh_Struct 
		ns.Neigh0.Type = OTHER
		ns.Vrf_name =v
		if !reflect.ValueOf(ND.Dev).IsZero(){
			vrf, _ := netlink.LinkByName(ND.Dev)
                        ns.Neigh0.LinkIndex = vrf.Attrs().Index
		}	
		if !reflect.ValueOf(ND.Dst).IsZero(){
			    ipnet := &net.IPNet{
                                        IP: net.ParseIP(ND.Dst),
                                }
                                ns.Neigh0.IP = ipnet.IP
		}	
		if !reflect.ValueOf(ND.State).IsZero(){
			ns.Neigh0.State = get_state(ND.State[0])
		}	
		if !reflect.ValueOf(ND.Lladdr).IsZero(){
				ns.Neigh0.HardwareAddr,_ = net.ParseMAC(ND.Lladdr)
		}		
		if !reflect.ValueOf(ND.Protocol).IsZero(){
			    ns.Protocol = ND.Protocol
		}	    
	//	ns  =  neighbor_annotate(ns)   /* Need InfraDB to finish for fetching LB/BP information */
		ns.Key = Neigh_key{VRF_name: v ,Dst :ns.Neigh0.IP.String(), Dev: ns.Neigh0.LinkIndex}
		    if pre_filter_neighbor(ns)==true {
			NL.NS= append(NL.NS,ns)
		    }
	      }
	return NL
}


func get_vrf_name(v string) Vrf{
	var vr Vrf
	for _,V :=range vrf_table{
		if   V.Name == v {
			return V
		}
	}
	return vr
}

func  get_neighbor_routes() []Route_cmd_info{ // []map[string]string{
            //Return a list of /32 or /128 routes & Nexthops to be inserted into
            //the routing tables for Resolved neighbors on connected subnets
            //on physical and SVI interfaces.
	    var neighbor_routes []Route_cmd_info //[]map[string]string
            for _,N := range LatestNeighbors{
                //if N.Type == PHY || N.Type == SVI {
                if ((Name_index[N.Neigh0.LinkIndex] == "enp0s1f0d1" || Name_index[N.Neigh0.LinkIndex] == "enp0s1f0d3") && N.Neigh0.State == netlink.NUD_REACHABLE ) {
			vrf:=get_vrf_name(N.Vrf_name)
                    //# Create a special route with dst == gateway to resolve
                    //# the nexthop to the existing neighbor
             R0:=  Route_cmd_info {Type: "neighbor", Dst: N.Neigh0.IP.String(), Protocol : "ipu_infra_mgr",Scope : "global" ,Gateway: N.Neigh0.IP.String(), Dev:Name_index[N.Neigh0.LinkIndex],VRF:vrf,Table:int(vrf.Routing_tables[0])}
		    neighbor_routes= append(neighbor_routes,R0)
                }
            }
            return neighbor_routes
}


func read_neighbors(V Vrf) {
	var N  Neigh_list
	var err error
	var Nb string
	if V.Vni == 0 {
	/* No support for "ip neighbor show" command in netlink library Raised ticket https://github.com/vishvananda/netlink/issues/913 ,
	   so using ip command as WA */ 
	       Nb,err= run([]string{"ip","-j","-d", "neighbor", "show"})
	       /*	neigh.NS.Neigh0 , neigh.NS.Err = netlink.NeighList(0, netlink.FAMILY_V4)
		if neigh.NS.Err != nil {
		    log.Print("Failed to NeighList: %v", neigh.NS.Err)
		}
	*/
	} else {
	       Nb,err= run([]string{"ip","-j","-d", "neighbor", "show", "vrf",V.Name})
	/*     vrf, _ := netlink.LinkByName(V.Name)
		neigh.NS.Neigh0 , neigh.Err = netlink.NeighList(vrf.Attrs().Index, netlink.FAMILY_V4)
		if neigh.NS.Err != nil {
		    log.Print("Failed to NeighList: %v", neigh.NS.Err)
		}
	*/
	}
	if len(Nb)!= 3 || err == nil {
		N =cmd_process_Nb(Nb,V.Name)
	}
	add_neigh(N)

}

type NH_route_info struct {
	Id int
	Gateway string
	Dev string 
	Scope string
	Protocol string
	Flags []string
}

type Route_cmd_info struct {
	Type string
	Dst  string
	Nhid   int
	Gateway  string
	Dev string
	Protocol string
	Scope  string
	Prefsrc string
	Metric int
	Flags  []string
	Weight int
	VRF Vrf
	Table int
	Nh_info NH_route_info //{id gateway Dev scope protocol flags}
}

func pre_filter_mac(F FdbEntry_struct) bool{
        //TODO M.nexthop.dst
        //if F.Vlan_id != 0 || !(reflect.ValueOf(F.Nexthop.Dst).IsZero()){
        if F.Vlan_id != 0  { //|| !(reflect.ValueOf(F.Nexthop.Dst).IsZero()){
		log.Printf("%d vlan \n",len(F.Nexthop.Dst.String()))
                return true
        }
        return false
}


func cmd_process_Rt(V Vrf,R string,T int) Route_list{
	var Route_data []Route_cmd_info
	CPs := strings.Split(R[2:len(R)-3], "},{")
                for i := 0; i<len(CPs); i++{
                        var ri Route_cmd_info
                        log.Println(CPs[i])
                        err := json.Unmarshal([]byte(fmt.Sprintf("{%v}",CPs[i])), &ri)
                        if err != nil{
                                log.Println("error-",err)
                        }
                        Route_data = append(Route_data, ri)
                }
		route:=Parse_Route(V,Route_data,T)
                return  route
}



func read_route_from_ip(V Vrf) {
		var Rl Route_list
		var rm []Route_cmd_info //map[string]string
		for _,Rt := range V.Routing_tables {
			Raw,err:=run([]string{"ip","-j","-d","route", "show","table",strconv.Itoa(int(Rt))})
			if err != nil {
				log.Printf("Err Command route\n")
				return
			}
			Rl = cmd_process_Rt(V,Raw,int(Rt))
			for _,R := range Rl.RS {
		               add_route(R)
			}
		}
		nl :=get_neighbor_routes()   //Add extra routes for Resolved neighbors on connected subnets
		for i:=0;i<len(nl);i++{
			rm = append(rm,nl[i])
		}
		nr:=Parse_Route(Vrf{},rm,0)
		for _,R := range nr.RS {
                       add_route(R)
                }
}



func read_routes(V Vrf) {
//	for _,str := range link_int  {
//	 link,err := netlink.LinkByName(I.Attrs().Name)
//		if err != nil {
//			log.Println(err)
//			return 
//		}
	//log.Printf("Ifname %s\n",str)		
	//var routes Route_list
//	routes.R,routes.Err = netlink.RouteList(nil, netlink.FAMILY_MPLS)
//	routes.R,routes.Err = netlink.RouteListFiltered(netlink.FAMILY_V4, &netlink.Route{
//		LinkIndex: link.Attrs().Index,
		//Table :  int(V.Routing_tables[0]),
//	}, netlink.RT_FILTER_IIF )
//	if routes.Err != nil {
//		log.Println(routes.Err)
//	}
//	log.Println(V.Routing_tables)
	read_route_from_ip(V)
//	dump_RouteDB()
}

func notify_route_added(R Route_struct){
//      log.Printf("Notify Route: Adding {%+v)\n",R)

        EventBus.Publish("route_added", R)
}

func notify_route_deleted(R Route_struct){
//      log.Printf("Notify: Route: Deleting {%+v}\n",R)
        EventBus.Publish("route_deleted", R)
}

func notify_route_updated(new_R Route_struct, old_R Route_struct){
//      log.Printf("Notify: Route Replacing new {%+v} old {%+v}\n",new_R,old_R)
        EventBus.Publish("route_updated", new_R)
}

func notify_nexthop_added(NH Nexthop_struct){
//      log.Printf("Notify: Nexthop Adding {%+v}\n",NH)
        EventBus.Publish("nexthop_added", NH)
}

func notify_nexthop_deleted( NH Nexthop_struct){
//      log.Printf("Notify: Nexthop Deleting {%+v}\n",NH)
        EventBus.Publish("nexthop_deleted", NH)
}

func notify_nexthop_updated(new_NH Nexthop_struct, old_NH Nexthop_struct){
//      log.Printf("Notify: Nexthop Replacing old {%+v} with {%+v}\n",old_NH,new_NH)
        EventBus.Publish("nexthop_updated", new_NH)
}


func notify_fdb_added(M FdbEntry_struct){
//	log.Println("Notify: Adding {M.format()}.")
	EventBus.Publish("fdb_entry_added", M)
}

func notify_fdb_deleted(M FdbEntry_struct){
//    log.Println("Notify: Deleting {M.format()}.")
	EventBus.Publish("fdb_entry_deleted", M)
}

func notify_fdb_updated(new_M FdbEntry_struct, old_M FdbEntry_struct){
//    log.Println("Notify: Replacing {old.format()} with {new.format()}.")
	EventBus.Publish("fdb_entry_updated", new_M)
}
func notify_L2nexthop_added(L2N L2Nexthop_struct){
//    log.Println("Notify: Adding {L2N.format()}.")
	EventBus.Publish("l2_nexthop_added", L2N)
}

func notify_L2nexthop_deleted(L2N L2Nexthop_struct){
//    log.Println("Notify: Deleting {L2N.format()}.")
	EventBus.Publish("l2_nexthop_delete", L2N)
}

func notify_L2nexthop_updated(new_L2N L2Nexthop_struct, old_L2N L2Nexthop_struct){
//	log.Println("Notify: Replacing {old.format()} with {new.format()}.")
	EventBus.Publish("l2_nexthop_updated", new_L2N)
}

func get_routes_changes(R1 []Route_key, R2 []Route_key,new_db map[Route_key]Route_struct, old_db map[Route_key]Route_struct,mod bool)([]Route_struct,[]Route_struct) {
	var old_route []Route_struct 
	var new_route []Route_struct
	var r2key Route_key
	var route_changes []Route_struct
	for _,r1_key := range R1 {
		change_flag := false
		mod_flag := false
		for _,r2_key := range R2 {
			if reflect.DeepEqual(r1_key,r2_key) {
				if mod == true{
					if !reflect.DeepEqual(new_db[r1_key],old_db[r2_key]) && new_db[r1_key].Nl_type == "neighbor" {
						log.Printf("mod DIfference route: new\n %+v \n\n old\n  %+v\n\n\n",new_db[r1_key],old_db[r2_key])
						//log.Printf("new Type %s and VRF %s\n old Type %s and VRF %s\n",new_db[r1_key].nl_type,new_db[r1_key].vrf.Name,old_db[r2_key].nl_type,old_db[r2_key].vrf.Name )
						mod_flag =true
					} else {
						//log.Printf("mod SAME route: new\n %+v \n\n old\n  %+v\n\n\n",new_db[r1_key],old_db[r2_key])
					}
				} else {
					change_flag = true
				}
			r2key = r2_key
			break
			}
		}
		if !mod && !change_flag {
			route_changes = append(route_changes,new_db[r1_key])
		}
		if mod && mod_flag {
			old_route = append(old_route,old_db[r2key])
			new_route = append(new_route,new_db[r1_key])
			mod_flag = false
		}
	}
	if mod {
		return new_route , old_route
	} else {
		return route_changes, nil
	}
}

func notify_route_changes(new_db map[Route_key]Route_struct, old_db map[Route_key]Route_struct) {
	var old_keys []Route_key
	var new_keys []Route_key
	for  k , _ :=range old_db{
		old_keys = append(old_keys,k)	
	} 
	for  k , _ :=range new_db{
		new_keys = append(new_keys,k)	
	}
	if len(new_keys)- len(old_keys) > 0{   // ADDED
		log.Printf("Added new route len %d old route len %d\n",len(new_keys),len(old_keys))
		Added_routes,_ := get_routes_changes(new_keys,old_keys,new_db,old_db,false)
		if len(Added_routes) != 0{
			for _,R := range Added_routes{
				notify_route_added(R)	
			}	
		}
	} else if len(new_keys)- len(old_keys) < 0 { // DELETED
		log.Printf("Deleted :new route len %d old route len %d\n",len(new_keys),len(old_keys))
		Deleted_routes,_ := get_routes_changes(old_keys,new_keys,new_db,old_db,false)
		if len(Deleted_routes) != 0{
			for _,R := range Deleted_routes{
				notify_route_deleted(R)	
			}	
		}
	} else if len(new_keys)- len(old_keys) == 0 {
		if !reflect.DeepEqual(new_db, old_db) {  // UPDATED
			//log.Printf("Updated :new route len %d old route len %d old: %+v \n new :%+v\n",len(new_keys),len(old_keys),old_keys,new_keys)
			new_routes,old_routes := get_routes_changes(new_keys,old_keys,new_db,old_db,true)
			if len(new_routes) != 0 && len(old_routes)!=0 {
				for i:=0;i<len(new_routes);i++ {
					notify_route_updated(new_routes[i],old_routes[i])
				}
			}
		} else {
			log.Printf("No changes noticed in route DB\n")
		}
	}
}


func get_fdb_changes(R1 []FDB_key, R2 []FDB_key,new_db map[FDB_key]FdbEntry_struct, old_db map[FDB_key]FdbEntry_struct,mod bool)([]FdbEntry_struct,[]FdbEntry_struct) {
	var old_fdb []FdbEntry_struct
	var new_fdb []FdbEntry_struct
	var r2key FDB_key
	var fdb_changes []FdbEntry_struct
	for _,r1_key := range R1 {
		change_flag := false
		mod_flag := false
		for _,r2_key := range R2 {
			if reflect.DeepEqual(r1_key,r2_key) {
				if mod == true{
					if !reflect.DeepEqual(new_db[r1_key],old_db[r2_key]) {
						log.Printf("mod DIfference route: new\n %+v \n\n old\n  %+v\n\n\n",new_db[r1_key],old_db[r2_key])
						//fmt.Printf("new Type %s and VRF %s\n old Type %s and VRF %s\n",new_db[r1_key].nl_type,new_db[r1_key].vrf.Name,old_db[r2_key].nl_type,old_db[r2_key].vrf.Name )
						mod_flag =true
					}	
				} else {
					change_flag = true
				}
			r2key = r2_key
			break
			}
		}
		if !mod && !change_flag {
			fdb_changes = append(fdb_changes,new_db[r1_key])
		}
		if mod && mod_flag {
			old_fdb = append(old_fdb,old_db[r2key])
			new_fdb = append(new_fdb,new_db[r1_key])
			mod_flag = false
		}
	}
	if mod {
		return new_fdb , old_fdb
	} else {
		return fdb_changes, nil
	}
}

func notify_FDB_changes(new_db map[FDB_key]FdbEntry_struct, old_db map[FDB_key]FdbEntry_struct) {
        var old_keys []FDB_key
        var new_keys []FDB_key
        for  k , _ :=range old_db{
                old_keys = append(old_keys,k)
        }
        for  k , _ :=range new_db{
                new_keys = append(new_keys,k)
        }
        if len(new_keys)- len(old_keys) > 0{   // ADDED
                log.Printf("Added new FDB len %d old FDB len %d\n",len(new_keys),len(old_keys))
                Added_FDBs,_ := get_fdb_changes(new_keys,old_keys,new_db,old_db,false)
                if len(Added_FDBs) != 0{
                        for _,R := range Added_FDBs{
                                notify_fdb_added(R)
                        }
                }
        } else if len(new_keys)- len(old_keys) < 0 { // DELETED
                log.Printf("Deleted :new FDB len %d old FDB len %d\n",len(new_keys),len(old_keys))
                Deleted_fdb,_ := get_fdb_changes(old_keys,new_keys,new_db,old_db,false)
                if len(Deleted_fdb) != 0{
                        for _,R := range Deleted_fdb{
                                notify_fdb_deleted(R)
                        }
                }
        } else if len(new_keys)- len(old_keys) == 0 {
                if !reflect.DeepEqual(new_db, old_db) {  // UPDATED
                        log.Printf("Updated :new route len %d old route len %d old: %+v \n new :%+v\n",len(new_keys),len(old_keys),old_keys,new_keys)
                        new_fdb,old_fdb := get_fdb_changes(new_keys,old_keys,new_db,old_db,true)
                        if len(new_fdb) != 0 && len(old_fdb)!=0 {
                                for i:=0;i<len(new_fdb);i++ {
                                        notify_fdb_updated(new_fdb[i],old_fdb[i])
                                }
                        }
                } else {
                        log.Printf("No changes noticed in route DB\n")
                }
        }
}


func get_L2Nexthop_changes(R1 []L2Nexthop_key, R2 []L2Nexthop_key,new_db map[L2Nexthop_key]L2Nexthop_struct, old_db map[L2Nexthop_key]L2Nexthop_struct,mod bool)([]L2Nexthop_struct,[]L2Nexthop_struct) {
	var old_l2nexthop []L2Nexthop_struct
	var new_l2nexthop []L2Nexthop_struct
	var r2key L2Nexthop_key
	var l2nexthop_changes []L2Nexthop_struct
	for _,r1_key := range R1 {
		change_flag := false
		mod_flag := false
		for _,r2_key := range R2 {
			if reflect.DeepEqual(r1_key,r2_key) {
				if mod == true{
					if !reflect.DeepEqual(new_db[r1_key],old_db[r2_key]) {
						log.Printf("mod DIfference route: new\n %+v \n\n old\n  %+v\n\n\n",new_db[r1_key],old_db[r2_key])
						//fmt.Printf("new Type %s and VRF %s\n old Type %s and VRF %s\n",new_db[r1_key].nl_type,new_db[r1_key].vrf.Name,old_db[r2_key].nl_type,old_db[r2_key].vrf.Name )
						mod_flag =true
					}
				} else {
					change_flag = true
				}
			r2key = r2_key
			break
			}
		}
		if !mod && !change_flag {
			l2nexthop_changes = append(l2nexthop_changes,new_db[r1_key])
		}
		if mod && mod_flag {
			old_l2nexthop = append(old_l2nexthop,old_db[r2key])
			new_l2nexthop = append(new_l2nexthop,new_db[r1_key])
			mod_flag = false
		}
	}
	if mod {
		return new_l2nexthop , old_l2nexthop
	} else {
		return l2nexthop_changes, nil
	}
}

func notify_L2nexthop_changes(new_db map[L2Nexthop_key]L2Nexthop_struct, old_db map[L2Nexthop_key]L2Nexthop_struct) {
	var old_keys []L2Nexthop_key
	var new_keys []L2Nexthop_key
	for  k , _ :=range old_db{
		old_keys = append(old_keys,k)	
	} 
	for  k , _ :=range new_db{
		new_keys = append(new_keys,k)	
	}
	if len(new_keys)- len(old_keys) > 0{   // ADDED
		log.Printf("Added new L2Nexthop len %d old L2Nexthop len %d\n",len(new_keys),len(old_keys))
		Added_L2Nexthop,_ := get_L2Nexthop_changes(new_keys,old_keys,new_db,old_db,false)
		if len(Added_L2Nexthop) != 0{
			for _,R := range Added_L2Nexthop{
				notify_L2nexthop_added(R)	
			}	
		}
	} else if len(new_keys)- len(old_keys) < 0 { // DELETED
		log.Printf("Deleted :new L2Nexthop len %d old L2Nexthop len %d\n",len(new_keys),len(old_keys))
		Deleted_L2Nexthop,_ := get_L2Nexthop_changes(old_keys,new_keys,new_db,old_db,false)
		if len(Deleted_L2Nexthop) != 0{
			for _,R := range Deleted_L2Nexthop{
				notify_L2nexthop_deleted(R)	
			}	
		}
	} else if len(new_keys)- len(old_keys) == 0 {
		if !reflect.DeepEqual(new_db, old_db) {  // UPDATED
			log.Printf("Updated :new route len %d old route len %d old: %+v \n new :%+v\n",len(new_keys),len(old_keys),old_keys,new_keys)
			new_L2Nexthop,old_L2Nexthop := get_L2Nexthop_changes(new_keys,old_keys,new_db,old_db,true)
			if len(new_L2Nexthop) != 0 && len(old_L2Nexthop)!=0 {
				for i:=0;i<len(new_L2Nexthop);i++ {
					notify_L2nexthop_updated(new_L2Nexthop[i],old_L2Nexthop[i])
				}
			}
		} else {
			log.Printf("No changes noticed in route DB\n")
		}
	}
}


func get_nexthop_changes(R1 []Nexthop_key, R2 []Nexthop_key,new_db map[Nexthop_key]Nexthop_struct, old_db map[Nexthop_key]Nexthop_struct,mod bool)([]Nexthop_struct,[]Nexthop_struct) {
	var old_nexthop []Nexthop_struct 
	var new_nexthop []Nexthop_struct
	var nexthop_changes []Nexthop_struct
	var n2key Nexthop_key
	for _,n1_key := range R1 {
		change_flag := false
		mod_flag := false  	
		for _,n2_key := range R2 {
			if reflect.DeepEqual(n1_key,n2_key) {
				if mod == true{
					if !reflect.DeepEqual(new_db[n1_key],old_db[n2_key]){
						mod_flag =true
						n2key = n2_key
					}
				} else {
						change_flag = true
				}
			break
			}
		}
		if !mod && !change_flag {	 
			nexthop_changes = append(nexthop_changes,new_db[n1_key])
		}
		if mod && mod_flag {
			old_nexthop = append(old_nexthop,old_db[n2key])
			new_nexthop = append(new_nexthop,new_db[n1_key])
			mod_flag = false
		}
	}
	if mod {
		return new_nexthop , old_nexthop
	} else {
		return nexthop_changes,nil
	}	
} 

func read_FDB() []FdbEntry_struct{
        var fdbs []Fdb_IP_Struct
        var macs []FdbEntry_struct
        var fs FdbEntry_struct

        CP,err := run([]string{"bridge", "-d", "-j", "fdb", "show", "br", "br-tenant", "dynamic"})
	if err!=nil || len(CP) == 0 {
		log.Fatal("FDB: Command error\n")
	}
        CPs := strings.Split(CP[2:len(CP)-3], "},{")
        for i := 0; i<len(CPs); i++{
                var fi Fdb_IP_Struct
                err := json.Unmarshal([]byte(fmt.Sprintf("{%v}",CPs[i])), &fi)
                if err != nil{
                        log.Println("error-",err)
                }
                fdbs = append(fdbs, fi)
        }
        for _, M := range fdbs{
                        fs = Parse_Fdb(M, fs)
                        if pre_filter_mac(fs){
                                macs = append(macs, fs)
                        }
        }
     //   log.Printf("Macs--%+v\n",macs)
        return macs
}



func notify_nexthop_changes(new_db map[Nexthop_key]Nexthop_struct,old_db map[Nexthop_key]Nexthop_struct ){
	var old_keys []Nexthop_key
	var new_keys []Nexthop_key
	for  k , _ :=range old_db{
		old_keys = append(old_keys,k)	
	} 
	for  k , _ :=range new_db{
		new_keys = append(new_keys,k)	
	}
	if len(new_keys)- len(old_keys) > 0{   // ADDED
		Added_nexthops,_ := get_nexthop_changes(new_keys,old_keys,new_db,old_db,false)
		if len(Added_nexthops) != 0{
			for _,R := range Added_nexthops{
				notify_nexthop_added(R)	
			}	
		}
	} else if len(new_keys)- len(old_keys) < 0{ // DELETED
		Deleted_nexthop,_ := get_nexthop_changes(old_keys,new_keys,new_db,old_db,false)
		if len(Deleted_nexthop) != 0{
				for _,R := range Deleted_nexthop{
					notify_nexthop_deleted(R)	
				}	
		}
	} else  if len(new_keys)- len(old_keys) == 0{
			if !reflect.DeepEqual(new_db, old_db) {  // UPDATED
				new_nexthop,old_nexthop := get_nexthop_changes(new_keys,old_keys,new_db,old_db,true)
			if len(new_nexthop) != 0 && len(old_nexthop)!=0 {
				for i:=0;i<len(new_nexthop);i++ {
					notify_nexthop_updated(new_nexthop[i],old_nexthop[i])	
				}	
			}
		} else {
		}
	}
}

func lookup_route(dst net.IP, V Vrf)Route_struct{
	// FIXME: If the semantic is to return the current entry of the NetlinkDB
	//  routing table, a direct lookup in Linux should only be done as fallback
	//  if there is no match in the DB.
	var CP string
	var err error
	if V.Vni!=0{
		CP,err = run([]string{"ip", "-j", "route", "get", dst.String(), "vrf" ,V.Name, "fibmatch"})
	} else{
		CP,err = run([]string{"ip", "-j", "route", "get", dst.String(), "fibmatch"})
	}
	if err != nil{
		log.Fatal("Command error\n")
		return Route_struct{} 
	}
	R := cmd_process_Rt(V, CP,254)
	log.Printf("%+v\n",R)
	if len(R.RS)!=0 {
		R1:= R.RS[0]
	// ###  Search the LatestRoutes DB snapshot if that exists, else
	// ###  the current DB Route table.
	var RouteTable map[Route_key]Route_struct
	if len(LatestRoutes)!=0{
			RouteTable = LatestRoutes
	} else {
		    RouteTable= Routes
	}		
	R_DB := RouteTable[R1.Key]
	if !reflect.ValueOf(R_DB).IsZero(){
		// Return the existing route in the DB
		return R_DB
	} else{
		// Return the just constructed non-DB route
		return R1
	}
	} else {
		log.Printf("Failed to lookup route {dst} in VRF {V}")
		return Route_struct{}
	}	
}

func (nexthop Nexthop_struct)annotate() {
	    nexthop.Metadata=make(map[interface{}]interface{})
	if (!reflect.ValueOf(nexthop.NH.Gw).IsZero()) && nexthop.NH.LinkIndex != 0 && strings.HasPrefix(Name_index[nexthop.NH.LinkIndex], nexthop.Vrf.Name+"-") && !nexthop.Local {
		nexthop.Nh_type = SVI
		link, _ := netlink.LinkByName(Name_index[nexthop.NH.LinkIndex])
		nexthop.Metadata["smac"] = link.Attrs().HardwareAddr.String()
		if (!reflect.ValueOf(nexthop.Neighbor).IsZero()) {
			if nexthop.Neighbor.Type == SVI{
				nexthop.Metadata["dmac"] = nexthop.Neighbor.Neigh0.HardwareAddr.String()
                nexthop.Metadata["egress_vport"] = nexthop.Neighbor.Metadata["vport_id"]
                nexthop.Metadata["vlan_id"] = nexthop.Neighbor.Metadata["vlan_id"]
                nexthop.Metadata["port_type"] = nexthop.Neighbor.Metadata["port_type"]
			}
		} else {
			nexthop.Resolved = false
			log.Printf("Failed to gather data for nexthop on physical port\n")
		}
    } else if (!reflect.ValueOf(nexthop.NH.Gw).IsZero()) && !nexthop.Local{
		for _, k := range phy_ports{
			if nexthop.NH.LinkIndex == k{
				nexthop.Nh_type = PHY
				link1, _ := netlink.LinkByName(Name_index[nexthop.NH.LinkIndex])
				nexthop.Metadata["smac"] =  link1.Attrs().HardwareAddr.String()
				nexthop.Metadata["egress_vport"] = phy_ports[nexthop.NH.Gw.String()]
			}
			if (!reflect.ValueOf(nexthop.Neighbor).IsZero()) {
				if nexthop.Neighbor.Type == PHY{
					nexthop.Metadata["dmac"] = nexthop.Neighbor.Neigh0.HardwareAddr.String()
				}
			} else{
				nexthop.Resolved = false	
				log.Printf("Failed to gather data for nexthop on physical port")
			}
		}	
	} else if (!reflect.ValueOf(nexthop.NH.Gw).IsZero()) && Name_index[nexthop.NH.LinkIndex] == fmt.Sprintf("br-%s",nexthop.Vrf.Name) && !nexthop.Local{
		nexthop.Nh_type = VXLAN
		//nexthop.Metadata["inner_smac"] = nexthop.vrf.Rmac   //Need infra DB support for the getting RMAC info in vrf
		if (reflect.ValueOf(nexthop.Vrf.Rmac).IsZero()){
			nexthop.Resolved = false
		}
		//nexthop.Metadata["Local_vtep_ip"] = nexthop.vrf.Vtep
                nexthop.Metadata["remote_vtep_ip"] = nexthop.NH.Gw.String()
                nexthop.Metadata["vni"] = string(nexthop.Vrf.Vni)
                if (!reflect.ValueOf(nexthop.Neighbor).IsZero()){
				//if nexthop.neighbor.Type == SVI{
				nexthop.Metadata["inner_dmac"] = nexthop.Neighbor.Neigh0.HardwareAddr.String()
				//GRD = InfraDB.get_VRF(vni=None)
				R := lookup_route(nexthop.NH.Gw, vrf_table[0])
				if !reflect.ValueOf(R).IsZero() {
				    // For now pick the first physical nexthop (no ECMP yet)
				    phy_nh := R.Nexthops[0]
				    link, _ := netlink.LinkByName(Name_index[phy_nh.NH.LinkIndex])
			            nexthop.Metadata["phy_smac"] = link.Attrs().HardwareAddr.String()
				    nexthop.Metadata["egress_vport"] = phy_ports[Name_index[phy_nh.NH.LinkIndex]]
				    if !reflect.ValueOf(phy_nh.Neighbor).IsZero(){
					nexthop.Metadata["phy_dmac"] = phy_nh.Neighbor.Neigh0.HardwareAddr.String()
				    } else{
					// The VXLAN nexthop can only be installed when the phy_nexthops are Resolved.
					nexthop.Resolved = false
		   		 }
			 } 
		} else{
			nexthop.Resolved = false	
			//return ""
		}
	} else {
		nexthop.Nh_type = ACC
		link1, err := netlink.LinkByName("rep-"+nexthop.Vrf.Name)
		if err != nil {	
			log.Printf("Error in getting rep information\n")
			//return ""
		}
		nexthop.Metadata["dmac"] = link1.Attrs().HardwareAddr.String()
		nexthop.Metadata["egress_vport"] = string(0xa) //ipu_db.vport_id_from_mac_address(mac)
		if (reflect.ValueOf(nexthop.Vrf.Vni).IsZero()){
			nexthop.Metadata["vlan_id"] = string(4089)
		} else {
			nexthop.Metadata["vlan_id"] = string(nexthop.Vrf.Vni)
		}
	}
	log.Printf("METADATA %+v\n",nexthop.Metadata)
	//return ""
}

func (L2N L2Nexthop_struct) annotate(){
        // Annotate certain L2 Nexthops with additional information from LB and GRD
	// TODO
	//LB := L2N.lb
        //if !(reflect.ValueOf(LB).IsZero()) {
          //  if L2N.Type == SVI {
                // MAC address learned on SVI interface of bridge
            //    if reflect.ValueOf(LB.Svi).IsZero() {
	//		log.Printf("Error in L2nexthop annotate\n")
	//		return
	//	}
          //      L2N.Metadata["vrf_id"] = LB.Svi.vni
	    if L2N.Type == VXLAN {
                //# Remote EVPN MAC address learned on the VXLAN interface
		        //# The L2 nexthop must have a destination IP address in dst
                L2N.Resolved = false
                L2N.Metadata["local_vtep_ip"] = "0.0.0.0" //LB.vtep
                L2N.Metadata["remote_vtep_ip"] = L2N.Dst
                L2N.Metadata["vni"] = 2000 //LB.vni
                //# The below physical nexthops are needed to transmit the VXLAN-encapsuleted packets
                //# directly from the nexthop table to a physical port (and avoid another recirculation
                //# for route lookup in the GRD table.)
		//GRD = InfraDB.get_VRF(vni=None)  TODO : need infraDB for fetching 
		VRF := get_vrf_name("GRD")
		R := lookup_route(L2N.Dst, VRF)
		if !reflect.ValueOf(R).IsZero(){
			//  # For now pick the first physical nexthop (no ECMP yet)
			phy_nh := R.Nexthops[0]
			link, _ := netlink.LinkByName(Name_index[phy_nh.NH.LinkIndex])
			L2N.Metadata["phy_smac"] = link.Attrs().HardwareAddr.String()
			L2N.Metadata["egress_vport"] = phy_ports[Name_index[phy_nh.NH.LinkIndex]]
			if !reflect.ValueOf(phy_nh.Neighbor).IsZero(){
				if (phy_nh.Neighbor.Type == PHY){
					L2N.Metadata["phy_dmac"] = phy_nh.Neighbor.Neigh0.HardwareAddr.String()
				} else {
					log.Printf("Error: Neighbor type not PHY\n")
					return
				}
				L2N.Resolved = false
			}
		}
	} else if L2N.Type == BRIDGE_PORT {
		// BridgePort as L2 nexthop
		L2N.Metadata["vport_id"] = "2000" // TODO L2N.bp.vport_id
		L2N.Metadata["port_type"] = "host" // TODO L2N.bp.Type
	}
//}	
	//print(f"Annotated {self}: extra={self.metadata}")
}

func (fdb FdbEntry_struct) annotate(){
	if fdb.Vlan_id == 0{
		return
	}
	//TODO
	//if not self.lb: return
        
	fdb.Metadata = make(map[interface{}]interface{})
	L2N := fdb.Nexthop
	if (!reflect.ValueOf(L2N).IsZero()){
		fdb.Metadata["nh_id"] = L2N.Id
		//TODO
		/*if L2N.Type == VXLAN{
			sibling = NetlinkDB.LatestFDB.get((None, self.mac))
                L2N.dst = sibling.nexthop.dst if sibling else None
            # The relevant directions for the FDB entry are derived from the nexthop type
            
		}*/
		if L2N.Type == VXLAN{
			fdb.Metadata["direction"] = string(TX)
		} else if L2N.Type == SVI || L2N.Type == BRIDGE_PORT{
			fdb.Metadata["direction"] = string(RX_TX)
		} else{
			fdb.Metadata["direction"] = "NONE"
		}
	    //TODO
	    //logger.debug(f"Annotated {self}: extra={self.Metadata}")
        }
}


func annotate_db_entries(){
	
	for _,NH := range LatestNexthop{
		NH.annotate()
	}	
	for _,R := range LatestRoutes{
		R.annotate()
	}	

	for _,M := range LatestFDB{
		M.annotate()
	}	
	for _,L2N := range LatestL2Nexthop{
		L2N.annotate()
	}	

}


func install_filter_route(Rt *Route_struct)bool{
	var nh  []Nexthop_struct
	for _,n:=range Rt.Nexthops{
		if n.Resolved == true{
			nh = append(nh,n)
		}
	}
	Rt.Nexthops = nh
	keep:= check_Rtype(Rt.Nl_type) &&  len(nh)!=0  && strings.Compare(Rt.Route0.Dst.IP.String(),"0.0.0.0")!=0
	return keep
}

 
func check_nh_type(N_type int) bool {
	ntype := []int { PHY , SVI , ACC ,VXLAN}
	for _, i := range ntype{
		if i== N_type {
			return true
		}
	}
	return false 
}

func install_filter_NH(Nh Nexthop_struct) bool{
	check := check_nh_type(Nh.Nh_type)
	keep := check && Nh.Resolved && len(Nh.Route_refs )!=0
//	if !keep {
//		log.Printf("install_filter: dropping {%v}",Nh)
//	}
	return keep 
}

func check_fdb_type(Type int) bool{
        var port_type= []int{BRIDGE_PORT, VXLAN }
        for  _,port := range port_type {
                if port == Type {
                        return true
                }
        }
        return false
}


func install_filter_FDB(fdb FdbEntry_struct) bool{
        // Drop entries w/o VLAN ID or associated LogicalBridge ...
        // ... other than with L2 nexthops of type VXLAN and BridgePort ...
        // ... and VXLAN entries with unresolved underlay nextop.
        keep := reflect.ValueOf(fdb.Vlan_id).IsZero() && /*reflect.ValueOf(fdb.lb).IsZero() &&*/ check_fdb_type(fdb.Type) && fdb.Nexthop.Resolved
        if keep == false{
        //    log.Printf("install_filter: dropping {%v}",fdb)
        }
        return keep
}

func install_filter_L2N(l2n L2Nexthop_struct) bool {
        keep := (reflect.ValueOf(l2n.Type).IsZero() && l2n.Resolved == true  && reflect.ValueOf(l2n.Fdb_refs).IsZero())
        if keep == false{
    //        log.Printf("install_filter FDB: dropping {%+v}",l2n)
        }
        return keep
}


func apply_install_filters(){
	for K, R := range LatestRoutes{
		if  (install_filter_route(&R)!=true){
			//Remove route from its nexthop(s)
				delete(LatestRoutes,K)
		}
	}

	for k, NH := range LatestNexthop{
		if install_filter_NH(NH){
		     delete(LatestNexthop,k)
		}
	}

	for k, M := range LatestFDB{
		if install_filter_FDB(M){
		     delete(LatestFDB,k)
		}
	}
	for k, L2 := range LatestL2Nexthop{
		if ! install_filter_L2N(L2){
			delete(LatestL2Nexthop,k)
		}
	}

}

func resync_with_kernel() {
	// Build a new DB snapshot from netlink and other sources
	read_latest_netlink_state()
	// Annotate the latest DB entries
	annotate_db_entries()
	//Filter the latest DB to retain only entries to be installed
	apply_install_filters()
	// Compute changes between current and latest DB versions and
    // inform subscribers about the changes
	notify_route_changes(LatestRoutes, Routes)
        notify_nexthop_changes(LatestNexthop, Nexthops)
        notify_FDB_changes(LatestFDB, FDB)
        notify_L2nexthop_changes(LatestL2Nexthop, L2Nexthops)
	// with db_lock:
	Routes = LatestRoutes
	Nexthops = LatestNexthop
	Neighbors = LatestNeighbors
	FDB = LatestFDB
	L2Nexthops = LatestL2Nexthop
	delete_latestDB()

}

func delete_latestDB(){
	LatestRoutes = make(map[Route_key]Route_struct)
        LatestNeighbors = make(map[Neigh_key]Neigh_Struct)
        LatestNexthop = make(map[Nexthop_key]Nexthop_struct)
        LatestFDB  =  make(map[FDB_key]FdbEntry_struct)
        LatestL2Nexthop = make(map[L2Nexthop_key]L2Nexthop_struct)
}

func monitor_netlink(p4_enabled bool) {
	for stop_monitoring != true {
		log.Printf("netlink: Polling netlink databases.")
		resync_with_kernel()
		log.Printf("netlink: Polling netlink databases completed.")
		time.Sleep(time.Duration(poll_interval) * time.Second)
	}
	log.Println("netlink: Stopped periodic polling. Waiting for Infra DB cleanup to finish.\n")
	time.Sleep(2 * time.Second)
	log.Println("netlink: One final netlink poll to identify what's still left.")
	resync_with_kernel()
	// Inform subscribers to delete configuration for any still remaining Netlink DB objects.
	log.Println("netlink: Delete any residual objects in DB")
	for _,R :=  range Routes{
		notify_route_deleted(R)
	}
	for _,NH := range Nexthops{
		notify_nexthop_deleted(NH)
	}
	for _,M := range  FDB {
		notify_fdb_deleted(M)
	}
	log.Println("netlink: DB cleanup completed.")
	wg.Done()
}

func Init() {
	LOG_FILE = "./ipu_infra_manager.log"
	// open log file
	logFile, err := os.OpenFile(LOG_FILE, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Panic(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	var config Config_t
	yfile, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Fatal(err)
		//os.Exit(0)
	}
	err2 := yaml.Unmarshal(yfile, &config)
	if err2 != nil {
		log.Fatal(err2)
	}
	wg.Add(1)
	poll_interval = config.Netlink.Poll_interval
	log.Println(poll_interval)
	br_tenant = config.Linux_frr.Br_tenant
	log.Println(br_tenant)
	nl_enabled := config.Netlink.Enable
	if nl_enabled != true {
		log.Println("netlink_monitor disabled")
		return
	}
	for i:=0 ; i< len(config.Netlink.Phy_ports) ;i++ {
		phy_ports[config.Netlink.Phy_ports[i].Name]=config.Netlink.Phy_ports[i].Vsi
	}
	getlink()
	get_vrf_info()
	go monitor_netlink(config.P4.Enable) //monitor Thread started
	//log.Println("Started netlink_monitor thread with {poll_interval} s poll interval.")
	//	time.Sleep(1 * time.Second)
	//	stop_monitoring = true
	wg.Wait()
}
