package rdp

import (
	"log"
)

func OpenRemoteDesktopClient(rdpPort int) {
	log.Printf("open your prefered rdp client and connect to the server on localhost:%v as the Administrator user", rdpPort)
}
