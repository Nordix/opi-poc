package main

import(
	p4init "vendor/intel/p4runtime/pkg/p4translation"
)

func main(){
	// Initialise the vendor specific component
	// offloads static entries to the Target
	// Listens for netlink & Infra DB events and offloads to the Target
	p4init.Init()
	// Clean Up
	p4init.Exit()
}
