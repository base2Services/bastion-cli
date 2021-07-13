package rdp

import (
	"log"
	"os/exec"
)

func OpenRemoteDesktopClient(rdpPort int) {
	err := WaitForRDPPort(rdpPort)
	if err != nil {
		log.Printf("Failed to wait for rdp port %d, %s", rdpPort, err)
		return
	}

	rdpUrl := CreateRDPURL(rdpPort)

	cmd := exec.Command("open", rdpUrl)
	err = cmd.Run()
	if err != nil {
		log.Printf("Failed to run the remote desktop client, %s", err)
	}
}
