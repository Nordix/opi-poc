package p4translation
import(
	"reflect"
	"strings"
	"math"
	"strconv"
	"net"
	p4client "vendor/intel/p4runtime/pkg/p4driverAPI"
)
var TcamPrefix = struct{
	GRD,VRF uint32
}{
	GRD: 1,
        VRF: 2,

}

var Direction = struct {
	RX,Tx int
}{
	RX: 0,
	Tx: 1,
}

var Vlan = struct{
	GRD,PHY0,PHY1,PHY2,PHY3 uint16
}{
	GRD: 4089,
    PHY0: 4090,
    PHY1: 4091,
    PHY2: 4092,
    PHY3: 4093,
}

var PortId = struct{
	PHY0,PHY1,PHY2,PHY3 int
}{
	PHY0: 0,
    PHY1: 1,
    PHY2: 2,
    PHY3: 3,
}

var EntryType = struct{
	IGNORE_PTR,L2_FLOODING_PTR,PTR_MIN_RANGE,PTR_MAX_RANGE int
}{
	IGNORE_PTR: 0,
	L2_FLOODING_PTR: 1,
	PTR_MIN_RANGE: 2,
	PTR_MAX_RANGE: int(math.Pow(2,16)) - 1,
}

var ModPointer = struct{
	IGNORE_PTR,L2_FLOODING_PTR,PTR_MIN_RANGE,PTR_MAX_RANGE uint32
}{
	IGNORE_PTR: 0,
	L2_FLOODING_PTR: 1,
	PTR_MIN_RANGE: 2,
	PTR_MAX_RANGE: uint32(math.Pow(2,16)) - 1,
}

var TrieIndex = struct{
	TRIEIDX_MIN_RANGE,TRIEIDX_MAX_RANGE int
}{
	TRIEIDX_MIN_RANGE: 1,
	TRIEIDX_MAX_RANGE: int(math.Pow(2,16)) - 1,
}

var RefCountOp = struct{
	RESET,INCREMENT,DECREMENT int
}{
	RESET: 0,
	INCREMENT: 1,
	DECREMENT: 2,
}

type Table string

