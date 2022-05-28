package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Telmate/proxmox-api-go/proxmox"
)

const usageString = `
Usage:
  %s [options] <command> [<args>...]

  Commands:
	help [command]     Show help for a command
	list [object type] List objects of a given type (e.g. cluster, node, storage, vm, ...)
	login              Login to Proxmox server and display credentials (not necessary for most commands)
`

func usage() {
	fmt.Printf(usageString, os.Args[0])
}

func help(command string) {
	switch command {
	case "list":
		fmt.Printf(`
List objects of a given type (e.g. cluster, node, storage, vm, ...)
examples:
	%s list cluster
	%s list node
	%s list storage
	%s list vm
`[1:], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
	default:
		fmt.Printf("Unknown command: %s\n", command)
	}
}

func main() {
	username := flag.String("username", "root", "Username")
	password := flag.String("password", "bad-password", "Password")
	otp := flag.String("otp", "", "OTP Code")
	serverURL := flag.String("server", "https://localhost:8006/api2/json", "Proxmox server URL")
	skipTLSVerify := flag.Bool("skiptls", false, "Skip TLS verification. Avoid this whenver possible.")
	debug := flag.Bool("debug", false, "Debug mode")
	realm := flag.String("realm", "pam", "Authentication realm") // See https://pve.proxmox.com/wiki/User_Management#pveum_authentication_realms

	flag.Parse()

	command := flag.Arg(0)

	if command == "" {
		usage()
		os.Exit(1)
	}

	if *debug {
		log.Println(
			"Connecting to",
			serverURL,
			"with username",
			*username,
			"and password",
			*password,
			"while skipping tls?",
			fmt.Sprintf("%t", *skipTLSVerify),
		)
	}

	httpClient := http.DefaultClient
	if *skipTLSVerify {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	client, err := proxmox.NewClient(*serverURL, httpClient, nil, 10)
	if err != nil {
		log.Fatalln("Failed to create client", err.Error())
	}

	usernameWithRealm := *username
	if !strings.Contains(usernameWithRealm, "@") {
		usernameWithRealm = usernameWithRealm + "@" + *realm
	}

	err = client.Login(usernameWithRealm, *password, *otp)
	if err != nil {
		log.Fatalln("Failed to login", err.Error())
	}

	switch command {
	case "h", "help":
		if flag.NArg() == 2 {
			// Help for a specific command
			help(command)
			os.Exit(0)
		} else {
			usage()
			os.Exit(0)
		}
	case "l", "ls", "list":
		switch flag.Arg(1) {
		case "c", "cluster", "clusters":
			listClusters(client)
		case "n", "node", "nodes":
			listNodes(client)
		case "s", "storage":
			listStorages(client)
		case "v", "vm", "vms":
			listVMs(client)
		default:
			help(command)
			os.Exit(0)
		}
	}

}

func listClusters(client *proxmox.Client) {
	log.Fatalln("Cluster operations are not yet supported :(")
	// TODO: Make API available
	// clusters, err := client.GetClusterList()
	// if err != nil {
	// 	log.Fatalln("Failed to list clusters", err.Error())
	// }
	// for _, cluster := range clusters {
	// 	fmt.Println(cluster)
	// }
}

func listNodes(client *proxmox.Client) {
	nodes, err := client.GetNodeList()
	if err != nil {
		log.Fatalln("Failed to list nodes", err.Error())
	}
	for _, nodeInfo := range nodes {
		for _, node := range nodeInfo.([]interface{}) {
			name, _ := node.(map[string]interface{})["node"].(string)
			fmt.Println(name)
			for nodeAttr, nodeAttrValue := range node.(map[string]interface{}) {
				if nodeAttr != "node" {
					fmt.Println("\t", nodeAttr, ":", nodeAttrValue)
				}
			}
		}
	}
}

func listStorages(client *proxmox.Client) {
	nodes, err := client.GetNodeList()
	if err != nil {
		log.Fatalln("Failed to list nodes", err.Error())
	}
	for _, nodeInfo := range nodes {
		for _, node := range nodeInfo.([]interface{}) {
			nodeName := node.(map[string]interface{})["node"].(string)
			fmt.Println(nodeName)
			storages, err := client.ListStorages(nodeName)
			if err != nil {
				log.Fatalf("Failed to fetch storages for node %s: %s\n", nodeName, err.Error())
			}

			for _, storage := range storages {
				storageName := storage.(map[string]interface{})["storage"].(string)
				fmt.Println("\t", storageName)
				for attrName, attrVal := range storage.(map[string]interface{}) {
					if attrName != "storage" {
						fmt.Printf("\t\t%s:%+v\n", attrName, attrVal)
					}
				}
			}
		}
	}

	// TODO: Make API available
	// clusters, err := client.GetClusterList()
	// if err != nil {
	// 	log.Fatalln("Failed to list clusters", err.Error())
	// }
	// for _, cluster := range clusters {
	// 	fmt.Println(cluster)
	// }
}

func listVMs(client *proxmox.Client) {
	vms, err := client.GetVmList()
	if err != nil {
		log.Fatalln("Failed to list VMs", err.Error())
	}
	for _, vmInfo := range vms {
		for _, vm := range vmInfo.([]interface{}) {
			name, _ := vm.(map[string]interface{})["name"].(string)
			fmt.Println(name)
			fmt.Println(" Status:")
			for vmAttr, vmAttrVal := range vm.(map[string]interface{}) {
				if vmAttr != "name" {
					fmt.Printf("\t%s: %+v\n", vmAttr, vmAttrVal)
				}
			}

			vmRef, err := client.GetVmRefByName(name)
			if err != nil {
				log.Fatalln("Failed to get VM reference", err.Error())
			}
			vmConfig, err := client.GetVmConfig(vmRef)
			if err != nil {
				log.Fatalln("Failed to get VM config", err.Error())
			}
			fmt.Println(" Config:")
			for vmConfigAttr, vmConfigAttrValue := range vmConfig {
				fmt.Printf("\t%s: %+v\n", vmConfigAttr, vmConfigAttrValue)
			}

			fmt.Println(" Agent network interfaces:")
			agentNetworkInterfaces, err := client.GetVmAgentNetworkInterfaces(vmRef)
			if err != nil {
				fmt.Println("\tNot available:", err.Error())
			} else {
				for _, agentNetworkInterface := range agentNetworkInterfaces {
					fmt.Println("\t", agentNetworkInterface)
				}
			}
		}
	}
}
