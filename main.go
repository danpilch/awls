package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/olekukonko/tablewriter"
)

var Version = "development"

var cli struct {
	Search     string `required arg help:"EC2 Instance Name search term"`
	IpOnly     bool   `short:"i" help:"Output only Private IPs"`
	NewLine    bool   `short:"n" help:"Output each IP on a new line" default:"false"`
	Delimiter  string `short:"d" help:"IP delimiter" default:" "`
	FilterType string `short:"f" help:"EC2 Filter Type (https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeInstances.html)" default:"tag:Name"`
	Version    bool   `short:"v" help:"Print version"`
}

func buildSearchFilter(filterName string) *ec2.DescribeInstancesInput {
	// Define search params - only basic pattern matching supported right now
	filter := &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name: aws.String(filterName),
				Values: []string{
					"*" + cli.Search + "*",
				},
			},
		},
	}
	return filter
}

func buildPrivateIpData(result *ec2.DescribeInstancesOutput) []string {
	var privateIps = []string{}
	for _, reservation := range result.Reservations {
		for _, i := range reservation.Instances {
			if i.PrivateIpAddress != nil {
				privateIps = append(privateIps, string(*i.PrivateIpAddress))
			}
		}
	}
	return privateIps
}

func buildTableData(result *ec2.DescribeInstancesOutput) ([][]string, []string) {
	var tbl = [][]string{}
	var tblHeaders = []string{"Name", "PrivateIp", "State", "AZ", "InstanceId", "InstanceType", "LaunchTime"}

	for _, reservation := range result.Reservations {
		for _, i := range reservation.Instances {
			if string(i.State.Name) != "terminated" {
				var nameTag string
				for _, t := range i.Tags {
					if *t.Key == "Name" {
						nameTag = *t.Value
						break
					}
				}
	
				tbl = append(tbl, []string{
					nameTag,
					*i.PrivateIpAddress,
					string(i.State.Name),
					*i.Placement.AvailabilityZone,
					*i.InstanceId,
					string(i.InstanceType),
					string(i.LaunchTime.Format("2006-01-02 15:04:05")),
				})
			}
		}
	}
	return tbl, tblHeaders
}

func main() {
	// Parse cli args
	kong.Parse(&cli)

	// Version check
	if cli.Version {
		fmt.Println(Version)
		return
	}

	// Create config
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic(err)
	}

	// Create client
	ec2Client := ec2.NewFromConfig(cfg)

	// Generate an EC2 search filter
	filter := buildSearchFilter(cli.FilterType)

	// find relevant resources from aws api
	result, err := ec2Client.DescribeInstances(context.TODO(), filter)

	// return if no results found
	if len(result.Reservations) < 1 {
		fmt.Println("no matching instances found")
		return
	}

	// output a list of ips
	if cli.IpOnly {
		if cli.NewLine {
			for _, ip := range buildPrivateIpData(result) {
				fmt.Println(ip)
			}
		} else {
			fmt.Println(strings.Join(buildPrivateIpData(result)[:], cli.Delimiter))
		}
	} else {
		tableData, tableHeaders := buildTableData(result)
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader(tableHeaders)
		table.SetAutoFormatHeaders(true)
		table.AppendBulk(tableData)
		table.Render()
	}
}