const(
	L3_RT    = "linux_networking_control.l3_routing_table"       // VRFs routing table in LPM
//                            TableKeys (
//                                ipv4_table_lpm_root2,  // Exact
//                                vrf,                   // LPM
//                                direction,             // LPM
//                                dst_ip,                // LPM
//                            )
//                            Actions (
//                                set_neighbor(neighbor),
//                            )
    L3_RT_HOST         = "linux_networking_control.l3_lem_table"
//                            TableKeys (
//                                vrf,                   // Exact
//                                direction,             // Exact
//                                dst_ip,                // Exact
//                            )
//                            Actions (
//                                set_neighbor(neighbor)
//                            )
    L3_NH              = "linux_networking_control.l3_nexthop_table"       // VRFs next hop table
//                            TableKeys (
//                                neighbor,              // Exact
//                                bit32_zeros,           // Exact
//                            )
//                            Actions (
//                               push_dmac_vlan(mod_ptr, vport)
//                               push_vlan(mod_ptr, vport)
//                               push_mac(mod_ptr, vport)
//                               push_outermac_vxlan_innermac(mod_ptr, vport)
//                               push_mac_vlan(mod_ptr, vport)
//                            )
    PHY_IN_IP          = "linux_networking_control.phy_ingress_ip_table"    // PHY ingress table - IP traffic
//                           TableKeys(
//                               port_id,                // Exact
//                               bit32_zeros,            // Exact
//                           )
//                           Actions(
//                               set_vrf_id(tcam_prefix, vport, vrf),
//                           )                          
    PHY_IN_ARP         = "linux_networking_control.phy_ingress_arp_table"   // PHY ingress table - ARP traffic
//                           TableKeys(
//                               port_id,                // Exact
//                               bit32_zeros,            // Exact
//                           )
//                           Actions(
//                               fwd_to_port(port)
//                           )
    PHY_IN_VXLAN       = "linux_networking_control.phy_ingress_vxlan_table" // PHY ingress table - VXLAN traffic
//                           TableKeys(
//                               dst_ip
//                               vni,
//                               da
//                           )
//                           Actions(
//                               pop_vxlan_set_vrf_id(mod_ptr, tcam_prefix, vport, vrf),
//                           )
    PHY_IN_VXLAN_L2    = "linux_networking_control.phy_ingress_vxlan_vlan_table"
//                           Keys {
//                               dst_ip                  // Exact
//                               vni                     // Exact
//                           }
//                           Actions(
//                               pop_vxlan_set_vlan_id(mod_ptr, vlan_id, vport)
//                           )
    POD_IN_ARP_ACCESS  = "linux_networking_control.vport_arp_ingress_table"
//                       Keys {
//                           vsi,                        // Exact
//                           bit32_zeros                 // Exact
//                       }
//                       Actions(
//                           fwd_to_port(port),
//                           send_to_port_mux_access(mod_ptr, vport)
//                       )
    POD_IN_ARP_TRUNK   = "linux_networking_control.tagged_vport_arp_ingress_table"
//                       Key {
//                           vsi,                        // Exact
//                           vid                         // Exact
//                       }
//                       Actions(
//                           send_to_port_mux_trunk(mod_ptr, vport),
//                           fwd_to_port(port),
//                           pop_vlan(mod_ptr, vport)
//                       )
    POD_IN_IP_ACCESS   = "linux_networking_control.vport_ingress_table"
//                       Key {
//                           vsi,                        // Exact
//                           bit32_zeros                 // Exact
//                       }
//                       Actions(
//                          fwd_to_port(port)
//                          set_vlan(vlan_id, vport)
//                       )
    POD_IN_IP_TRUNK    = "linux_networking_control.tagged_vport_ingress_table"
//                       Key {
//                           vsi,                        // Exact
//                           vid                         // Exact
//                       }
//                       Actions(
//                           //pop_vlan(mod_ptr, vport)
//                           //pop_vlan_set_vrfid(mod_ptr, vport, tcam_prefix, vrf)
//                           set_vlan_and_pop_vlan(mod_ptr, vlan_id, vport)
//                       )
    POD_IN_SVI_ACCESS  = "linux_networking_control.vport_svi_ingress_table"
//                       Key {
//                           vsi,                        // Exact
//                           da                          // Exact
//                       }
//                       Actions(
//                           set_vrf_id_tx(tcam_prefix, vport, vrf)
//                           fwd_to_port(port)
//                       )
    POD_IN_SVI_TRUNK   = "linux_networking_control.tagged_vport_svi_ingress_table"
//                       Key {
//                           vsi,                        // Exact
//                           vid,                        // Exact
//                           da                          // Exact
//                       }
//                       Actions(
//                           pop_vlan_set_vrf_id(tcam_prefix, mod_ptr, vport, vrf)
//                       )
    PORT_MUX_IN        = "linux_networking_control.port_mux_ingress_table"
//                       Key {
//                           vsi,                        // Exact
//                           vid                         // Exact
//                       }
//                       Actions(
//                           set_def_vsi_loopback()
//                           pop_ctag_stag_vlan(mod_ptr, vport),
//                           pop_stag_vlan(mod_ptr, vport)
//                       )
//    PORT_MUX_RX        = "linux_networking_control.port_mux_rx_table"
//                       Key {
//                           vid,                        // Exact
//                           bit32_zeros                 // Exact
//                       }
//                       Actions(
//                           pop_ctag_stag_vlan(mod_ptr, vport),
//                           pop_stag_vlan(mod_ptr, vport)
//                       )
    PORT_MUX_FWD       = "linux_networking_control.port_mux_fwd_table"
//                       Key {
//                           bit32_zeros                 // Exact
//                       }
//                       Actions(
//                           "linux_networking_control.send_to_port_mux(vport)"
//                       )
    L2_FWD_LOOP        = "linux_networking_control.l2_fwd_rx_table"
//                       Key {
//                           da                          // Exact (MAC)
//                       }
//                       Actions(
//                           l2_fwd(port)
//                       )
    L2_FWD             = "linux_networking_control.l2_dmac_table"
//                       Key {
//                           vlan_id,                    // Exact
//                           da,                         // Exact
//                           direction                   // Exact
//                       }
//                       Actions(
//                           set_neighbor(neighbor)
//                       )
    L2_NH              = "linux_networking_control.l2_nexthop_table"
//                       Key {
//                           neighbor                    // Exact
//                           bit32_zeros                 // Exact
//                       }
//                       Actions(
//                           //push_dmac_vlan(mod_ptr, vport)
//                           push_stag_ctag(mod_ptr, vport)
//                           push_vlan(mod_ptr, vport)
//                           fwd_to_port(port)
//                           push_outermac_vxlan(mod_ptr, vport)
//                       )
    TCAM_ENTRIES       = "linux_networking_control.ecmp_lpm_root_lut1"
//                       Key {
//                           tcam_prefix,                 // Exact
//                           MATCH_PRIORITY,              // Exact
//                       }
//                       Actions(
//                           None(ipv4_table_lpm_root1)
//                       )

)

