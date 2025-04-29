package honeypot

import (
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
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
	log.Debug(
		"Attempting to set up reverse tunnel.",
		"Remote Host:", fmt.Sprintf("%s@%s:%s", tunnelUser, tunnelHost, tunnelSSHPort),
		"SSH Key:", sshKeyPath,
		"Known Hosts:", knownHostsPath,
		"Forwarding Remote Port:", fmt.Sprintf("%s:%s", remoteBindAddr, remoteForwardPort),
		"Local Service:", localServiceAddr,
	)

	// Get SSH Auth Method
	authMethod, err := publicKeyFile(sshKeyPath)
	if err != nil {
		log.Error("Failed to load private key: %v", err)
		return
	}

	// Host Key Verification (Recommended)
	hostKeyCallback := ssh.InsecureIgnoreHostKey()
	if knownHostsPath != "" {
		hostKeyCallback, err = knownhosts.New(knownHostsPath)
		if err != nil {
			// If known_hosts doesn't exist or is invalid, fallback is less secure.
			// Consider if you want to *require* known_hosts.
			log.Warn("Could not load known_hosts file. Using InsecureIgnoreHostKey.", "file", knownHostsPath, "error", err)
		}
	} else {
		log.Warn("No known host file set. Using InsecureIgnoreHostKey.")

	}

	// Configure SSH Client
	sshConfig := &ssh.ClientConfig{
		User: tunnelUser,
		Auth: []ssh.AuthMethod{
			authMethod,
		},
		HostKeyCallback: hostKeyCallback,
		Timeout:         15 * time.Second, // Connection timeout
	}

	// Connect to remote SSH server
	remoteServerAddr := fmt.Sprintf("%s:%s", tunnelHost, tunnelSSHPort)
	log.Debug("Connecting to remote SSH server...", "host", remoteServerAddr)

	// Loop for potential reconnection (optional, basic implementation)
	retryWaitFactor := 1
	for {
		sshClient, err := ssh.Dial("tcp", remoteServerAddr, sshConfig)
		if err != nil {
			log.Warn("Failed to dial remote SSH server. Retrying in 10 seconds...", "error", err)
			time.Sleep(10 * time.Second)
			continue // Retry connection
		}
		log.Debug("Successfully connected to remote SSH server.")

		// Request the remote side to listen and forward
		remoteListenAddr := fmt.Sprintf("%s:%s", remoteBindAddr, remoteForwardPort)
		log.Debug("Requesting remote server to listen on...", "host", remoteListenAddr)
		listener, err := sshClient.Listen("tcp", remoteListenAddr)
		if err != nil {
			log.Warn("Failed to request remote listener: %v. (Check remote sshd_config AllowTcpForwarding). Retrying connection in %d seconds...", err, retryWaitFactor*10)
			sshClient.Close() // Close the potentially broken client
			tunnelActive = 0  // Tunnel is not active

			time.Sleep(time.Duration(10*retryWaitFactor) * time.Second)
			retryWaitFactor++ // Increase wait time for next retry
			continue          // Retry connection
		}

		log.Info("Remote server is now active!", "listening on", remoteListenAddr, "forwarding to", localServiceAddr)
		tunnelActive = 1    // Tunnel is active
		retryWaitFactor = 1 // Reset retry wait factor

		// --- Accept loop: Handle incoming connections from the tunnel ---
		acceptLoop(listener, localServiceAddr)

		// If acceptLoop returns, it means the listener failed (likely SSH connection dropped)
		log.Warn("Tunnel listener closed. Attempting to reconnect...")
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
			log.Warn("Failed to accept incoming tunnel connection.", "error", err)
			// Exit the loop so the outer function can attempt reconnection.
			return
		}
		log.Info("Accepted tunneled connection.", "from", remoteConn.RemoteAddr())

		// Handle the connection in a new goroutine
		go func(tunneledConn net.Conn) {
			defer tunneledConn.Close()

			// Dial the local web service
			localConn, err := net.DialTimeout("tcp", localServiceAddr, 10*time.Second) // Added timeout
			if err != nil {
				log.Warn("Failed to dial local web service.", "host", localServiceAddr, "error", err)
				return // Close the tunneled connection
			}
			defer localConn.Close()
			log.Debug("Successfully connected to local web service or tunneled connection", "addr", localServiceAddr)

			// Proxy data
			log.Debug("Proxying data between tunneled <--> local", "tunneled", tunneledConn.RemoteAddr(), "local", localConn.RemoteAddr())
			errChan := make(chan error, 2)
			go func() { _, err := io.Copy(localConn, tunneledConn); errChan <- err }()
			go func() { _, err := io.Copy(tunneledConn, localConn); errChan <- err }()

			// Wait for one side to finish/error
			err = <-errChan
			if err != nil && err != io.EOF {
				log.Error("Proxy copy error.", "error", err)
			}
			log.Info("Finished proxying for tunneled connection.", "tunnel", tunneledConn.RemoteAddr())

		}(remoteConn)
	}
}
