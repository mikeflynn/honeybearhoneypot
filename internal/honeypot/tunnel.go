package honeypot

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

// Helper function to load the private key
func publicKeyFile(file string) (ssh.AuthMethod, error) {
	buffer, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("cannot read SSH private key file %s: %w", file, err)
	}
	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil, fmt.Errorf("cannot parse SSH private key file %s: %w", file, err)
	}
	return ssh.PublicKeys(key), nil
}

// setupReverseTunnel establishes the reverse tunnel and keeps it alive.
func setupReverseTunnel(
	tunnelUser, tunnelHost string,
	tunnelSSHPort string,
	sshKeyPath, knownHostsPath string,
	localServiceAddr string, // Address of the local web server (e.g., "localhost:8080")
	remoteBindAddr string, // Address to bind on remote server (e.g., "0.0.0.0")
	remoteForwardPort string, // Port to forward on remote server (e.g., 8022)
) {
	log.Printf("Attempting to set up reverse tunnel:")
	log.Printf("  Remote Host: %s@%s:%s", tunnelUser, tunnelHost, tunnelSSHPort)
	log.Printf("  SSH Key: %s", sshKeyPath)
	log.Printf("  Known Hosts: %s", knownHostsPath)
	log.Printf("  Forwarding Remote Port: %s:%s", remoteBindAddr, remoteForwardPort)
	log.Printf("  To Local Web Service: %s", localServiceAddr)

	// Get SSH Auth Method
	authMethod, err := publicKeyFile(sshKeyPath)
	if err != nil {
		log.Fatalf("Failed to load private key: %v", err)
	}

	// Host Key Verification (Recommended)
	/*
		hostKeyCallback, err := knownhosts.New(knownHostsPath)
		if err != nil {
			// If known_hosts doesn't exist or is invalid, fallback is less secure.
			// Consider if you want to *require* known_hosts.
			log.Printf("WARN: Could not load known_hosts file '%s': %v. Using InsecureIgnoreHostKey.", knownHostsPath, err)
			// **SECURITY RISK**: Only use InsecureIgnoreHostKey if you understand the implications (MitM vulnerability)
			//hostKeyCallback = ssh.InsecureIgnoreHostKey()
			log.Fatalf("Known hosts file is required for security. Please create or specify a valid file.") // Safer default
		}
	*/

	// Configure SSH Client
	sshConfig := &ssh.ClientConfig{
		User: tunnelUser,
		Auth: []ssh.AuthMethod{
			authMethod,
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second, // Connection timeout
	}

	// Connect to remote SSH server
	remoteServerAddr := fmt.Sprintf("%s:%s", tunnelHost, tunnelSSHPort)
	log.Printf("Connecting to remote SSH server %s...", remoteServerAddr)

	// Loop for potential reconnection (optional, basic implementation)
	for {
		sshClient, err := ssh.Dial("tcp", remoteServerAddr, sshConfig)
		if err != nil {
			log.Printf("Failed to dial remote SSH server: %v. Retrying in 10 seconds...", err)
			time.Sleep(10 * time.Second)
			continue // Retry connection
		}
		log.Println("Successfully connected to remote SSH server.")

		// Request the remote side to listen and forward
		remoteListenAddr := fmt.Sprintf("%s:%s", remoteBindAddr, remoteForwardPort)
		log.Printf("Requesting remote server to listen on %s...", remoteListenAddr)
		listener, err := sshClient.Listen("tcp", remoteListenAddr)
		if err != nil {
			log.Printf("Failed to request remote listener: %v. (Check remote sshd_config AllowTcpForwarding). Retrying connection in 10 seconds...", err)
			sshClient.Close() // Close the potentially broken client
			time.Sleep(10 * time.Second)
			continue // Retry connection
		}
		log.Printf("Remote server is now listening on %s and forwarding to %s.", remoteListenAddr, localServiceAddr)

		// --- Accept loop: Handle incoming connections from the tunnel ---
		acceptLoop(listener, localServiceAddr)

		// If acceptLoop returns, it means the listener failed (likely SSH connection dropped)
		log.Println("Tunnel listener closed. Attempting to reconnect...")
		sshClient.Close()           // Ensure client is closed before retrying
		time.Sleep(5 * time.Second) // Wait a bit before reconnecting
	}
}

// acceptLoop handles incoming connections for a listener
func acceptLoop(listener net.Listener, localServiceAddr string) {
	defer listener.Close() // Ensure listener is closed when this function exits
	for {
		remoteConn, err := listener.Accept()
		if err != nil {
			// This error often happens if the SSH connection drops or listener is closed.
			log.Printf("Failed to accept incoming tunnel connection: %v.", err)
			// Exit the loop so the outer function can attempt reconnection.
			return
		}
		log.Printf("Accepted tunneled connection from %s", remoteConn.RemoteAddr())

		// Handle the connection in a new goroutine
		go func(tunneledConn net.Conn) {
			defer tunneledConn.Close()

			// Dial the local web service
			localConn, err := net.DialTimeout("tcp", localServiceAddr, 10*time.Second) // Added timeout
			if err != nil {
				log.Printf("Failed to dial local web service %s: %v", localServiceAddr, err)
				return // Close the tunneled connection
			}
			defer localConn.Close()
			log.Printf("Successfully connected to local web service %s for tunneled connection", localServiceAddr)

			// Proxy data
			log.Printf("Proxying data between tunneled %s <--> local %s", tunneledConn.RemoteAddr(), localConn.RemoteAddr())
			errChan := make(chan error, 2)
			go func() { _, err := io.Copy(localConn, tunneledConn); errChan <- err }()
			go func() { _, err := io.Copy(tunneledConn, localConn); errChan <- err }()

			// Wait for one side to finish/error
			err = <-errChan
			if err != nil && err != io.EOF {
				log.Printf("Proxy copy error: %v", err)
			}
			log.Printf("Finished proxying for tunneled connection from %s", tunneledConn.RemoteAddr())

		}(remoteConn)
	}
}
