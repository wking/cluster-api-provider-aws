// Copyright © 2018 The Kubernetes Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cluster

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/clientcmd"
	providerv1 "sigs.k8s.io/cluster-api-provider-aws/pkg/apis/awsprovider/v1alpha1"
	service "sigs.k8s.io/cluster-api-provider-aws/pkg/cloud/aws/services"
	"sigs.k8s.io/cluster-api-provider-aws/pkg/cloud/aws/services/certificates"
	ec2svc "sigs.k8s.io/cluster-api-provider-aws/pkg/cloud/aws/services/ec2"
	elbsvc "sigs.k8s.io/cluster-api-provider-aws/pkg/cloud/aws/services/elb"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	client "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/typed/cluster/v1alpha1"
	controllerError "sigs.k8s.io/cluster-api/pkg/controller/error"
)

// Actuator is responsible for performing cluster reconciliation
type Actuator struct {
	clustersGetter client.ClustersGetter
	servicesGetter service.Getter
}

// ActuatorParams holds parameter information for Actuator
type ActuatorParams struct {
	ClustersGetter client.ClustersGetter
	ServicesGetter service.Getter
}

// NewActuator creates a new Actuator
func NewActuator(params ActuatorParams) *Actuator {
	res := &Actuator{
		clustersGetter: params.ClustersGetter,
		servicesGetter: params.ServicesGetter,
	}

	if res.servicesGetter == nil {
		res.servicesGetter = new(defaultServicesGetter)
	}

	return res
}

// Reconcile reconciles a cluster and is invoked by the Cluster Controller
func (a *Actuator) Reconcile(cluster *clusterv1.Cluster) (reterr error) {
	glog.Infof("Reconciling cluster %v.", cluster.Name)

	// Load provider config.
	config, err := providerv1.ClusterConfigFromProviderConfig(cluster.Spec.ProviderConfig)
	if err != nil {
		return errors.Errorf("failed to load cluster provider config: %v", err)
	}

	// Load provider status.
	status, err := providerv1.ClusterStatusFromProviderStatus(cluster.Status.ProviderStatus)
	if err != nil {
		return errors.Errorf("failed to load cluster provider status: %v", err)
	}

	defer func() {
		if err := a.storeClusterStatus(cluster, status); err != nil {
			glog.Errorf("failed to store provider status for cluster %q in namespace %q: %v", cluster.Name, cluster.Namespace, err)
		}
	}()

	// Store some config parameters in the status.
	status.Region = config.Region

	if len(status.CACertificate) == 0 {
		caCert, caKey, err := certificates.NewCertificateAuthority()
		if err != nil {
			return errors.Wrap(err, "Failed to generate a CA for the control plane")
		}
		status.CACertificate = certificates.EncodeCertPEM(caCert)
		status.CAPrivateKey = certificates.EncodePrivateKeyPEM(caKey)
	}

	// Create new aws session.
	sess := a.servicesGetter.Session(config)

	// Load ec2 client.
	ec2 := a.servicesGetter.EC2(sess)

	if err := ec2.ReconcileNetwork(cluster.Name, &status.Network); err != nil {
		return errors.Errorf("unable to reconcile network: %v", err)
	}

	if err := ec2.ReconcileBastion(cluster.Name, config.SSHKeyName, status); err != nil {
		return errors.Errorf("unable to reconcile network: %v", err)
	}

	// Load elb client.
	elb := a.servicesGetter.ELB(sess)

	if err := elb.ReconcileLoadbalancers(cluster.Name, &status.Network); err != nil {
		return errors.Errorf("unable to reconcile load balancers: %v", err)
	}

	return nil
}

// Delete deletes a cluster and is invoked by the Cluster Controller
func (a *Actuator) Delete(cluster *clusterv1.Cluster) error {
	glog.Infof("Deleting cluster %v.", cluster.Name)

	// Load provider config.
	config, err := providerv1.ClusterConfigFromProviderConfig(cluster.Spec.ProviderConfig)
	if err != nil {
		return errors.Errorf("failed to load cluster provider config: %v", err)
	}

	// Load provider status.
	status, err := providerv1.ClusterStatusFromProviderStatus(cluster.Status.ProviderStatus)
	if err != nil {
		return errors.Errorf("failed to load cluster provider status: %v", err)
	}

	// Store some config parameters in the status.
	status.Region = config.Region

	// Create new aws session.
	sess := a.servicesGetter.Session(config)

	// Load ec2 client.
	ec2 := a.servicesGetter.EC2(sess)

	// Load elb client.
	elb := a.servicesGetter.ELB(sess)

	if err := elb.DeleteLoadbalancers(cluster.Name, &status.Network); err != nil {
		return errors.Errorf("unable to delete load balancers: %v", err)
	}

	if err := ec2.DeleteBastion(cluster.Name, status); err != nil {
		return errors.Errorf("unable to delete bastion: %v", err)
	}

	if err := ec2.DeleteNetwork(cluster.Name, &status.Network); err != nil {
		glog.Errorf("Error deleting cluster %v: %v.", cluster.Name, err)
		return &controllerError.RequeueAfterError{
			RequeueAfter: 5 * 1000 * 1000 * 1000,
		}
	}

	return nil
}

