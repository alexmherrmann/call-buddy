package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/terminal"
	//"../telephono"
)

type remoteSyncClient struct {
	sshSession *ssh.Session
	sftpClient *sftp.Client
}

type remoteFileState struct {
	localPath  string
	remotePath string
	// Map between files we've seen and the last time they were modified
	modTimes map[string]time.Time
}

var quit = make(chan struct{})
var waitGroup = sync.WaitGroup{}

func parseArgs() (target, arch *string, args []string) {
	arch = flag.String("a", "", "Architecture to use: e.g. linux-amd64, darwin-amd64, linux-arm")
	target = flag.String("t", "", "Target host: user@host")
	flag.Parse()

	if *target != "" && !strings.Contains(*target, "@") {
		user, err := user.Current()
		if err != nil {
			log.Fatal("Failed to get username")
		}
		username := user.Username
		*target = username + "@" + *target
	}
	return target, arch, flag.Args()
}

func getSSHAgent() ssh.AuthMethod {
	if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
	}
	return nil
}

func getClient(username, hostname string) *ssh.Client {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			getSSHAgent(),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", hostname+":22", config)
	if err != nil {
		log.Fatal("Failed to dial: ", err)
	}
	return client
}

func remoteMetadata(client *ssh.Client) (string, string) {
	// We need the architecture to copy up the right call-buddy binary
	archSession, err := client.NewSession()
	if err != nil {
		log.Fatal("Failed to create meta session: ", err)
	}
	defer archSession.Close()
	var out bytes.Buffer
	archSession.Stdout = &out

	archSession.Run("uname -s -m | tr ' ' '-' | tr '[:upper:]' '[:lower:]'")
	// arch should look like "linux-x86_64"
	arch := strings.TrimSpace(out.String())

	// We can't assume ~/ is /home/<username>/ so we have to get the abs path here
	out.Reset()
	homedirSession, err := client.NewSession()
	if err != nil {
		log.Fatal("Failed to create meta session: ", err)
	}
	defer homedirSession.Close()
	homedirSession.Stdout = &out
	homedirSession.Run("eval echo ~$USER")
	homedir := strings.TrimSpace(out.String())
	return arch, homedir
}

func getPty(client *ssh.Client) (*ssh.Session, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Request pseudo terminal
	// FIXME DG: Make adjustments to current terminal propogate to remote pty
	// (also handle signals)
	width, height, err := terminal.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		log.Printf("Failed to get terminal size! %s\n", err)
		return nil, err
	}
	if err := session.RequestPty("xterm", height, width, modes); err != nil {
		log.Printf("Failed to get pseudo terminal: %s\n", err)
		return nil, err
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		log.Printf("Unable to setup stdin for session: %s\n", err)
		return nil, err
	}
	go io.Copy(stdin, os.Stdin)

	stdout, err := session.StdoutPipe()
	if err != nil {
		log.Fatalf("Unable to setup stdout for session: %v", err)
	}
	go io.Copy(os.Stdout, stdout)

	stderr, err := session.StderrPipe()
	if err != nil {
		log.Fatalf("Unable to setup stderr for session: %v", err)
	}
	go io.Copy(os.Stderr, stderr)

	return session, nil
}

func cleanupBootstrapCallBuddy(remoteSftpClient *sftp.Client, remoteCallBuddyPath string) {
	remoteSftpClient.Remove(remoteCallBuddyPath)
	remoteSftpClient.RemoveDirectory(filepath.Dir(remoteCallBuddyPath))
}

