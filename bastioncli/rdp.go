package bastioncli

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/avast/retry-go/v3"
)

func GetRandomRDPPort() int {
	return 57564 //rand.Intn(59999-50000) + 50000
}

func CreateRDPFile(port int, id string) (string, error) {
	contents := fmt.Sprintf(`auto connect:i:1
audiomode=i:2
disable themes=i:1
full address:s:localhost:%d
username:s:Administrator
prompt for credentials on client=i:1`,
		port)

	filename := fmt.Sprintf("bastion_%s", id)
	file, err := os.Create(filename)
	if err != nil {
		return "", err
	}

	defer file.Close()

	_, err = file.WriteString(contents)
	if err != nil {
		return "", err
	}

	return filename, nil
}

func CreateRDPURL(port int) string {
	return fmt.Sprintf("rdp:///auto%%20connect:i:1&full%%20address=s:localhost:%d&audiomode=i:2&disable%%20themes=i:1&username=s:Administrator&prompt%%20for%%20credentials%%20on%%20client=i:1", port)
}

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

func WaitForRDPPort(port int) error {
	err := retry.Do(
		func() error {
			return CheckTCPConnection(fmt.Sprint(port))
		},
		retry.Delay(time.Second),
	)

	if err != nil {
		return err
	}

	return nil
}

func CheckTCPConnection(port string) error {
	timeout := time.Second
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("", port), timeout)
	if err != nil {
		return err
	}
	if conn != nil {
		defer conn.Close()
		return nil
	}
	log.Println("not sure how we get here")
	return errors.New("this")
}
