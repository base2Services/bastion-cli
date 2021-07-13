package rdp

import (
	"fmt"
	"log"
	"os/exec"
)

func OpenRemoteDesktopClient(rdpPort int) {
	err := WaitForRDPPort(rdpPort)
	if err != nil {
		log.Printf("Failed to wait for rdp port %d, %s", rdpPort, err)
		return
	}

	cmd := exec.Command("mstsc", fmt.Sprintf("/v:localhost:%v", rdpPort), "/admin")
	err = cmd.Run()
	if err != nil {
		log.Printf("Failed to run the remote desktop client, %s", err)
	}
}