type ModTable string

const(
	PUSH_VLAN      = "linux_networking_control.vlan_push_mod_table"
//                        src_action="push_vlan"
//			  Actions(
// 				vlan_push(pcp, dei, vlan_id),
//                        )
    PUSH_MAC_VLAN  = "linux_networking_control.mac_vlan_push_mod_table"
//                       src_action=""
//                       Actions(
//                          update_smac_dmac_vlan(src_mac_addr, dst_mac_addr, pcp, dei, vlan_id)
    PUSH_DMAC_VLAN = "linux_networking_control.dmac_vlan_push_mod_table"
//                        src_action="push_dmac_vlan",
//                       Actions(
//                           dmac_vlan_push(pcp, dei, vlan_id, dst_mac_addr),
//                        )
    MAC_MOD        = "linux_networking_control.mac_mod_table"
//                       src_action="push_mac"
//                        Actions(
//                            update_smac_dmac(src_mac_addr, dst_mac_addr),
//                        )
    PUSH_VXLAN_HDR = "linux_networking_control.omac_vxlan_imac_push_mod_table"
//                       src_action="push_outermac_vxlan_innermac"
//                       Actions(
//                           omac_vxlan_imac_push(outer_smac_addr,
//                                                outer_dmac_addr,
//                                                src_addr,
//                                                dst_addr,
//                                                dst_port,
//                                                vni,
//                                                inner_smac_addr,
//                                                inner_dmac_addr)
//                       )
    POD_OUT_ACCESS  = "linux_networking_control.vlan_encap_ctag_stag_mod_table"
//                       src_actions="send_to_port_mux_access"
//                       Actions(
//                           vlan_push_access(pcp, dei, ctag_id, pcp_s, dei_s, stag_id, dst_mac)
//                       )
    POD_OUT_TRUNK   = "linux_networking_control.vlan_encap_stag_mod_table"
//                       src_actions="send_to_port_mux_trunk"
//                       Actions(
//                           vlan_push_trunk(pcp, dei, stag_id, dst_mac)
//                       )
    POP_CTAG_STAG   = "linux_networking_control.vlan_ctag_stag_pop_mod_table"
//                       src_actions=""
//                       Actions(
//                           vlan_ctag_stag_pop()
//                       )
    POP_STAG        = "linux_networking_control.vlan_stag_pop_mod_table"
//                       src_actions=""
//                       Actions(
//                           vlan_stag_pop()
//                       )
    PUSH_QNQ_FLOOD  = "linux_networking_control.vlan_encap_ctag_stag_flood_mod_table"
//                       src_action="l2_nexthop_table.push_stag_ctag()"
//                       Action(
//                           vlan_push_stag_ctag_flood()
//                       )
    PUSH_VXLAN_OUT_HDR = "linux_networking_control.omac_vxlan_push_mod_table"
//                      src_action="l2_nexthop_table.push_outermac_vxlan()"
//			Action(
//                           omac_vxlan_push(outer_smac_addr, outer_dmac_addr, src_addr, dst_addr, dst_port, vni)
//                       )

)

