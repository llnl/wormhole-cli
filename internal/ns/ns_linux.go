//go:build linux

package ns

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type linuxNamespace struct{}

func Initialize() (NamespaceExec, error) {
	data, err := os.ReadFile("/proc/sys/user/max_user_namespaces")
	if err != nil {
		return nil, err
	}
	s := strings.TrimSpace(string(data))
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil, fmt.Errorf("Failed sysctl user.max_user_namespaces: %w", err)
	} else {
		if v <= 0 {
			return nil, fmt.Errorf("user namespaces disabled: sysctl user.max_user_namespaces=%d", v)
		}
	}
	return linuxNamespace{}, nil
}

func doAsync(f func() error) chan error {
	done := make(chan error, 1)
	go func() {
		done <- f()
		close(done)
	}()
	return done
}

func terminateChild(child *exec.Cmd, killTimeout time.Duration) func() error {
	return func() error {
		child.Process.Signal(syscall.SIGTERM)
		childDone := doAsync(child.Wait)
		timeExpired := time.After(killTimeout)
		select {
		case <-timeExpired:
			child.Process.Signal(syscall.SIGKILL)
			child.Wait()
			return nil
		case err := <-childDone:
			return err
		}
	}
}

func waitChild(child *exec.Cmd) chan error {
	return doAsync(child.Wait)
}

func getIDMap(ns, host, size int) []syscall.SysProcIDMap {
	return []syscall.SysProcIDMap{{
		ContainerID: ns,
		HostID:      host,
		Size:        size,
	}}
}

func recvSync(pipeR *os.File, syncData []byte) error {
	b := make([]byte, len(syncData))
	if _, err := pipeR.Read(b); err != nil {
		return err
	}
	if !bytes.Equal(b, syncData) {
		return fmt.Errorf("sync mismatch: got %q want %q", string(b), string(syncData))
	}
	return nil
}

func sendSync(pipeW *os.File, syncData []byte) error {
	if _, err := pipeW.Write(syncData); err != nil {
		return err
	}
	return nil
}

func execChildInRemap(ctx context.Context, command ...string) error {
	euid, egid := 0, 0
	fmt.Sscanf(os.Getenv("_CONTAINERS_ROOTLESS_UID"), "%d", &euid)
	fmt.Sscanf(os.Getenv("_CONTAINERS_ROOTLESS_GID"), "%d", &egid)
	sysProcAttr := syscall.SysProcAttr{
		Cloneflags:  uintptr(syscall.CLONE_NEWUSER),
		UidMappings: getIDMap(euid, 0, 1),
		GidMappings: getIDMap(egid, 0, 1),
	}

	return execChild(ctx, &sysProcAttr, command...)
}

func execChild(ctx context.Context, sysProcAttr *syscall.SysProcAttr, command ...string) error {
	childCmd := exec.CommandContext(ctx, command[0], command[1:]...)
	childCmd.Stdin = os.Stdin
	childCmd.Stdout = os.Stdout
	childCmd.Stderr = os.Stderr
	childCmd.SysProcAttr = sysProcAttr
	childCmd.Cancel = terminateChild(childCmd, 5*time.Second)

	err := childCmd.Start()
	if err != nil {
		return err
	}

	childDone := doAsync(childCmd.Wait)

	select {
	case <-ctx.Done():
		return nil
	case err := <-childDone:
		return err
	}
}

func launchNetwork(ctx context.Context, slirpcmd string, pid int, ready chan bool) error {
	defer close(ready)

	parentR, childW, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("%s ready-fd: %s", slirpcmd, err.Error())
	}

	defer childW.Close()
	defer parentR.Close()

	childCmd := exec.CommandContext(ctx, slirpcmd, "-c", "--ready-fd=3", fmt.Sprintf("%d", pid), "tap0")
	childCmd.Stdin = nil
	childCmd.Stdout = nil
	childCmd.Stderr = nil
	childCmd.ExtraFiles = []*os.File{childW}
	childCmd.Cancel = terminateChild(childCmd, 0)

	if err := childCmd.Start(); err != nil {
		return err
	}

	childDone := waitChild(childCmd)
	defer childCmd.Cancel()

	slirpReady := doAsync(func() error {
		b := make([]byte, 256)
		_, err = parentR.Read(b)
		if err != nil {
			return err
		}
		ready <- true
		return nil
	})

	for {
		select {
		case err := <-slirpReady:
			if err != nil {
				return err
			} else {
				slirpReady = nil
			}
		case <-ctx.Done():
			return nil
		case err := <-childDone:
			return err
		}
	}
}

