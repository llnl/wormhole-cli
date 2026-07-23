package ns

import "context"

type SidecarFunc func(context.Context) error

type NamespaceExec interface {
	Exec(ctx context.Context, command []string, podman bool, sidecar SidecarFunc) error
	LaunchNS(ctx context.Context, networkHandler string, secondStage []string) error
}