func _to_egress_vsi(vsi_id int) int{
	return vsi_id + 16
}
type PhyPort struct{
	id int
	vsi int
	mac string
}
func (p PhyPort) PhyPort_Init(id int, vsi string, mac string) PhyPort{
	p.id = id
	p.vsi,_ = strconv.Atoi(vsi)
	p.mac = mac

	return p
}
type GrpcPairPort struct{
	vsi int
	mac string
	peer map[string]string
}
func (g GrpcPairPort) GrpcPairPort_Init(vsi string, mac string) GrpcPairPort{
	g.vsi, _= strconv.Atoi(vsi)
	g.mac= mac
	return g
}

func (g GrpcPairPort) set_remote_peer(peer [2]string) GrpcPairPort{
	g.peer = make(map[string]string)
	g.peer["vsi"]= peer[0]
	g.peer["mac"]= peer[1]
	return g
}

type L3Decoder struct{
	_mux_vsi uint16
	_default_vsi int
	_phy_ports []PhyPort
	_grpc_ports []GrpcPairPort
	PhyPort
	GrpcPairPort
}

func (l L3Decoder) L3DecoderInit(representors map[string][2]string)(L3Decoder){
	s := L3Decoder{
		_mux_vsi: l.set_mux_vsi(representors),
		_default_vsi: 0x25,
		_phy_ports: l._get_phy_info(representors),
		_grpc_ports: l._get_grpc_info(representors),
	}
	return s
}
func (l L3Decoder) set_mux_vsi(representors map[string][2]string) uint16{
	var a string = representors["vrf_mux"][0]
	var mux_vsi, _ = strconv.Atoi(a)
	return uint16(mux_vsi)
}
func (l L3Decoder) _get_phy_info(representors map[string][2]string) []PhyPort{
	var enabled_ports []PhyPort
	var vsi string
	var mac string
	var p = reflect.TypeOf(PortId)
	for i := 0; i < p.NumField(); i++ {
		var k = p.Field(i).Name
		var key = strings.ToLower(k) + "_rep"
		for k,_= range representors{
			if (key == k){
				vsi = representors[key][0]
				mac = representors[key][1]
				enabled_ports = append(enabled_ports, l.PhyPort_Init(i, vsi, mac))
			}
		}
	}
	return enabled_ports //should return tuple
}