func bootstrapCallBuddy(remoteSftpClient *sftp.Client, localCallBuddyPath, remoteCallBuddyPath string) error {
	remoteCallBuddyDir := filepath.Dir(remoteCallBuddyPath)

	// Check whether the call-buddy dir exists
	_, err := remoteSftpClient.Stat(remoteCallBuddyDir)
	if err != nil {
		if os.IsNotExist(err) {
			err = remoteSftpClient.Mkdir(remoteCallBuddyDir)
			if err != nil {
				log.Printf("Failed to create remote call buddy dir '%s': %s\n", remoteCallBuddyDir, err)
				return err
			}
		} else {
			log.Printf("Failed to read remote call-buddy dir '%s': %s\n", remoteCallBuddyDir, err)
			return err
		}
	} else {
		// We don't know if the existing binary is old, so we'll remove it and
		// replace it with a new one
		err = remoteSftpClient.Remove(remoteCallBuddyPath)
		if err != nil {
			// Catch the case where ~/.call-buddy exists, but ~/.call-buddy/call-buddy does not
			if !os.IsNotExist(err) {
				log.Printf("Failed to remove existing remote call-buddy path '%s': %s\n", remoteCallBuddyPath, err)
				return err
			}
		}
	}
	remoteExe, err := remoteSftpClient.Create(remoteCallBuddyPath)
	if err != nil {
		log.Println("Failed to create remote call-buddy exe: ", err)
		return err
	}
	defer remoteExe.Close()
	remoteExe.Seek(0, os.SEEK_SET) // Make sure we're pointing to the beginning

	buf, err := ioutil.ReadFile(localCallBuddyPath)
	if err != nil {
		log.Println("Failed to read local call-buddy path: ", err)
		return err
	}
	remoteExe.Write(buf)
	remoteExe.Chmod(0755) // Make the file executable
	return nil
}

// This is a best effort syncing function, so if we can't do things we're not
// going to die :)
func remoteSync(remoteSftpClient *sftp.Client, remoteState *remoteFileState) error {
	files, err := remoteSftpClient.ReadDir(remoteState.remotePath)
	if err != nil {
		return err
	}

	var seenFilenames = make([]string, 1)
	for _, file := range files {
		filename := file.Name()

		// Clever hack to ensure that if a file changes to a directory, we
		// don't try and copy a directory rather than a file
		if file.IsDir() {
			filename += "/"
		}
		seenFilenames = append(seenFilenames, filename)

		if filename == "call-buddy" {
			continue
		}

		lastModTime, inMap := remoteState.modTimes[file.Name()]
		modTime := file.ModTime()
		if inMap && !modTime.After(lastModTime) {
			continue
		}

		remoteFilepath := remoteState.remotePath + "/" + filename
		remoteFd, err := remoteSftpClient.Open(remoteFilepath)
		if err != nil {
			log.Printf("Failed to remotely open %s for syncing\n", remoteFilepath)
			continue
		}

		localFilepath := remoteState.localPath + "/" + filename
		// FIXME DG: This is _really_ dangerous... We'll truncate existing files and if there are issues we'll end up destroying things!
		localFd, err := os.Create(localFilepath)
		if err != nil {
			log.Printf("Failed to locally open %s for syncing\n", localFilepath)
			continue
		}
		_, err = remoteFd.WriteTo(localFd)
		if err != nil {
			log.Fatalf("Failed to locally write %s for syncing\n", localFilepath)
			continue
		}

		remoteState.modTimes[filename] = modTime
		log.Printf("Sucessfully synced remote %s to local %s after it was modifed at %s\n", remoteFilepath, localFilepath, modTime)
	}
	return nil
}

func cleanupRemoteSyncing(remoteClient *remoteSyncClient) {
	// TODO: Cleanup syncing goroutine
	remoteClient.sshSession.Close()
	remoteClient.sftpClient.Close()
}

