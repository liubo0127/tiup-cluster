// Copyright 2020 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package task

import (
	"fmt"
	"os"

	"github.com/pingcap-incubator/tiup-cluster/pkg/meta"
)

// ScaleConfig is used to copy all configurations to the target directory of path
type ScaleConfig struct {
	name       string
	instance   meta.Instance
	base       *meta.TopologySpecification
	deployUser string
	paths      meta.DirPaths
}

// Execute implements the Task interface
func (c *ScaleConfig) Execute(ctx *Context) error {
	// Copy to remote server
	exec, found := ctx.GetExecutor(c.instance.GetHost())
	if !found {
		return ErrNoExecutor
	}

	c.paths.Cache = meta.ClusterPath(c.name, "config")
	if err := os.MkdirAll(c.paths.Cache, 0755); err != nil {
		return err
	}

	return c.instance.ScaleConfig(exec, c.base, c.name, c.deployUser, c.paths)
}

// Rollback implements the Task interface
func (c *ScaleConfig) Rollback(ctx *Context) error {
	return ErrUnsupportedRollback
}

// String implements the fmt.Stringer interface
func (c *ScaleConfig) String() string {
	return fmt.Sprintf("ScaleConfig: cluster=%s, user=%s, host=%s, service=%s, %s",
		c.name, c.deployUser, c.instance.GetHost(), c.instance.ServiceName(), c.paths)
}
