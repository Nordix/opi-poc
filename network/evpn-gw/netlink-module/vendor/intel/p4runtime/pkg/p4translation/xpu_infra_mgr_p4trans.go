package p4translation
import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
	"regexp"
	"encoding/json"
	"os/exec"
	"log"
	"strconv"
	"google.golang.org/grpc"
	p4client "vendor/intel/p4runtime/pkg/p4driverAPI"
)

var L3 L3Decoder
//var Vxlan VxlanDecoder
var Pod PodDecoder

func isValidMAC(mac string) bool {
	macPattern := `^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`

	match, _ := regexp.MatchString(macPattern, mac)
	return match
}
func getMac(dev string) string {
	cmd := exec.Command("ip", "-d", "-j", "link", "show", dev)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error running command: %v", err)
		return ""
	}

	var links []struct {
		Address string `json:"address"`
	}
	if err := json.Unmarshal(out, &links); err != nil {
		log.Printf("Error unmarshaling JSON: %v", err)
		return ""
	}

	if len(links) > 0 {
		mac := links[0].Address
		return mac
	}

	return ""
}

func vport_from_mac(mac string) int {
	mbyte := strings.Split(mac, ":")
	if len(mbyte) < 5 {
		return -1
	}
	byte0, _ := strconv.ParseInt(mbyte[0], 16, 64)
	byte1, _ := strconv.ParseInt(mbyte[1], 16, 64)

	return int(byte0<<8 + byte1)
}

func ids_of(value string)(string, string, error){
	if isValidMAC(value) {
		return strconv.Itoa(vport_from_mac(value)),value, nil
	}
	mac := getMac(value)
	vsi := vport_from_mac(mac)
	return strconv.Itoa(vsi), mac, nil
}

var (
        defaultAddr = fmt.Sprintf("127.0.0.1:9559")
	Conn *grpc.ClientConn
)

func Init(){
	Conn, err := grpc.Dial(defaultAddr, grpc.WithInsecure())
        if err != nil {
                log.Fatalf("Cannot connect to server: %v", err)
        }
	configFile, err := ioutil.ReadFile("../../config.yaml")
	if err != nil {
		fmt.Println("Error reading config file:", err)
		return
	}
	var configMap map[string]interface{}
	err = yaml.Unmarshal(configFile, &configMap)
	if err != nil {
		fmt.Println("Error parsing config:", err)
		return
	}
	p4 := configMap["p4"].(map[interface{}]interface{})
	p4config := p4["config"].(map[interface{}]interface{})
	infoFile, ok := p4config["p4info_file"].(string)
	if !ok {
		log.Fatal("Error accessing info_file")
	}
	binFile, ok := p4config["bin_file"].(string)
	if !ok {
		log.Fatal("Error accessing bin_file")
	}

	err1 := p4client.NewP4RuntimeClient(binFile, infoFile, Conn)
        if err1 != nil {
                log.Fatalf("Failed to create P4Runtime client: %v", err1)
        }
	representors := make(map[string][2]string)
	for k, v := range p4["representors"].(map[interface{}]interface{}) {
		vsi, mac, err := ids_of(v.(string))
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		representors[k.(string)] = [2]string{vsi, mac}
	}
	L3 = L3.L3DecoderInit(representors)
	Pod = Pod.PodDecoderInit(representors)
	L3entries := L3.Static_additions()
	for _, entry := range L3entries {
		if e, ok := entry.(p4client.TableEntry); ok {
			p4client.Add_entry(e)
		} else {
			fmt.Println("Entry is not of type p4client.TableEntry")
		}
	}
	Podentries := Pod.Static_additions()
	for _, entry := range Podentries {
                if e, ok := entry.(p4client.TableEntry); ok {
                        p4client.Add_entry(e)
                } else {
                        fmt.Println("Entry is not of type p4client.TableEntry")
                }
        }
}

func Exit(){
	L3entries := L3.Static_deletions()
        for _, entry := range L3entries {
                if e, ok := entry.(p4client.TableEntry); ok {
                        p4client.Del_entry(e)
                } else {
                        fmt.Println("Entry is not of type p4client.TableEntry")
                }
        }
	Podentries := Pod.Static_deletions()
        for _, entry := range Podentries {
                if e, ok := entry.(p4client.TableEntry); ok {
                        p4client.Del_entry(e)
                } else {
                        fmt.Println("Entry is not of type p4client.TableEntry")
                }
        }
}
