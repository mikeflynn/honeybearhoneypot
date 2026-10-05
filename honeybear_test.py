import paramiko
import time

def test_honeybear():
    # start the SSH client
    ssh = paramiko.SSHClient()
    ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())

    # some commands to test; add more where needed
    commands = ['uname -a', 'whoami', 'ls', 'ls -la', 'bearsay hello', 'pwd', 'ctf']
            
    host = 'localhost'
    port = 2222

    
    print(f"Connecting to Honey Bear Honeypot on {host}:{port}....")
    
    try:
        ssh.connect(host, port=port, username='hacker', password='password123', timeout=5)
        print("Connected and authenticated!")

        # use a persistent shell
        channel = ssh.invoke_shell()
        time.sleep(0.5)
        
        if channel.recv_ready():
            print(channel.recv(4096).decode('utf-8'))
            
        for cmd in commands:
            print(f"\nRunning command: {cmd}")
            channel.send(cmd + '\n')
            time.sleep(0.5)
            
            if channel.recv_ready():
                print(channel.recv(4096).decode('utf-8'))
        
        
        
        # Fallback method if the persistent shell acts up beacuse I have no idea which version is going to work:
        # for cmd in commands:
        #     print(f"Executing: {cmd}")
        #     stdin, stdout, stderr = ssh.exec_command(cmd)
        #     time.sleep(0.5)
        #     print(stdout.read().decode('utf-8').strip())
            
        print("\nDone testing.")
        
    except Exception as e:
        print(f"Error running tests: {e}")
    finally:
        ssh.close()

if __name__ == "__main__":
    test_honeybear()
