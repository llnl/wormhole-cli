//go:build !linux

package ns

import (
	"context"
	"fmt"
)

type notImplementedNamespace struct{}

func Initialize() (NamespaceExec, error) {
	return notImplementedNamespace{}, fmt.Errorf("Not implemented")
}

func (c notImplementedNamespace) Exec(ctx context.Context, command []string, podman bool, sidecar SidecarFunc) error {
	return fmt.Errorf("Not implemented")
}

func (c notImplementedNamespace) LaunchNS(ctx context.Context, networkHandler string, secondStage []string) error {
	return fmt.Errorf("Not implemented")
}