func (config linuxNamespace) Exec(ctx context.Context, command []string, podman bool, sidecar SidecarFunc) error {
	// inherit pipe fd
	childR := os.NewFile(3, "r")
	defer childR.Close()

	// block on start so parent can set up our namespace
	if err := recvSync(childR, []byte("SYNC")); err != nil {
		return fmt.Errorf("Child launch sync (pre-exec): %s", err.Error())
	}

	// setup namespace mounts
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("mount /proc: %s", err.Error())
	}
	if err := syscall.Mount("devpts", "/dev/pts", "devpts", 0, ""); err != nil {
		return fmt.Errorf("mount /dev/pts: %s", err.Error())
	}
	if err := syscall.Mount("tmpfs", "/run/user", "tmpfs", 0, ""); err != nil {
		return fmt.Errorf("mount /run/user: %s", err.Error())
	}
	if podman {
		tmpdir := os.Getenv("TMPDIR")
		if tmpdir != "" {
			if err := syscall.Mount("tmpfs", tmpdir, "tmpfs", 0, ""); err != nil {
				return fmt.Errorf("mount %s: %s", tmpdir, err.Error())
			}
		}

		homedir := os.Getenv("HOME")
		if homedir != "" {
			storagedir := fmt.Sprintf("%s/.local/share/containers/", homedir)
			if err := syscall.Mount("tmpfs", storagedir, "tmpfs", 0, ""); err != nil {
				return fmt.Errorf("mount %s: %s", storagedir, err.Error())
			}
		}
	}

	os.Setenv("_CONTAINERS_USERNS_CONFIGURED", "done")

	cCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var childDone chan error
	if podman {
		// continue as root
		childDone = doAsync(func() error { return execChild(cCtx, nil, command...) })
	} else {
		// launch user command remapped from root to original ID
		childDone = doAsync(func() error { return execChildInRemap(cCtx, command...) })
	}

	var sidecarDone chan error
	if sidecar != nil {
		sidecarDone = doAsync(func() error {
			return sidecar(cCtx)
		})
	} else {
		sidecarDone = doAsync(func() error {
			<-cCtx.Done()
			return nil
		})
	}

	// terminate if: 1. context cancelled; 2. child process terminates; 3. wormhole exits
	select {
	case <-ctx.Done():
		cancel()
		<-sidecarDone
		return ctx.Err()
	case err := <-childDone:
		cancel()
		<-sidecarDone
		return err
	case err := <-sidecarDone:
		return err
	}
}

func (config linuxNamespace) LaunchNS(ctx context.Context, networkHandler string, secondStage []string) error {
	childR, parentW, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("create sync pipe: %s", err.Error())
	}
	defer childR.Close()
	defer parentW.Close()

	cCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	childCmd := exec.CommandContext(cCtx, secondStage[0], secondStage[1:]...)
	childCmd.Stdin = os.Stdin
	childCmd.Stdout = os.Stdout
	childCmd.Stderr = os.Stderr
	childCmd.ExtraFiles = []*os.File{childR}
	childCmd.Env = append(os.Environ(),
		"_CONTAINERS_USERNS_CONFIGURED=init",
		fmt.Sprintf("_CONTAINERS_ROOTLESS_UID=%d", os.Geteuid()),
		fmt.Sprintf("_CONTAINERS_ROOTLESS_GID=%d", os.Getegid()),
	)
	childCmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:  uintptr(syscall.CLONE_NEWNS | syscall.CLONE_NEWNET | syscall.CLONE_NEWPID | syscall.CLONE_NEWUSER | syscall.CLONE_NEWIPC),
		UidMappings: getIDMap(0, os.Geteuid(), 1),
		GidMappings: getIDMap(0, os.Getegid(), 1),
	}
	childCmd.Cancel = terminateChild(childCmd, 5*time.Second)

	if err := childCmd.Start(); err != nil {
		return err
	}

	slirpReady := make(chan bool, 1)
	childDone := waitChild(childCmd)
	slirpDone := doAsync(func() error { return launchNetwork(cCtx, networkHandler, childCmd.Process.Pid, slirpReady) })
	if msg := <-slirpReady; msg != true {
		// recv error from async channel
		return <-slirpDone
	}

	// send sync to child to continue to exec of user command
	if err := sendSync(parentW, []byte("SYNC")); err != nil {
		return fmt.Errorf("Parent launch sync: %s", err.Error())
	}

	// terminate if: 1. context cancelled; 2. child process terminates; 3. slirp exits
	select {
	case <-ctx.Done():
		return nil
	case err := <-childDone:
		return err
	case err := <-slirpDone:
		return err
	}
}
