package ec2

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type Controller struct {
	client     *ec2.EC2
	region     string
	instanceId string
	ipAddress  string
	running    bool
	mu         sync.Mutex
}

func MustNewController(
	region string,
	instanceId string,
	instanceAddressType string,
	urlSchema string,
	port int,
) *Controller {
	instanceAddressType = strings.ToLower(strings.TrimSpace(instanceAddressType))
	if instanceAddressType != "private" && instanceAddressType != "public" {
		log.Panicf("invalid instanceAddressType %v\n", instanceAddressType)
	}

	// The session.NewSession function automatically handles AWS credentials using the default credential provider chain.
	// This means that the AWS credentials can be obtained from multiple sources such as environment variables,
	// shared credentials file, or IAM roles assigned to the running instance (in case of EC2).
	// Therefore, there is no need to explicitly specify AWS credentials in this code.
	sess, err := session.NewSession(&aws.Config{Region: &region})
	if err != nil {
		log.Panicln(fmt.Errorf("failed to create ec2 controller: %w", err))
	}

	controller := &Controller{region: region, instanceId: instanceId, client: ec2.New(sess)}
	if err := controller.updateState(instanceAddressType, urlSchema, port); err != nil {
		log.Panicln(fmt.Errorf("failed to update ec2 controller: %w", err))
	}
	return controller
}

func (c *Controller) updateState(instanceAddressType string, urlSchema string, port int) error {
	instance, err := c.findInstance()
	if err != nil {
		return fmt.Errorf("failed to read prover instance info (instanceId: %s): %w", c.instanceId, err)
	}

	c.running = aws.StringValue(instance.State.Name) == ec2.InstanceStateNameRunning || aws.StringValue(instance.State.Name) == ec2.InstanceStateNamePending
	c.ipAddress, err = findAddress(instance, instanceAddressType, urlSchema, port)
	if err != nil {
		return fmt.Errorf("failed to find prover instance address (instanceId: %s): %w", c.instanceId, err)
	}
	log.Printf("prover instance ip address %s\n", c.ipAddress)
	return nil
}

func (c *Controller) findInstance() (*ec2.Instance, error) {
	output, err := c.client.DescribeInstances(&ec2.DescribeInstancesInput{InstanceIds: c.instanceIds()})
	if err != nil {
		return nil, err
	}
	if len(output.Reservations) == 0 || len(output.Reservations[0].Instances) == 0 {
		return nil, errors.New("prover instance not found")
	}
	return output.Reservations[0].Instances[0], nil
}

func findAddress(instance *ec2.Instance, instanceAddressType string, schema string, port int) (string, error) {
	for _, networkInterface := range instance.NetworkInterfaces {
		if networkInterface != nil {
			for _, ipAddress := range networkInterface.PrivateIpAddresses {
				if ipAddress != nil {
					var address string
					switch instanceAddressType {
					case "private":
						address = aws.StringValue(ipAddress.PrivateIpAddress)
					case "public":
						if ipAddress.Association != nil {
							address = aws.StringValue(ipAddress.Association.PublicIp)
						}
					}
					if len(address) != 0 {
						return schema + "://" + address + ":" + strconv.Itoa(port), nil
					}
				}
			}
		}
	}
	return "", errors.New("failed to retrieve prover instance address")
}

func (c *Controller) StartIfNotRunning() error {
	log.Printf("starting prover instance (instanceId: %s)...", c.instanceId)
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		log.Println("prover instance is already running")
		return nil
	}

	for {
		instance, err := c.findInstance()
		if err != nil {
			return fmt.Errorf("failed to read prover instance info (instanceId: %s): %w", c.instanceId, err)
		}
		instanceState := aws.StringValue(instance.State.Name)
		if instanceState == ec2.InstanceStateNameStopped {
			break
		}
		log.Printf("prover instance state is not stopped: %s", instanceState)
		time.Sleep(1 * time.Second)
	}

	_, err := c.client.StartInstances(&ec2.StartInstancesInput{InstanceIds: c.instanceIds()})
	if err != nil {
		return fmt.Errorf("failed to start prover instance (instanceId: %s): %w", c.instanceId, err)
	}
	log.Printf("started prover instance (instanceId: %s)", c.instanceId)
	c.running = true
	return nil
}

func (c *Controller) StopIfRunning() {
	log.Printf("stopping prover instance (instanceId: %s)...", c.instanceId)
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		log.Println("prover instance is not running")
		return
	}

	_, err := c.client.StopInstances(&ec2.StopInstancesInput{InstanceIds: c.instanceIds()})
	if err != nil {
		log.Println(fmt.Errorf("failed to stop prover instance (instanceId: %s): %w", c.instanceId, err))
		return
	}
	log.Printf("stopped prover instance (instanceId: %s)", c.instanceId)
	c.running = false
}

func (c *Controller) IpAddress() string      { return c.ipAddress }
func (c *Controller) instanceIds() []*string { return []*string{&c.instanceId} }
func (c *Controller) Running() bool          { return c.running }