func (l L3Decoder) _get_grpc_info(representors map[string][2]string) []GrpcPairPort {
	var acc_host GrpcPairPort
	var host_port GrpcPairPort
	var grpc_ports []GrpcPairPort

	var acc_vsi string = representors["grpc_acc"][0]
	var acc_mac string = representors["grpc_acc"][1]
	acc_host = acc_host.GrpcPairPort_Init(acc_vsi, acc_mac) //??

	var host_vsi string = representors["grpc_host"][0]
	var host_mac string = representors["grpc_host"][1]
	host_port = host_port.GrpcPairPort_Init(host_vsi, host_mac) //??

	var acc_peer [2]string = representors["grpc_host"]
	var host_peer [2]string = representors["grpc_acc"]
	acc_host = acc_host.set_remote_peer(acc_peer)

	host_port = host_port.set_remote_peer(host_peer)

	grpc_ports = append(grpc_ports, acc_host, host_port)
	return grpc_ports
}
func (l L3Decoder) Static_additions() []interface{}{
	var tcam_prefix= TcamPrefix.GRD
	var entries []interface{}

	entries = append(entries, p4client.TableEntry{
		Tablename: POD_IN_IP_TRUNK,
		TableField: p4client.TableField{
			FieldValue: map[string][2]interface{}{
				"vsi": {l._mux_vsi,"exact"},
				"vid": {Vlan.GRD,"exact"},
			},
			Priority: int32(0),
		},
		Action: p4client.Action{
			Action_name : "linux_networking_control.pop_vlan_set_vrfid",
			Params : []interface{}{ModPointer.IGNORE_PTR, uint32(0), tcam_prefix, uint32(0)},
		},
	})

	for _ , port := range l._grpc_ports {
		var peer_vsi, _ = strconv.Atoi(port.peer["vsi"])
		var peer_da, _ =  net.ParseMAC(port.peer["mac"])
		var port_da, _ = net.ParseMAC(port.mac)
		entries = append(entries, p4client.TableEntry{
			Tablename:POD_IN_SVI_ACCESS,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"vsi":{uint16(port.vsi),"exact"},
					"da":{peer_da,"exact"},
				},
				Priority: int32(0),
			},
			Action: p4client.Action{
				Action_name : "linux_networking_control.fwd_to_port",
				Params: []interface{}{uint32(_to_egress_vsi(peer_vsi))},
			},
		},
		p4client.TableEntry{
                        Tablename: L2_FWD_LOOP,
			TableField: p4client.TableField{
                                FieldValue: map[string][2]interface{}{
                                        "da":{port_da,"exact"},
                                },
                                Priority: int32(0),
                        },
                        Action: p4client.Action{
                                Action_name : "linux_networking_control.l2_fwd",
                                Params: []interface{}{uint32(_to_egress_vsi(port.vsi))},
                        },
                })
	}
	for _, port := range l._phy_ports {
		entries = append(entries, p4client.TableEntry{
			Tablename: PHY_IN_IP,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"port_id": {uint16(port.id),"exact"},
					"bit32_zeros": {uint32(0),"exact"},
				},
				Priority: int32(0),
			},
			Action: p4client.Action{
				Action_name : "linux_networking_control.set_vrf_id",
				Params: []interface{}{tcam_prefix, uint32(_to_egress_vsi(l._default_vsi)),  uint32(0)},
			},
		},
		p4client.TableEntry{
			Tablename: PHY_IN_ARP,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"port_id": {uint16(port.id),"exact"},
                                        "bit32_zeros": {uint32(0),"exact"},
                                },
                                Priority: int32(0),
                        },
			Action: p4client.Action{
				Action_name : "linux_networking_control.fwd_to_port",
				Params: []interface{}{uint32(_to_egress_vsi(port.vsi))},
			},
		},
		p4client.TableEntry{
			Tablename: POD_IN_IP_ACCESS,
			TableField: p4client.TableField{
                                FieldValue: map[string][2]interface{}{
					"vsi": {uint16(port.vsi),"exact"},
					"bit32_zeros": {uint32(0),"exact"},
				},
				Priority: int32(0),
			},
			Action: p4client.Action{
				Action_name : "linux_networking_control.fwd_to_port",
				Params: []interface{}{uint32(port.id)},
			},
		},
		p4client.TableEntry{
                        Tablename: POD_IN_ARP_ACCESS,
			TableField: p4client.TableField{
                                FieldValue: map[string][2]interface{}{
                                        "vsi": {uint16(port.vsi),"exact"},
                                        "bit32_zeros": {uint32(0),"exact"},
                                },
                                Priority: int32(0),
                        },
                        Action: p4client.Action{
                                Action_name : "linux_networking_control.fwd_to_port",
                                Params: []interface{}{uint32(port.id)},
                        },
                })
	}

	return entries
}


