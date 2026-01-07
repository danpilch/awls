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
			// Skip terminated instances or instances with nil state
			if i.State == nil || string(i.State.Name) == "terminated" {
				continue
			}
			// Only add instances with valid private IP addresses
			if i.PrivateIpAddress != nil {
				privateIps = append(privateIps, *i.PrivateIpAddress)
			}
		}
	}
	return privateIps
}

func toAny(s []string) []any {
	result := make([]any, len(s))
	for i, v := range s {
		result[i] = v
	}
	return result
}

func buildTableData(result *ec2.DescribeInstancesOutput) ([][]string, []string) {
	var tbl = [][]string{}
	var tblHeaders = []string{"Name", "PrivateIp", "State", "AZ", "InstanceId", "InstanceType", "LaunchTime"}

	for _, reservation := range result.Reservations {
		for _, i := range reservation.Instances {
			// Skip terminated instances or instances with nil state
			if i.State == nil || string(i.State.Name) == "terminated" {
				continue
			}

			// Extract name tag with nil checks
			var nameTag string
			for _, t := range i.Tags {
				if t.Key != nil && *t.Key == "Name" && t.Value != nil {
					nameTag = *t.Value
					break
				}
			}

			// Get instance fields with safe defaults for nil pointers
			privateIp := "N/A"
			if i.PrivateIpAddress != nil {
				privateIp = *i.PrivateIpAddress
			}

			state := "unknown"
			if i.State != nil {
				state = string(i.State.Name)
			}

			az := "N/A"
			if i.Placement != nil && i.Placement.AvailabilityZone != nil {
				az = *i.Placement.AvailabilityZone
			}

			instanceId := "N/A"
			if i.InstanceId != nil {
				instanceId = *i.InstanceId
			}

			instanceType := "N/A"
			if i.InstanceType != "" {
				instanceType = string(i.InstanceType)
			}

			launchTime := "N/A"
			if i.LaunchTime != nil {
				launchTime = i.LaunchTime.Format("2006-01-02 15:04:05")
			}

			tbl = append(tbl, []string{
				nameTag,
				privateIp,
				state,
				az,
				instanceId,
				instanceType,
				launchTime,
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

	// Create config
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load AWS configuration: %v\n", err)
		fmt.Fprintln(os.Stderr, "\nPlease ensure:")
		fmt.Fprintln(os.Stderr, "  - AWS credentials are configured (via ~/.aws/credentials or environment variables)")
		fmt.Fprintln(os.Stderr, "  - AWS region is set (via AWS_REGION environment variable or ~/.aws/config)")
		fmt.Fprintln(os.Stderr, "  - IAM permissions are properly configured")
		os.Exit(1)
	}

	// Create client
	ec2Client := ec2.NewFromConfig(cfg)

	// Generate an EC2 search filter
	filter := buildSearchFilter(cli.FilterType)

	// find relevant resources from aws api
	result, err := ec2Client.DescribeInstances(context.TODO(), filter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to describe EC2 instances: %v\n", err)
		fmt.Fprintln(os.Stderr, "\nPossible causes:")
		fmt.Fprintln(os.Stderr, "  - Insufficient IAM permissions (ec2:DescribeInstances required)")
		fmt.Fprintln(os.Stderr, "  - Invalid filter parameters")
		fmt.Fprintln(os.Stderr, "  - Network connectivity issues")
		fmt.Fprintln(os.Stderr, "  - AWS API throttling or rate limiting")
		os.Exit(1)
	}

	// return if no results found
	if result == nil || len(result.Reservations) < 1 {
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
			fmt.Println(strings.Join(buildPrivateIpData(result), cli.Delimiter))
		}
	} else {
		tableData, tableHeaders := buildTableData(result)
		table := tablewriter.NewWriter(os.Stdout)
		table.Header(toAny(tableHeaders)...)
		table.Bulk(tableData)
		table.Render()
	}
}
