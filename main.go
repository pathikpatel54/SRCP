package main

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/buger/goterm"
)

func TestChatApp(t *testing.T) {
	// Step 1: Open 3 terminal windows and change directory to SRCP folder
	terminals := openTerminals(3)
	defer closeTerminals(terminals)

	// Step 2: Install standard library package 'golang.org/x/term'
	installTermPackage(terminals[0])

	// Step 3: Start server in the first terminal window
	startServer(terminals[0])

	// Step 4: Start clients in the second and third terminal windows
	startClient(terminals[1])
	startClient(terminals[2])

	// Step 5: Enter server address in both client windows
	enterServerAddress(terminals[1])
	enterServerAddress(terminals[2])

	// Step 6: Enter username and random password in both client windows
	enterCredentials(terminals[1], "alex", "password123")
	enterCredentials(terminals[2], "bob", "password456")

	// Step 7: Display online participants list
	displayParticipants(terminals[1])
	displayParticipants(terminals[2])

	// Step 8: Select participant in both windows
	selectParticipant(terminals[1], 1)
	selectParticipant(terminals[2], 1)

	// Step 9: Send messages between clients
	sendMessage(terminals[1], terminals[2], "Hello from Alex!")
	sendMessage(terminals[2], terminals[1], "Hi, this is Bob!")

	// Add additional assertions or checks here if needed
}

func openTerminals(num int) []*exec.Cmd {
	terminals := make([]*exec.Cmd, num)

	// Open new terminal windows using goterm
	for i := 0; i < num; i++ {
		terminals[i] = exec.Command(goterm.GetTermCmd(), "-e", "bash", "-c", "cd SRCP && exec bash")
		terminals[i].Start()
		time.Sleep(2 * time.Second) // Wait for the terminal window to open
	}

	return terminals
}

func closeTerminals(terminals []*exec.Cmd) {
	for _, terminal := range terminals {
		terminal.Process.Kill()
	}
}

func installTermPackage(cmd *exec.Cmd) {
	cmd.Dir = "SRCP" // Change to the actual SRCP folder path
	cmd.Args = append(cmd.Args, "go", "get", "golang.org/x/term")

	err := cmd.Run()
	if err != nil {
		fmt.Println("Error installing term package:", err)
	}
}

func startServer(cmd *exec.Cmd) {
	cmd.Dir = "SRCP/server" // Change to the actual server folder path
	cmd.Args = append(cmd.Args, "go", "run", "main.go")

	err := cmd.Start()
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
	time.Sleep(2 * time.Second) // Wait for the server to start
}

func startClient(cmd *exec.Cmd) {
	cmd.Dir = "SRCP/client" // Change to the actual client folder path
	cmd.Args = append(cmd.Args, "go", "run", "main.go")

	err := cmd.Start()
	if err != nil {
		fmt.Println("Error starting client:", err)
	}
	time.Sleep(2 * time.Second) // Wait for the client to start
}

func enterServerAddress(cmd *exec.Cmd) {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("Enter server address (e.g., localhost or 192.168.1.100):")
	err := cmd.Run()
	if err != nil {
		fmt.Println("Error entering server address:", err)
	}
	time.Sleep(2 * time.Second) // Wait for the input to be entered
}

func enterCredentials(cmd *exec.Cmd, username, password string) {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("Enter username:", username)
	fmt.Println("Enter password:", password)

	err := cmd.Run()
	if err != nil {
		fmt.Println("Error entering credentials:", err)
	}
	time.Sleep(2 * time.Second) // Wait for the input to be entered
}

func displayParticipants(cmd *exec.Cmd) {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Args = append(cmd.Args, "list")

	err := cmd.Run()
	if err != nil {
		fmt.Println("Error displaying participants:", err)
	}
	time.Sleep(2 * time.Second) // Wait for the output to be displayed
}

func selectParticipant(cmd *exec.Cmd, participantIndex int) {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("Select participant:", participantIndex)

	err := cmd.Run()
	if err != nil {
		fmt.Println("Error selecting participant:", err)
	}
	time.Sleep(2 * time.Second) // Wait for the input to be entered
}

func sendMessage(senderCmd, receiverCmd *exec.Cmd, message string) {
	senderCmd.Stdin = os.Stdin
	senderCmd.Stdout = os.Stdout
	senderCmd.Stderr = os.Stderr

	fmt.Println("Sending message:", message)

	// Send the message in the senderCmd window
	err := senderCmd.Run()
	if err != nil {
		fmt.Println("Error sending message:", err)
	}
	time.Sleep(2 * time.Second) // Wait for the input to be entered

	// Receive the message in the receiverCmd window
	receiverCmd.Stdin = os.Stdin
	receiverCmd.Stdout = os.Stdout
	receiverCmd.Stderr = os.Stderr

	err = receiverCmd.Run()
	if err != nil {
		fmt.Println("Error receiving message:", err)
	}
	time.Sleep(2 * time.Second) // Wait for the output to be displayed
}

func main() {
	// Run the test
	os.Exit(func() int {
		tests := []testing.InternalTest{
			{Name: "TestChatApp", F: TestChatApp},
		}
		matchString := func(pat, str string) (bool, error) {
			return true, nil
		}
		testsRun := testing.RunTests(matchString, tests)
		if testsRun == 0 {
			fmt.Println("No tests found")
			return 1
		}
		if testsRun > 0 && testsRun < len(tests) {
			fmt.Println("Some tests failed")
			return 1
		}
		return 0
	}())
}
