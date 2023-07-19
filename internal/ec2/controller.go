package ec2

import (
	"fmt"
	"log"
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

func MustNewController(region string, instanceId string) *Controller {
	sess, err := session.NewSession(&aws.Config{Region: &region})
	if err != nil {
		log.Panicln(fmt.Errorf("failed to create ec2 controller: %w", err))
	}
	instance := &Controller{region: region, instanceId: instanceId, client: ec2.New(sess)}
	if err := instance.updateState(); err != nil {
		log.Panicln(fmt.Errorf("failed to update ec2 controller: %w", err))
	}
	return instance
}

func (c *Controller) IpAddress() string { return c.ipAddress }

func (c *Controller) updateState() error {
	instance, err := c.findInstance()
	if err == nil {
		c.running = aws.StringValue(instance.State.Name) == "running" || aws.StringValue(instance.State.Name) == "pending"
		for _, networkInterface := range instance.NetworkInterfaces {
			for _, ipAddress := range networkInterface.PrivateIpAddresses {
				c.ipAddress = aws.StringValue(ipAddress.PrivateIpAddress) // private
				// aws.StringValue(ipAddress.Association.PublicIp) public
			}
		}
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
