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

package cmd

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/pingcap-incubator/tiup-cluster/pkg/cliutil"
	"github.com/pingcap-incubator/tiup-cluster/pkg/edit"
	"github.com/pingcap-incubator/tiup-cluster/pkg/log"
	"github.com/pingcap-incubator/tiup-cluster/pkg/logger"
	"github.com/pingcap-incubator/tiup-cluster/pkg/meta"
	tiuputils "github.com/pingcap-incubator/tiup/pkg/utils"
	"github.com/pingcap/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func newEditConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit-config <cluster-name>",
		Short: "Edit TiDB cluster config",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return cmd.Help()
			}

			clusterName := args[0]
			if tiuputils.IsNotExist(meta.ClusterPath(clusterName, meta.MetaFileName)) {
				return errors.Errorf("cannot start non-exists cluster %s", clusterName)
			}

			logger.EnableAuditLog()
			metadata, err := meta.ClusterMetadata(clusterName)
			if err != nil {
				return err
			}

			return editTopo(clusterName, metadata)
		},
	}

	return cmd
}

// 1. Write Topology to a temporary file.
// 2. Open file in editro.
// 3. Check and update Topology.
// 4. Save meta file.
func editTopo(clusterName string, metadata *meta.ClusterMeta) error {
	data, err := yaml.Marshal(metadata.Topology)
	if err != nil {
		return errors.AddStack(err)
	}

	file, err := ioutil.TempFile(os.TempDir(), "*")
	if err != nil {
		return errors.AddStack(err)
	}

	name := file.Name()

	_, err = io.Copy(file, bytes.NewReader(data))
	if err != nil {
		return errors.AddStack(err)
	}

	err = file.Close()
	if err != nil {
		return errors.AddStack(err)
	}

	err = edit.OpenFileInEditor(name)
	if err != nil {
		return errors.AddStack(err)
	}

	// Now user finish editing the file.
	newData, err := ioutil.ReadFile(name)
	if err != nil {
		return errors.AddStack(err)
	}

	newTopo := new(meta.TopologySpecification)
	err = yaml.Unmarshal(newData, newTopo)
	if err != nil {
		log.Infof("The file is not in the correct format")
		return errors.AddStack(err)
	}

	if bytes.Equal(data, newData) {
		log.Infof("The file has nothing changed")
		return nil
	}

	edit.ShowDiff(string(data), string(newData), os.Stdout)

	msg := color.HiYellowString("Please check change, do you want to apply the change?")
	msg = msg + "\n[Y]es/[N]o:"
	ans := cliutil.Prompt(msg)
	switch strings.ToLower(ans) {
	case "y", "yes":
		log.Infof("Apply the change...")

		metadata.Topology = newTopo
		err = meta.SaveClusterMeta(clusterName, metadata)
		if err != nil {
			return errors.Annotate(err, "failed to save")
		}

		log.Infof("Apply change success, please use `tiup cluster reload %s` to reload config.", clusterName)

		return nil
	case "n", "no":
		fallthrough
	default:
		log.Infof("Abandom the change")
		return nil
	}
}
