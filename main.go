package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
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
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name: aws.String(filterName),
				Values: []*string{
					aws.String("*" + cli.Search + "*"),
				},
			},
		},
	}
	return filter
}

func buildPrivateIpData(result *ec2.DescribeInstancesOutput) []string {
	var privateIps = []string{}
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			privateIps = append(privateIps, string(*instance.PrivateIpAddress))
		}
	}
	return privateIps
}

func buildTableData(result *ec2.DescribeInstancesOutput) ([][]string, []string) {
	var tbl = [][]string{}
	var tblHeaders = []string{"Name", "PrivateIp", "State", "AZ", "InstanceId", "InstanceType", "LaunchTime"}

	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			var nameTag string
			for _, t := range instance.Tags {
				if *t.Key == "Name" {
					nameTag = *t.Value
					break
				}
			}

			tbl = append(tbl, []string{
				string(nameTag),
				string(*instance.PrivateIpAddress),
				string(*instance.State.Name),
				string(*instance.Placement.AvailabilityZone),
				string(*instance.InstanceId),
				string(*instance.InstanceType),
				string(instance.LaunchTime.Format("2006-01-02 15:04:05")),
			})
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

	// Load session from shared config
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	// Create new EC2 client
	svc := ec2.New(sess)

	// Generate an EC2 search filter
	filter := buildSearchFilter(cli.FilterType)

	// find relevant resources from aws api
	result, err := svc.DescribeInstances(filter)

	// check results and error if something went wrong
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}

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