func (l L3Decoder) Static_deletions() []interface{}{
	var entries []interface{}
	for _, port := range l._phy_ports {
                entries = append(entries, p4client.TableEntry{
                        Tablename: PHY_IN_IP,
                        TableField: p4client.TableField{
                                FieldValue: map[string][2]interface{}{
                                        "port_id": {uint16(port.id),"exact"},
                                        "bit32_zeros": {uint32(0),"exact"},
                                },
                                Priority: int32(0),
                        },
                },
                p4client.TableEntry{
                        Tablename: PHY_IN_ARP,
                        TableField: p4client.TableField{
                                FieldValue: map[string][2]interface{}{
                                        "port_id": {uint16(port.id),"exact"},
                                        "bit32_zeros": {uint32(0),"exact"},
                                },
                                Priority: int32(0),
                        },
                },
                p4client.TableEntry{
                        Tablename: POD_IN_IP_ACCESS,
                        TableField: p4client.TableField{
                                FieldValue: map[string][2]interface{}{
                                        "vsi": {uint16(port.vsi),"exact"},
                                        "bit32_zeros": {uint32(0),"exact"},
                                },
                                Priority: int32(0),
                        },
                },
                p4client.TableEntry{
                        Tablename: POD_IN_ARP_ACCESS,
                        TableField: p4client.TableField{
                                FieldValue: map[string][2]interface{}{
                                        "vsi": {uint16(port.vsi),"exact"},
                                        "bit32_zeros": {uint32(0),"exact"},
                                },
                                Priority: int32(0),
                        },
                })
        }
	for _ , port := range l._grpc_ports {
                var peer_da, _ =  net.ParseMAC(port.peer["mac"])
                var port_da, _ = net.ParseMAC(port.mac)
                entries = append(entries, p4client.TableEntry{
                        Tablename:POD_IN_SVI_ACCESS,
                        TableField: p4client.TableField{
                                FieldValue: map[string][2]interface{}{
                                        "vsi":{uint16(port.vsi),"exact"},
                                        "da":{peer_da,"exact"},
                                },
                                Priority: int32(0),
                        },
                },
                p4client.TableEntry{
                        Tablename: L2_FWD_LOOP,
                        TableField: p4client.TableField{
                                FieldValue: map[string][2]interface{}{
                                        "da":{port_da,"exact"},
                                },
                                Priority: int32(0),
                        },
                })
        }
	entries = append(entries, p4client.TableEntry{
                Tablename: POD_IN_IP_TRUNK,
                TableField: p4client.TableField{
                        FieldValue: map[string][2]interface{}{
                                "vsi": {l._mux_vsi,"exact"},
                                "vid": {Vlan.GRD,"exact"},
                        },
                        Priority: int32(0),
                },
        })
	return entries
}

type PodDecoder struct{
	port_mux_ids [2]string
	_port_mux_vsi int
	_port_mux_mac string
	vrf_mux_ids [2]string
	_vrf_mux_vsi int
	_vrf_mux_mac string
	FLOOD_MOD_PTR uint32
	FLOOD_NH_ID uint16
}

func (p PodDecoder) PodDecoderInit(representors map[string][2]string) PodDecoder{
	p.port_mux_ids = representors["port_mux"]
	p.vrf_mux_ids = representors["vrf_mux"]

	var port_mux_vsi, _  = strconv.Atoi(p.port_mux_ids[0])
        var vrf_mux_vsi, _ = strconv.Atoi(p.vrf_mux_ids[0])

	p._port_mux_vsi = port_mux_vsi
	p._port_mux_mac = p.port_mux_ids[1]
	p._vrf_mux_vsi = vrf_mux_vsi
	p._vrf_mux_mac = p.vrf_mux_ids[1]
	p.FLOOD_MOD_PTR = ModPointer.L2_FLOODING_PTR
        p.FLOOD_NH_ID = uint16(0)
	return p
}

