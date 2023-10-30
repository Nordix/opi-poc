package main

import (
    "os"	
    "fmt"
)


func main() {
	dat, err := os.ReadFile("netlink_dump")
	 if err != nil {
                panic(err)
        }
	fmt.Printf("len %d\n",len(dat))
 	fmt.Print(string(dat))
	if err := os.Truncate("netlink_dump", 0); err != nil {
 	   fmt.Printf("Failed to truncate: %v", err)
        }
}