// GetIP returns the IP of a machine, but this is going away.
func (a *Actuator) GetIP(cluster *clusterv1.Cluster, _ *clusterv1.Machine) (string, error) {
	if cluster.Status.ProviderStatus != nil {

		// Load provider status.
		status, err := providerv1.ClusterStatusFromProviderStatus(cluster.Status.ProviderStatus)
		if err != nil {
			return "", errors.Errorf("failed to load cluster provider status: %v", err)
		}

		if status.Network.APIServerELB.DNSName != "" {
			return status.Network.APIServerELB.DNSName, nil
		}
	}

	// Load provider config.
	config, err := providerv1.ClusterConfigFromProviderConfig(cluster.Spec.ProviderConfig)
	if err != nil {
		return "", errors.Errorf("failed to load cluster provider config: %v", err)
	}

	sess := a.servicesGetter.Session(config)
	elb := a.servicesGetter.ELB(sess)
	return elb.GetAPIServerDNSName(cluster.Name)
}

// GetKubeConfig returns the kubeconfig after the bootstrap process is complete.
func (a *Actuator) GetKubeConfig(cluster *clusterv1.Cluster, machine *clusterv1.Machine) (string, error) {

	// Load provider status.
	status, err := providerv1.ClusterStatusFromProviderStatus(cluster.Status.ProviderStatus)
	if err != nil {
		return "", errors.Errorf("failed to load cluster provider status: %v", err)
	}

	cert, err := certificates.DecodeCertPEM(status.CACertificate)
	if err != nil {
		return "", errors.Wrap(err, "failed to decode CA Cert")
	} else if cert == nil {
		return "", errors.New("certificate not found in status")
	}

	key, err := certificates.DecodePrivateKeyPEM(status.CAPrivateKey)
	if err != nil {
		return "", errors.Wrap(err, "failed to decode private key")
	} else if key == nil {
		return "", errors.New("key not found in status")
	}

	dnsName, err := a.GetIP(cluster, machine)
	if err != nil {
		return "", errors.Wrap(err, "failed to get DNS address")
	}

	server := fmt.Sprintf("https://%s:6443", dnsName)

	cfg, err := certificates.NewKubeconfig(server, cert, key)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate a kubeconfig")
	}

	yaml, err := clientcmd.Write(*cfg)
	if err != nil {
		return "", errors.Wrap(err, "failed to serialize config to yaml")
	}

	return string(yaml), nil
}

func (a *Actuator) storeClusterStatus(cluster *clusterv1.Cluster, status *providerv1.AWSClusterProviderStatus) error {
	clusterClient := a.clustersGetter.Clusters(cluster.Namespace)

	ext, err := providerv1.EncodeClusterStatus(status)
	if err != nil {
		return fmt.Errorf("failed to update cluster status for cluster %q in namespace %q: %v", cluster.Name, cluster.Namespace, err)
	}

	cluster.Status.ProviderStatus = ext

	if _, err := clusterClient.UpdateStatus(cluster); err != nil {
		return fmt.Errorf("failed to update cluster status for cluster %q in namespace %q: %v", cluster.Name, cluster.Namespace, err)
	}

	return nil
}

type defaultServicesGetter struct{}

func (d *defaultServicesGetter) Session(clusterConfig *providerv1.AWSClusterProviderConfig) *session.Session {
	return session.Must(session.NewSession(aws.NewConfig().WithRegion(clusterConfig.Region)))
}

func (d *defaultServicesGetter) EC2(session *session.Session) service.EC2Interface {
	return ec2svc.NewService(ec2.New(session))
}

func (d *defaultServicesGetter) ELB(session *session.Session) service.ELBInterface {
	return elbsvc.NewService(elb.New(session))
}
