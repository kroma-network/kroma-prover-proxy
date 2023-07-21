package ec2

import (
	"fmt"
	"log"
	"strings"
	"sync"

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
) *Controller {
	instanceAddressType = strings.ToLower(strings.TrimSpace(instanceAddressType))
	if instanceAddressType != "private" && instanceAddressType != "public" {
		log.Panicf("invalid instanceAddressType %v\n", instanceAddressType)
	}
	sess, err := session.NewSession(&aws.Config{Region: &region})
	if err != nil {
		log.Panicln(fmt.Errorf("failed to create ec2 controller: %w", err))
	}
	instance := &Controller{region: region, instanceId: instanceId, client: ec2.New(sess)}
	if err := instance.updateState(instanceAddressType); err != nil {
		log.Panicln(fmt.Errorf("failed to update ec2 controller: %w", err))
	}
	return instance
}

func (c *Controller) IpAddress() string { return c.ipAddress }

func (c *Controller) updateState(instanceAddressType string) error {
	instance, err := c.findInstance()
	if err == nil {
		c.running = aws.StringValue(instance.State.Name) == "running" || aws.StringValue(instance.State.Name) == "pending"
		c.ipAddress = findAddress(instance, instanceAddressType)

	}
	return err
}

func (c *Controller) findInstance() (*ec2.Instance, error) {
	output, err := c.client.DescribeInstances(&ec2.DescribeInstancesInput{InstanceIds: c.instanceIds()})
	if err != nil || len(output.Reservations) == 0 || len(output.Reservations[0].Instances) == 0 {
		return nil, err
	}
	return output.Reservations[0].Instances[0], nil
}

func findAddress(instance *ec2.Instance, instanceAddressType string) (address string) {
	for _, networkInterface := range instance.NetworkInterfaces {
		if networkInterface != nil {
			for _, ipAddress := range networkInterface.PrivateIpAddresses {
				if ipAddress != nil {
					switch instanceAddressType {
					case "private":
						address = aws.StringValue(ipAddress.PrivateIpAddress)
					case "public":
						if ipAddress.Association != nil {
							address = aws.StringValue(ipAddress.Association.PublicIp)
						}
					}
					if len(address) != 0 {
						return
					}
				}
			}
		}
	}
	return
}

func (c *Controller) StartIfNotRunning() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running {
		return nil
	}
	_, err := c.client.StartInstances(&ec2.StartInstancesInput{InstanceIds: c.instanceIds()})
	if err != nil {
		log.Println(fmt.Errorf("failed to start ec2 instance %s: %w", c.instanceId, err))
		return err
	}
	c.running = true
	return nil
}

func (c *Controller) StopIfRunning() {
	if c.running {
		c.mu.Lock()
		defer c.mu.Unlock()
		_, err := c.client.StopInstances(&ec2.StopInstancesInput{InstanceIds: c.instanceIds()})
		if err == nil {
			c.running = false
		} else {
			log.Println(fmt.Errorf("failed to stop ec2 instance %s: %w", c.instanceId, err))
		}
	}
}

func (c *Controller) instanceIds() []*string { return []*string{&c.instanceId} }