func spawnRemoteSyncing(client *ssh.Client, remoteCallBuddyDir string) (*remoteSyncClient, error) {
	// FIXME: Log things out
	syncingSession, err := client.NewSession()
	if err != nil {
		log.Println("Failed to create syncing session: ", err)
		return nil, err
	}
	syncingStdin, err := syncingSession.StdinPipe()
	if err != nil {
		syncingSession.Close()
		return nil, err
	}
	syncingStdout, err := syncingSession.StdoutPipe()
	if err != nil {
		syncingSession.Close()
		return nil, err
	}

	syncingSession.RequestSubsystem("sftp")
	syncingClient, err := sftp.NewClientPipe(syncingStdout, syncingStdin)
	if err != nil {
		syncingSession.Close()
		return nil, err
	}
	remoteClient := remoteSyncClient{syncingSession, syncingClient}

	cwd, _ := os.Getwd()
	remoteState := remoteFileState{
		localPath:  cwd,
		remotePath: remoteCallBuddyDir,
		modTimes:   make(map[string]time.Time),
	}

	syncPeriod := time.Duration(3)
	ticker := time.NewTicker(syncPeriod * time.Second)
	go func() {
		for {
			waitGroup.Add(1)
			select {
			case <-ticker.C:
				err := remoteSync(syncingClient, &remoteState)

				// We get a EOF when sftp closes
				// FIXME DG: Catch an EOF rather than a general error so we can
				//           be more tolerent
				// FIXME DG: We're getting an EOF since we are closing the ssh
				//		     session (which includes the sftp subsystem)
				//           outside of this goroutine, so ideally we have some
				//           syncronization primitive here to check whether the
				//           ssh session has closed and exit. Or maybe there's
				//           a way to externally shutdown a goroutine?
				if err != nil {
					ticker.Stop()
					log.Printf("Failed to sync files: %s\n", err)
				}
				break
			case <-quit:
				log.Println("Quitting from the goroutine!")
				ticker.Stop()
				waitGroup.Done()
				return
			}
			waitGroup.Done()
		}
	}()
	return &remoteClient, nil
}

func remoteRun(target, arch *string, args []string) {
	s := strings.Split(*target, "@")
	username, hostname := s[0], s[1]
	client := getClient(username, hostname)

	detectedArch, remoteHomeDir := remoteMetadata(client)
	if *arch != "" {
		if *arch != detectedArch {
			log.Printf("Given architecture does not match detected architecture: %s vs %s\n (btw -a can be ommited)", *arch, detectedArch)
		}
	} else {
		if detectedArch == "" {
			log.Fatalf("Failed to get architecture for %s! Try using the -a flag.", hostname)
		}
		arch = &detectedArch
	}

	remoteCallBuddyPath := remoteHomeDir + "/.call-buddy/call-buddy"
	remoteCallBuddyDir := filepath.Dir(remoteCallBuddyPath)
	log.Printf("Spawning off remote syncing client on %s\n", hostname)
	syncingClient, err := spawnRemoteSyncing(client, remoteCallBuddyDir)
	if err != nil {
		// TODO: Inspect the error and don't always fail
		log.Fatal("Failed to spawn off remote syncing")
	}

	cwd, _ := os.Getwd()
	localCallBuddyPath := filepath.Clean(cwd + "/../telephono-ui/build/" + *arch + "/call-buddy")
	log.Printf("Syncing call-buddy from %s to remote client at %s@%s:%s\n", localCallBuddyPath, username, hostname, remoteCallBuddyPath)
	err = bootstrapCallBuddy(syncingClient.sftpClient, localCallBuddyPath, remoteCallBuddyPath)
	bootstrapped := err == nil

	session, err := getPty(client)
	if err != nil {
		log.Fatalf("Failed to get pty on %s@%s\n")
	}

	session.Run(remoteCallBuddyPath + " " + strings.Join(args, " "))
	// FIXME: When we do ctl-c to quit call-buddy we don't get here!
	fmt.Printf("Removing call-buddy from remote client at %s@%s:%s\n", username, hostname, remoteCallBuddyPath)
	if bootstrapped {
		cleanupBootstrapCallBuddy(syncingClient.sftpClient, remoteCallBuddyPath)
	}
	log.Println("Got here!")
	quit <- struct{}{}
	waitGroup.Wait()
	session.Close()
	cleanupRemoteSyncing(syncingClient)
}

func localRun(args []string) {
	callBuddyPath, err := exec.LookPath("call-buddy")
	if err != nil {
		log.Fatal("Could not find call-buddy in $PATH!")
	}
	// We don't have a full argv here since we are missing arg 0: the executable name
	argv := append([]string{callBuddyPath}, args...)
	exe := &exec.Cmd{
		Path:   callBuddyPath,
		Args:   argv,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Stdin:  os.Stdin,
	}
	if err := exe.Run(); err != nil {
		log.Fatal("Error: ", err)
	}
}

func main() {
	target, arch, args := parseArgs()
	if *target != "" {
		remoteRun(target, arch, args)
	} else {
		localRun(args)
	}
}
