
package main

import (
	"os"
	"fmt"
	"os/exec"
	"encoding/json"
	"k8s.io/client-go/rest"
	"time"

	"k8s.io/client-go/kubernetes"
)

type AuthenticationResponse struct {
	Kind string
	ApiVersion string
	Status struct {
		Token string
	}
}

func getToken(profile string, cluster_name string) string {
	app := "aws-iam-authenticator"

	cmd := exec.Command(app, "token", "-i", cluster_name)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "AWS_PROFILE=" + profile)
	stdout, err := cmd.Output()

	if (err != nil) {
		 fmt.Fprintln(os.Stderr, err.Error())
		 fmt.Fprintln(os.Stderr, "Unable to obtain authentication token from aws-iam-authenticator. Is this app installed? see https://docs.aws.amazon.com/eks/latest/userguide/getting-started.html")
		 return ""
	}

	res := AuthenticationResponse{}
	if err := json.Unmarshal(stdout, &res); err != nil {
		panic(err)
	}

	return res.Status.Token

}

func getClient() (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error
  config, err = rest.InClusterConfig()

	if err == nil {
		return kubernetes.NewForConfig(config)
	}

	fmt.Fprintln(os.Stderr, "Could not use in-cluster configuration, trying with aws-iam-authenticator")
	fmt.Fprintln(os.Stderr, err.Error())

	config = &rest.Config{}
	config.Host = "http://127.0.0.1:8001"
	config.BearerToken = getToken("analytics", "eks-analytics-prod")

	go func() {
		for {
			config.BearerToken = getToken("analytics", "eks-analytics-prod")
			time.Sleep(1 * time.Minute)
		}
	}()

/*

	// in cluster access
	} else {
		logrus.Info("Using out of cluster config")
		config, err = clientcmd.BuildConfigFromFlags("", pathToCfg)
	}

	if err != nil {
		return nil, err
	}
	*/
	return kubernetes.NewForConfig(config)
}
