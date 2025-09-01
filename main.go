package main

import (
	"github.com/fujitsu/docker-machine-driver-fsas/pkg/drivers/fsas"
	"github.com/rancher/machine/libmachine/drivers/plugin"
)

func main() {
	plugin.RegisterDriver(fsas.NewDriver())
}
