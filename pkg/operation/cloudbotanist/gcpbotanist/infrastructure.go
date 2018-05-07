// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gcpbotanist

import (
	"fmt"

	"github.com/gardener/gardener/pkg/operation/common"
	"github.com/gardener/gardener/pkg/operation/terraformer"
	"github.com/gardener/gardener/pkg/utils"
)

// DeployInfrastructure kicks off a Terraform job which deploys the infrastructure.
func (b *GCPBotanist) DeployInfrastructure() error {
	var (
		vpcName   = "${google_compute_network.network.name}"
		createVPC = true
	)

	// check if we should use an existing VPC or create a new one
	if b.VPCName != "" {
		vpcName = b.VPCName
		createVPC = false
	}

	return terraformer.
		NewFromOperation(b.Operation, common.TerraformerPurposeInfra).
		SetVariablesEnvironment(b.generateTerraformInfraVariablesEnvironment()).
		DefineConfig("gcp-infra", b.generateTerraformInfraConfig(createVPC, vpcName)).
		Apply()
}

// DestroyInfrastructure kicks off a Terraform job which destroys the infrastructure.
func (b *GCPBotanist) DestroyInfrastructure() error {
	return terraformer.
		NewFromOperation(b.Operation, common.TerraformerPurposeInfra).
		SetVariablesEnvironment(b.generateTerraformInfraVariablesEnvironment()).
		Destroy()
}

// generateTerraformInfraVariablesEnvironment generates the environment containing the credentials which
// are required to validate/apply/destroy the Terraform configuration. These environment must contain
// Terraform variables which are prefixed with TF_VAR_.
func (b *GCPBotanist) generateTerraformInfraVariablesEnvironment() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":  "TF_VAR_SERVICEACCOUNT",
			"value": b.MinifiedServiceAccount,
		},
	}
}

// generateTerraformInfraConfig creates the Terraform variables and the Terraform config (for the infrastructure)
// and returns them (these values will be stored as a ConfigMap and a Secret in the Garden cluster.
func (b *GCPBotanist) generateTerraformInfraConfig(createVPC bool, vpcName string) map[string]interface{} {
	return map[string]interface{}{
		"google": map[string]interface{}{
			"region":  b.Shoot.Info.Spec.Cloud.Region,
			"project": b.Project,
		},
		"create": map[string]interface{}{
			"vpc": createVPC,
		},
		"vpc": map[string]interface{}{
			"name": vpcName,
		},
		"clusterName": b.Shoot.SeedNamespace,
		"networks": map[string]interface{}{
			"pods":     b.Shoot.GetPodNetwork(),
			"services": b.Shoot.GetServiceNetwork(),
			"worker":   b.Shoot.Info.Spec.Cloud.GCP.Networks.Workers[0],
		},
	}
}

// DeployBackupInfrastructure kicks off a Terraform job which deploys the infrastructure resources for backup.
func (b *GCPBotanist) DeployBackupInfrastructure() error {
	image := ""
	o := b.Operation
	if img, _ := o.ImageVector.FindImage("terraformer", o.K8sSeedClient.Version()); img != nil {
		image = img.String()
	}
	return terraformer.
		New(o.Logger, o.K8sSeedClient, common.TerraformerPurposeBackup, o.BackupInfrastructure.Name, common.GenerateBackupNamespaceName(o.BackupInfrastructure.Name), image).
		SetVariablesEnvironment(b.generateTerraformBackupVariablesEnvironment()).
		DefineConfig("gcp-backup", b.generateTerraformBackupConfig()).
		Apply()
}

// DestroyBackupInfrastructure kicks off a Terraform job which destroys the infrastructure for backup.
func (b *GCPBotanist) DestroyBackupInfrastructure() error {
	image := ""
	o := b.Operation
	if img, _ := o.ImageVector.FindImage("terraformer", o.K8sSeedClient.Version()); img != nil {
		image = img.String()
	}
	return terraformer.
		New(o.Logger, o.K8sSeedClient, common.TerraformerPurposeBackup, o.BackupInfrastructure.Name, common.GenerateBackupNamespaceName(o.BackupInfrastructure.Name), image).
		SetVariablesEnvironment(b.generateTerraformBackupVariablesEnvironment()).
		Destroy()
}

// generateTerraformBackupVariablesEnvironment generates the environment containing the credentials which
// are required to validate/apply/destroy the Terraform configuration. These environment must contain
// Terraform variables which are prefixed with TF_VAR_.
func (b *GCPBotanist) generateTerraformBackupVariablesEnvironment() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":  "TF_VAR_SERVICEACCOUNT",
			"value": b.MinifiedServiceAccount,
		},
	}
}

// generateTerraformBackupConfig creates the Terraform variables and the Terraform config (for the backup)
// and returns them.
func (b *GCPBotanist) generateTerraformBackupConfig() map[string]interface{} {
	return map[string]interface{}{
		"google": map[string]interface{}{
			"region":  b.Seed.Info.Spec.Cloud.Region,
			"project": b.Project,
		},
		"bucket": map[string]interface{}{
			"name": fmt.Sprintf("%s-%s", b.Shoot.SeedNamespace, utils.ComputeSHA1Hex([]byte(b.Shoot.Info.Status.UID))[:5]),
		},
		"clusterName": b.Shoot.SeedNamespace,
	}
}