func (p PodDecoder) Static_additions() []interface{}{
	var port_mux_da, _ = net.ParseMAC(p._port_mux_mac)
	var vrf_mux_da, _ = net.ParseMAC(p._vrf_mux_mac)
	var entries []interface{}
	entries = append(entries, p4client.TableEntry{
		Tablename: PORT_MUX_FWD,
		TableField: p4client.TableField{
			FieldValue: map[string][2]interface{}{
				"bit32_zeros": {uint32(0),"exact"},
			},
				Priority: int32(0),
			},
			Action: p4client.Action{
				Action_name : "linux_networking_control.send_to_port_mux",
				Params: []interface{}{uint32(_to_egress_vsi(p._port_mux_vsi))},
			},
		},
		p4client.TableEntry{
			Tablename: PORT_MUX_IN,
			TableField: p4client.TableField{
				FieldValue: map[string][2]interface{}{
					"vsi": {uint16(p._port_mux_vsi),"exact"},
					"vid": {Vlan.PHY0,"exact"},
                                },
                                Priority: int32(0),
                        },
			Action: p4client.Action{
				Action_name : "linux_networking_control.set_def_vsi_loopback",
				Params: []interface{}{uint32(0)},
			},
		},
		p4client.TableEntry{
                        Tablename: L2_FWD_LOOP,
                        TableField: p4client.TableField{
                                FieldValue: map[string][2]interface{}{
					"da":{port_mux_da,"exact"},
                                },
                                Priority: int32(0),
                        },
                        Action: p4client.Action{
                                Action_name : "linux_networking_control.l2_fwd",
                                Params: []interface{}{uint32(_to_egress_vsi(p._port_mux_vsi))},
                        },
                },
		p4client.TableEntry{
                        Tablename: L2_FWD_LOOP,
                        TableField: p4client.TableField{
                                FieldValue: map[string][2]interface{}{
                                        "da":{vrf_mux_da,"exact"},
                                },
                                Priority: int32(0),
                        },
                        Action: p4client.Action{
                                Action_name : "linux_networking_control.l2_fwd",
                                Params: []interface{}{uint32(_to_egress_vsi(p._vrf_mux_vsi))},
                        },
                },
		//NH entry for flooding
		p4client.TableEntry{
                        Tablename: PUSH_QNQ_FLOOD,
                        TableField: p4client.TableField{
                                FieldValue: map[string][2]interface{}{
                                        "mod_blob_ptr":{p.FLOOD_MOD_PTR,"exact"},
                                },
                                Priority: int32(0),
                        },
                        Action: p4client.Action{
                                Action_name : "linux_networking_control.vlan_push_stag_ctag_flood",
                                Params: []interface{}{uint32(0)},
                        },
                },
		p4client.TableEntry{
                        Tablename: L2_NH,
                        TableField: p4client.TableField{
                                FieldValue: map[string][2]interface{}{
                                        "neighbor":{p.FLOOD_NH_ID,"exact"},
					"bit32_zeros": {uint32(0),"exact"},
                                },
                                Priority: int32(0),
                        },
                        Action: p4client.Action{
                                Action_name : "linux_networking_control.push_stag_ctag",
                                Params: []interface{}{p.FLOOD_MOD_PTR, uint32(_to_egress_vsi(p._vrf_mux_vsi))},
                        },
                })
		return entries
}

func (p PodDecoder) Static_deletions() []interface{}{
        var entries []interface{}
        var port_mux_da, _ = net.ParseMAC(p._port_mux_mac)
        var vrf_mux_da, _ = net.ParseMAC(p._vrf_mux_mac)
        entries = append(entries, p4client.TableEntry{
                Tablename: PORT_MUX_FWD,
                TableField: p4client.TableField{
                        FieldValue: map[string][2]interface{}{
                                "bit32_zeros": {uint32(0),"exact"},
                        },
                                Priority: int32(0),
                        },
                },
                p4client.TableEntry{
                        Tablename: PORT_MUX_IN,
                        TableField: p4client.TableField{
                                FieldValue: map[string][2]interface{}{
                                        "vsi": {uint16(p._port_mux_vsi),"exact"},
                                        "vid": {Vlan.PHY0,"exact"},
                                },
                                Priority: int32(0),
                        },
                },
                p4client.TableEntry{
                        Tablename: L2_FWD_LOOP,
                        TableField: p4client.TableField{
                                FieldValue: map[string][2]interface{}{
                                        "da":{port_mux_da,"exact"},
                                },
                                Priority: int32(0),
                        },
                },
                p4client.TableEntry{
                        Tablename: L2_FWD_LOOP,
                        TableField: p4client.TableField{
                                FieldValue: map[string][2]interface{}{
                                        "da":{vrf_mux_da,"exact"},
                                },
                                Priority: int32(0),
                        },
                },
                //NH entry for flooding
                p4client.TableEntry{
                        Tablename: PUSH_QNQ_FLOOD,
                        TableField: p4client.TableField{
                                FieldValue: map[string][2]interface{}{
                                        "mod_blob_ptr":{p.FLOOD_MOD_PTR,"exact"},
                                },
                                Priority: int32(0),
                        },
                },
                p4client.TableEntry{
                        Tablename: L2_NH,
                        TableField: p4client.TableField{
                                FieldValue: map[string][2]interface{}{
                                        "neighbor":{p.FLOOD_NH_ID,"exact"},
                                        "bit32_zeros": {uint32(0),"exact"},
                                },
                                Priority: int32(0),
                        },
                })
                return entries
}
